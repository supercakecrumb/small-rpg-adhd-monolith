package store

import (
	"database/sql"
	"fmt"
	"small-rpg-adhd-monolith/internal/core"
)

// CreateTransaction creates a new transaction
func (s *Store) CreateTransaction(userID, groupID int64, amount int, sourceType core.SourceType, sourceID *int64) (*core.Transaction, error) {
	var result sql.Result
	var err error

	if sourceID != nil {
		result, err = s.DB.Exec(
			"INSERT INTO transactions (user_id, group_id, amount, source_type, source_id) VALUES (?, ?, ?, ?, ?)",
			userID, groupID, amount, string(sourceType), *sourceID,
		)
	} else {
		result, err = s.DB.Exec(
			"INSERT INTO transactions (user_id, group_id, amount, source_type) VALUES (?, ?, ?, ?)",
			userID, groupID, amount, string(sourceType),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetTransactionByID(id)
}

// GetTransactionByID retrieves a transaction by ID
func (s *Store) GetTransactionByID(id int64) (*core.Transaction, error) {
	tx := &core.Transaction{}
	var sourceType string
	var sourceID sql.NullInt64

	err := s.DB.QueryRow(
		"SELECT id, user_id, group_id, amount, source_type, source_id, created_at FROM transactions WHERE id = ?",
		id,
	).Scan(&tx.ID, &tx.UserID, &tx.GroupID, &tx.Amount, &sourceType, &sourceID, &tx.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	tx.SourceType = core.SourceType(sourceType)
	if sourceID.Valid {
		tx.SourceID = &sourceID.Int64
	}

	return tx, nil
}

// GetTransactionsByUserAndGroup retrieves all transactions for a user in a group
func (s *Store) GetTransactionsByUserAndGroup(userID, groupID int64) ([]*core.Transaction, error) {
	rows, err := s.DB.Query(
		"SELECT id, user_id, group_id, amount, source_type, source_id, created_at FROM transactions WHERE user_id = ? AND group_id = ?",
		userID, groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*core.Transaction
	for rows.Next() {
		tx := &core.Transaction{}
		var sourceType string
		var sourceID sql.NullInt64

		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.GroupID, &tx.Amount, &sourceType, &sourceID, &tx.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		tx.SourceType = core.SourceType(sourceType)
		if sourceID.Valid {
			tx.SourceID = &sourceID.Int64
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// GetBalance calculates the total balance for a user in a group
func (s *Store) GetBalance(userID, groupID int64) (int, error) {
	var balance sql.NullInt64

	err := s.DB.QueryRow(
		"SELECT SUM(amount) FROM transactions WHERE user_id = ? AND group_id = ?",
		userID, groupID,
	).Scan(&balance)

	if err != nil {
		return 0, fmt.Errorf("failed to calculate balance: %w", err)
	}

	if !balance.Valid {
		return 0, nil
	}

	return int(balance.Int64), nil
}
