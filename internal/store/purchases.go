package store

import (
	"database/sql"
	"fmt"
	"time"

	"small-rpg-adhd-monolith/internal/core"
)

// CreatePurchase creates a new purchase record
func (s *Store) CreatePurchase(transactionID, userID, groupID, shopItemID int64) (*core.Purchase, error) {
	query := `
		INSERT INTO purchases (transaction_id, user_id, group_id, shop_item_id)
		VALUES (?, ?, ?, ?)
	`

	result, err := s.DB.Exec(query, transactionID, userID, groupID, shopItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to create purchase: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get purchase ID: %w", err)
	}

	return s.GetPurchaseByID(id)
}

// GetPurchaseByID retrieves a purchase by ID
func (s *Store) GetPurchaseByID(id int64) (*core.Purchase, error) {
	query := `
		SELECT id, transaction_id, user_id, group_id, shop_item_id,
		       fulfilled, fulfilled_at, fulfilled_by, notes, created_at
		FROM purchases
		WHERE id = ?
	`

	var p core.Purchase
	var fulfilledAt sql.NullTime
	var fulfilledBy sql.NullInt64

	err := s.DB.QueryRow(query, id).Scan(
		&p.ID, &p.TransactionID, &p.UserID, &p.GroupID, &p.ShopItemID,
		&p.Fulfilled, &fulfilledAt, &fulfilledBy, &p.Notes, &p.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("purchase not found")
		}
		return nil, fmt.Errorf("failed to get purchase: %w", err)
	}

	if fulfilledAt.Valid {
		p.FulfilledAt = &fulfilledAt.Time
	}
	if fulfilledBy.Valid {
		p.FulfilledBy = &fulfilledBy.Int64
	}

	return &p, nil
}

// GetPurchasesByUserAndGroup retrieves all purchases for a user in a group
func (s *Store) GetPurchasesByUserAndGroup(userID, groupID int64) ([]*core.Purchase, error) {
	query := `
		SELECT id, transaction_id, user_id, group_id, shop_item_id,
		       fulfilled, fulfilled_at, fulfilled_by, notes, created_at
		FROM purchases
		WHERE user_id = ? AND group_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.DB.Query(query, userID, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchases: %w", err)
	}
	defer rows.Close()

	var purchases []*core.Purchase
	for rows.Next() {
		var p core.Purchase
		var fulfilledAt sql.NullTime
		var fulfilledBy sql.NullInt64

		if err := rows.Scan(
			&p.ID, &p.TransactionID, &p.UserID, &p.GroupID, &p.ShopItemID,
			&p.Fulfilled, &fulfilledAt, &fulfilledBy, &p.Notes, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan purchase: %w", err)
		}

		if fulfilledAt.Valid {
			p.FulfilledAt = &fulfilledAt.Time
		}
		if fulfilledBy.Valid {
			p.FulfilledBy = &fulfilledBy.Int64
		}

		purchases = append(purchases, &p)
	}

	return purchases, nil
}

// GetPurchasesByGroupID retrieves all purchases in a group
func (s *Store) GetPurchasesByGroupID(groupID int64) ([]*core.Purchase, error) {
	query := `
		SELECT id, transaction_id, user_id, group_id, shop_item_id,
		       fulfilled, fulfilled_at, fulfilled_by, COALESCE(notes, '') as notes, created_at
		FROM purchases
		WHERE group_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.DB.Query(query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchases: %w", err)
	}
	defer rows.Close()

	var purchases []*core.Purchase
	for rows.Next() {
		var p core.Purchase
		var fulfilledAt sql.NullTime
		var fulfilledBy sql.NullInt64

		if err := rows.Scan(
			&p.ID, &p.TransactionID, &p.UserID, &p.GroupID, &p.ShopItemID,
			&p.Fulfilled, &fulfilledAt, &fulfilledBy, &p.Notes, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan purchase: %w", err)
		}

		if fulfilledAt.Valid {
			p.FulfilledAt = &fulfilledAt.Time
		}
		if fulfilledBy.Valid {
			p.FulfilledBy = &fulfilledBy.Int64
		}

		purchases = append(purchases, &p)
	}

	return purchases, nil
}

