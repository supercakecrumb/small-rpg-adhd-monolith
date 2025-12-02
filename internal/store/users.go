package store

import (
	"database/sql"
	"fmt"
	"small-rpg-adhd-monolith/internal/core"
)

// CreateUser creates a new user
func (s *Store) CreateUser(username string, telegramID *int64) (*core.User, error) {
	var result sql.Result
	var err error

	if telegramID != nil {
		result, err = s.DB.Exec(
			"INSERT INTO users (username, telegram_id) VALUES (?, ?)",
			username, *telegramID,
		)
	} else {
		result, err = s.DB.Exec(
			"INSERT INTO users (username) VALUES (?)",
			username,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetUserByID(id)
}

// GetUserByID retrieves a user by ID
func (s *Store) GetUserByID(id int64) (*core.User, error) {
	user := &core.User{}
	var telegramID sql.NullInt64

	err := s.DB.QueryRow(
		"SELECT id, telegram_id, username, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &telegramID, &user.Username, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if telegramID.Valid {
		user.TelegramID = &telegramID.Int64
	}

	return user, nil
}

// GetUserByTelegramID retrieves a user by Telegram ID
func (s *Store) GetUserByTelegramID(telegramID int64) (*core.User, error) {
	user := &core.User{}
	var tgID sql.NullInt64

	err := s.DB.QueryRow(
		"SELECT id, telegram_id, username, created_at FROM users WHERE telegram_id = ?",
		telegramID,
	).Scan(&user.ID, &tgID, &user.Username, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if tgID.Valid {
		user.TelegramID = &tgID.Int64
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (s *Store) GetUserByUsername(username string) (*core.User, error) {
	user := &core.User{}
	var telegramID sql.NullInt64

	err := s.DB.QueryRow(
		"SELECT id, telegram_id, username, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &telegramID, &user.Username, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if telegramID.Valid {
		user.TelegramID = &telegramID.Int64
	}

	return user, nil
}

// GetUsersByGroupID retrieves all users in a group
func (s *Store) GetUsersByGroupID(groupID int64) ([]*core.User, error) {
	rows, err := s.DB.Query(`
		SELECT u.id, u.telegram_id, u.username, u.created_at
		FROM users u
		INNER JOIN group_members gm ON u.id = gm.user_id
		WHERE gm.group_id = ?
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*core.User
	for rows.Next() {
		user := &core.User{}
		var telegramID sql.NullInt64

		if err := rows.Scan(&user.ID, &telegramID, &user.Username, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if telegramID.Valid {
			user.TelegramID = &telegramID.Int64
		}

		users = append(users, user)
	}

	return users, nil
}
