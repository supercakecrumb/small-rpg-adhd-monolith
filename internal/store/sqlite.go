package store

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Store wraps the database connection
type Store struct {
	DB *sql.DB
}

// NewStore creates a new Store and initializes the database
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{DB: db}

	// Run migrations
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

// migrate creates all necessary tables
func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id INTEGER UNIQUE,
		username TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		invite_code TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS group_members (
		user_id INTEGER,
		group_id INTEGER,
		joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, group_id),
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(group_id) REFERENCES groups(id)
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id INTEGER,
		title TEXT NOT NULL,
		description TEXT,
		task_type TEXT CHECK(task_type IN ('boolean', 'integer')) NOT NULL,
		reward_value INTEGER NOT NULL,
		is_one_time BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(group_id) REFERENCES groups(id)
	);

	CREATE TABLE IF NOT EXISTS shop_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id INTEGER,
		title TEXT NOT NULL,
		description TEXT,
		cost INTEGER NOT NULL,
		is_one_time BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(group_id) REFERENCES groups(id)
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		group_id INTEGER,
		amount INTEGER NOT NULL,
		source_type TEXT CHECK(source_type IN ('task', 'shop_item', 'manual')),
		source_id INTEGER,
		quantity INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(group_id) REFERENCES groups(id)
	);

	CREATE TABLE IF NOT EXISTS purchases (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		transaction_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		group_id INTEGER NOT NULL,
		shop_item_id INTEGER NOT NULL,
		fulfilled BOOLEAN DEFAULT 0,
		fulfilled_at DATETIME,
		fulfilled_by INTEGER,
		notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(transaction_id) REFERENCES transactions(id),
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(group_id) REFERENCES groups(id),
		FOREIGN KEY(shop_item_id) REFERENCES shop_items(id),
		FOREIGN KEY(fulfilled_by) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS user_profiles (
		user_id INTEGER PRIMARY KEY,
		telegram_photo_url TEXT,
		telegram_photo_cached_at DATETIME,
		notification_enabled BOOLEAN DEFAULT 1,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	`

	_, err := s.DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Run migrations for existing databases
	if err := s.migrateIsOneTime(); err != nil {
		return fmt.Errorf("failed to migrate is_one_time columns: %w", err)
	}

	if err := s.migrateQuantity(); err != nil {
		return fmt.Errorf("failed to migrate quantity column: %w", err)
	}

	if err := s.migrateCancelledAt(); err != nil {
		return fmt.Errorf("failed to migrate cancelled_at column: %w", err)
	}

	return nil
}

// migrateIsOneTime adds is_one_time columns to existing tables if they don't exist
func (s *Store) migrateIsOneTime() error {
	// Try to add is_one_time column to tasks table
	// This will fail silently if column already exists
	_, err := s.DB.Exec(`ALTER TABLE tasks ADD COLUMN is_one_time BOOLEAN DEFAULT 0`)
	if err != nil {
		// Check if error is due to column already existing (which is fine)
		// SQLite returns "duplicate column name" error
		if err.Error() != "duplicate column name: is_one_time" {
			// Only log non-duplicate errors, but don't fail
			// Column might already exist from previous migration
		}
	}

	// Try to add is_one_time column to shop_items table
	_, err = s.DB.Exec(`ALTER TABLE shop_items ADD COLUMN is_one_time BOOLEAN DEFAULT 0`)
	if err != nil {
		// Check if error is due to column already existing (which is fine)
		if err.Error() != "duplicate column name: is_one_time" {
			// Only log non-duplicate errors, but don't fail
		}
	}

	return nil
}

// migrateQuantity adds quantity column to transactions table if it doesn't exist
func (s *Store) migrateQuantity() error {
	// Try to add quantity column to transactions table
	// This will fail silently if column already exists
	_, err := s.DB.Exec(`ALTER TABLE transactions ADD COLUMN quantity INTEGER DEFAULT 1`)
	if err != nil {
		// Check if error is due to column already existing (which is fine)
		if err.Error() != "duplicate column name: quantity" {
			// Only log non-duplicate errors, but don't fail
		}
	}

	return nil
}

// migrateCancelledAt adds cancelled_at column to purchases table if it doesn't exist
func (s *Store) migrateCancelledAt() error {
	// Try to add cancelled_at column to purchases table
	// This will fail silently if column already exists
	_, err := s.DB.Exec(`ALTER TABLE purchases ADD COLUMN cancelled_at DATETIME`)
	if err != nil {
		// Check if error is due to column already existing (which is fine)
		if err.Error() != "duplicate column name: cancelled_at" {
			// Only log non-duplicate errors, but don't fail
		}
	}

	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.DB.Close()
}