// MarkPurchaseFulfilled marks a purchase as fulfilled
func (s *Store) MarkPurchaseFulfilled(purchaseID, fulfilledByUserID int64, notes string) error {
	query := `
		UPDATE purchases
		SET fulfilled = 1, fulfilled_at = ?, fulfilled_by = ?, notes = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := s.DB.Exec(query, now, fulfilledByUserID, notes, purchaseID)
	if err != nil {
		return fmt.Errorf("failed to mark purchase as fulfilled: %w", err)
	}

	return nil
}

// CancelPurchaseByTransactionID marks a purchase as cancelled by transaction ID
func (s *Store) CancelPurchaseByTransactionID(transactionID int64) error {
	query := `
		UPDATE purchases
		SET cancelled_at = ?
		WHERE transaction_id = ? AND cancelled_at IS NULL
	`

	now := time.Now()
	result, err := s.DB.Exec(query, now, transactionID)
	if err != nil {
		return fmt.Errorf("failed to cancel purchase: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// It's okay if no rows were affected - might not be a purchase transaction
	_ = rowsAffected

	return nil
}

// GetPurchaseHistoryByUserAndGroup retrieves detailed purchase history
func (s *Store) GetPurchaseHistoryByUserAndGroup(userID, groupID int64) ([]*core.PurchaseHistory, error) {
	query := `
		SELECT
			p.id, p.transaction_id, p.user_id, p.group_id, p.shop_item_id,
			p.fulfilled, p.fulfilled_at, p.fulfilled_by, COALESCE(p.notes, '') as notes, p.created_at,
			t.description, t.notes,
			si.id, si.group_id, si.title, si.description, si.cost, si.created_at,
			u.id, u.telegram_id, u.username, u.created_at
		FROM purchases p
		JOIN transactions t ON p.transaction_id = t.id
		LEFT JOIN shop_items si ON p.shop_item_id = si.id
		JOIN users u ON p.user_id = u.id
		WHERE p.user_id = ? AND p.group_id = ?
		ORDER BY p.created_at DESC
	`

	rows, err := s.DB.Query(query, userID, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchase history: %w", err)
	}
	defer rows.Close()

	var history []*core.PurchaseHistory
	for rows.Next() {
		var ph core.PurchaseHistory
		ph.Purchase = &core.Purchase{}
		ph.ShopItem = &core.ShopItem{}
		ph.User = &core.User{}

		var fulfilledAt sql.NullTime
		var fulfilledBy sql.NullInt64
		var telegramID sql.NullInt64
		var transactionDescription sql.NullString
		var transactionNotes sql.NullString

		// Shop item fields are nullable since LEFT JOIN may not find the item
		var shopItemID sql.NullInt64
		var shopItemGroupID sql.NullInt64
		var shopItemTitle sql.NullString
		var shopItemDescription sql.NullString
		var shopItemCost sql.NullInt64
		var shopItemCreatedAt sql.NullTime

		if err := rows.Scan(
			&ph.Purchase.ID, &ph.Purchase.TransactionID, &ph.Purchase.UserID,
			&ph.Purchase.GroupID, &ph.Purchase.ShopItemID, &ph.Purchase.Fulfilled,
			&fulfilledAt, &fulfilledBy, &ph.Purchase.Notes, &ph.Purchase.CreatedAt,
			&transactionDescription, &transactionNotes,
			&shopItemID, &shopItemGroupID, &shopItemTitle,
			&shopItemDescription, &shopItemCost, &shopItemCreatedAt,
			&ph.User.ID, &telegramID, &ph.User.Username, &ph.User.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan purchase history: %w", err)
		}

		if fulfilledAt.Valid {
			ph.Purchase.FulfilledAt = &fulfilledAt.Time
		}
		if fulfilledBy.Valid {
			ph.Purchase.FulfilledBy = &fulfilledBy.Int64
		}
		if telegramID.Valid {
			ph.User.TelegramID = &telegramID.Int64
		}

		// Prefer transaction's stored description/notes, fall back to shop item if available
		if transactionDescription.Valid && transactionDescription.String != "" {
			// Use stored transaction data (preferred for deleted items)
			ph.ShopItem.Title = transactionDescription.String
			if transactionNotes.Valid {
				ph.ShopItem.Description = transactionNotes.String
			}
		} else if shopItemID.Valid {
			// Fall back to shop item data if transaction didn't store it (old records)
			ph.ShopItem.ID = shopItemID.Int64
			ph.ShopItem.GroupID = shopItemGroupID.Int64
			ph.ShopItem.Title = shopItemTitle.String
			ph.ShopItem.Description = shopItemDescription.String
			ph.ShopItem.Cost = int(shopItemCost.Int64)
			ph.ShopItem.CreatedAt = shopItemCreatedAt.Time
		} else {
			// Neither transaction data nor shop item exists
			ph.ShopItem.Title = "[Deleted Item]"
			ph.ShopItem.Description = "This shop item has been deleted"
		}

		history = append(history, &ph)
	}

	return history, nil
}
