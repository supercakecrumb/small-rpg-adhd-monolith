package store

import (
	"database/sql"
	"fmt"
	"time"

	"small-rpg-adhd-monolith/internal/core"
)

// CreateOrUpdateUserProfile creates or updates a user profile
func (s *Store) CreateOrUpdateUserProfile(userID int64, telegramPhotoURL string, notificationEnabled bool) error {
	query := `
		INSERT INTO user_profiles (user_id, telegram_photo_url, telegram_photo_cached_at, notification_enabled)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			telegram_photo_url = excluded.telegram_photo_url,
			telegram_photo_cached_at = excluded.telegram_photo_cached_at,
			notification_enabled = excluded.notification_enabled
	`

	now := time.Now()
	_, err := s.DB.Exec(query, userID, telegramPhotoURL, now, notificationEnabled)
	if err != nil {
		return fmt.Errorf("failed to create/update user profile: %w", err)
	}

	return nil
}

// GetUserProfile retrieves a user's profile
func (s *Store) GetUserProfile(userID int64) (*core.UserProfile, error) {
	query := `
		SELECT user_id, telegram_photo_url, telegram_photo_cached_at, notification_enabled
		FROM user_profiles
		WHERE user_id = ?
	`

	var up core.UserProfile
	var cachedAt sql.NullTime

	err := s.DB.QueryRow(query, userID).Scan(
		&up.UserID, &up.TelegramPhotoURL, &cachedAt, &up.NotificationEnabled,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default profile if not found
			return &core.UserProfile{
				UserID:              userID,
				NotificationEnabled: true,
			}, nil
		}
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	if cachedAt.Valid {
		up.TelegramPhotoCachedAt = &cachedAt.Time
	}

	return &up, nil
}

// UpdateTelegramPhoto updates a user's Telegram profile photo URL
func (s *Store) UpdateTelegramPhoto(userID int64, photoURL string) error {
	query := `
		INSERT INTO user_profiles (user_id, telegram_photo_url, telegram_photo_cached_at, notification_enabled)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(user_id) DO UPDATE SET
			telegram_photo_url = excluded.telegram_photo_url,
			telegram_photo_cached_at = excluded.telegram_photo_cached_at
	`

	now := time.Now()
	_, err := s.DB.Exec(query, userID, photoURL, now)
	if err != nil {
		return fmt.Errorf("failed to update telegram photo: %w", err)
	}

	return nil
}

// SetNotificationEnabled sets the notification preference for a user
func (s *Store) SetNotificationEnabled(userID int64, enabled bool) error {
	query := `
		INSERT INTO user_profiles (user_id, notification_enabled)
		VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			notification_enabled = excluded.notification_enabled
	`

	_, err := s.DB.Exec(query, userID, enabled)
	if err != nil {
		return fmt.Errorf("failed to set notification enabled: %w", err)
	}

	return nil
}
