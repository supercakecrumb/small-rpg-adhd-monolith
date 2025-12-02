package store

import (
	"database/sql"
	"fmt"
	"small-rpg-adhd-monolith/internal/core"
)

// CreateTransaction creates a new transaction
func (s *Store) CreateTransaction(userID, groupID int64, amount int, sourceType core.SourceType, sourceID *int64, quantity int) (*core.Transaction, error) {
	var result sql.Result
	var err error

	if sourceID != nil {
		result, err = s.DB.Exec(
			"INSERT INTO transactions (user_id, group_id, amount, source_type, source_id, quantity) VALUES (?, ?, ?, ?, ?, ?)",
			userID, groupID, amount, string(sourceType), *sourceID, quantity,
		)
	} else {
		result, err = s.DB.Exec(
			"INSERT INTO transactions (user_id, group_id, amount, source_type, quantity) VALUES (?, ?, ?, ?, ?)",
			userID, groupID, amount, string(sourceType), quantity,
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
		"SELECT id, user_id, group_id, amount, source_type, source_id, quantity, created_at FROM transactions WHERE id = ?",
		id,
	).Scan(&tx.ID, &tx.UserID, &tx.GroupID, &tx.Amount, &sourceType, &sourceID, &tx.Quantity, &tx.CreatedAt)

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
		"SELECT id, user_id, group_id, amount, source_type, source_id, quantity, created_at FROM transactions WHERE user_id = ? AND group_id = ? ORDER BY created_at DESC",
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

		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.GroupID, &tx.Amount, &sourceType, &sourceID, &tx.Quantity, &tx.CreatedAt); err != nil {
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

// GetTaskCompletionHistory retrieves detailed task completion history
func (s *Store) GetTaskCompletionHistory(userID, groupID int64) ([]*core.TaskCompletionHistory, error) {
	query := `
		SELECT
			t.id, t.user_id, t.group_id, t.amount, t.source_type, t.source_id, t.quantity, t.created_at,
			task.id, task.group_id, task.title, task.description, task.task_type, task.reward_value, task.created_at,
			u.id, u.telegram_id, u.username, u.created_at
		FROM transactions t
		JOIN tasks task ON t.source_id = task.id
		JOIN users u ON t.user_id = u.id
		WHERE t.user_id = ? AND t.group_id = ? AND t.source_type = 'task'
		ORDER BY t.created_at DESC
	`

	rows, err := s.DB.Query(query, userID, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query task completion history: %w", err)
	}
	defer rows.Close()

	var history []*core.TaskCompletionHistory
	for rows.Next() {
		var tch core.TaskCompletionHistory
		tch.Transaction = &core.Transaction{}
		tch.Task = &core.Task{}
		tch.User = &core.User{}

		var sourceType string
		var sourceID sql.NullInt64
		var taskType string
		var telegramID sql.NullInt64

		if err := rows.Scan(
			&tch.Transaction.ID, &tch.Transaction.UserID, &tch.Transaction.GroupID,
			&tch.Transaction.Amount, &sourceType, &sourceID, &tch.Transaction.Quantity, &tch.Transaction.CreatedAt,
			&tch.Task.ID, &tch.Task.GroupID, &tch.Task.Title, &tch.Task.Description,
			&taskType, &tch.Task.RewardValue, &tch.Task.CreatedAt,
			&tch.User.ID, &telegramID, &tch.User.Username, &tch.User.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task completion history: %w", err)
		}

		tch.Transaction.SourceType = core.SourceType(sourceType)
		if sourceID.Valid {
			tch.Transaction.SourceID = &sourceID.Int64
		}
		tch.Task.TaskType = core.TaskType(taskType)
		if telegramID.Valid {
			tch.User.TelegramID = &telegramID.Int64
		}

		history = append(history, &tch)
	}

	return history, nil
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
