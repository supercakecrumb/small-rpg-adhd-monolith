package core

import "time"

// User represents a user in the system
type User struct {
	ID         int64
	TelegramID *int64 // Nullable
	Username   string
	Language   string
	CreatedAt  time.Time
}

// Group represents a group in the system
type Group struct {
	ID         int64
	Name       string
	InviteCode string
	OwnerID    int64
	CreatedAt  time.Time
}

// GroupMember represents a user's membership in a group
type GroupMember struct {
	UserID   int64
	GroupID  int64
	JoinedAt time.Time
}

// TaskType represents the type of a task
type TaskType string

const (
	TaskTypeBoolean TaskType = "boolean"
	TaskTypeInteger TaskType = "integer"
)

// Task represents a task in a group
type Task struct {
	ID              int64
	GroupID         int64
	Title           string
	Description     string
	TaskType        TaskType
	RewardValue     int // Coins per completion or per unit
	DefaultQuantity int // Default quantity for integer tasks
	IsOneTime       bool
	DueAt           *time.Time // Optional deadline for the task
	CreatedAt       time.Time
}

// ShopItem represents an item in the group shop
type ShopItem struct {
	ID          int64
	GroupID     int64
	Title       string
	Description string
	Cost        int
	IsOneTime   bool
	CreatedAt   time.Time
}

// SourceType represents the source of a transaction
type SourceType string

const (
	SourceTypeTask     SourceType = "task"
	SourceTypeShopItem SourceType = "shop_item"
	SourceTypeManual   SourceType = "manual"
)

// Transaction represents a coin transaction
type Transaction struct {
	ID          int64
	UserID      int64
	GroupID     int64
	Amount      int // Positive for earnings, negative for spending
	SourceType  SourceType
	SourceID    *int64 // Nullable FK to Task or ShopItem
	Quantity    int    // For integer tasks: how many units were completed
	Description string // Stored task title or shop item title
	Notes       string // Stored task description or shop item description
	CreatedAt   time.Time
}

// Purchase represents a shop item purchase with fulfillment tracking
type Purchase struct {
	ID            int64
	TransactionID int64
	UserID        int64
	GroupID       int64
	ShopItemID    int64
	Fulfilled     bool
	FulfilledAt   *time.Time
	FulfilledBy   *int64 // User who fulfilled the purchase
	Notes         string
	CreatedAt     time.Time
}

// UserProfile represents extended user profile information
type UserProfile struct {
	UserID                int64
	TelegramPhotoURL      string
	TelegramPhotoCachedAt *time.Time
	NotificationEnabled   bool
}

// TaskCompletionHistory represents detailed task completion info
type TaskCompletionHistory struct {
	Transaction *Transaction
	Task        *Task
	User        *User
}

// PurchaseHistory represents detailed purchase info
type PurchaseHistory struct {
	Purchase *Purchase
	ShopItem *ShopItem
	User     *User
}

// NotificationSettings represents user-specific notification preferences
type NotificationSettings struct {
	UserID               int64
	ReminderDeltaMinutes int // Notify X minutes before deadline
	SnoozeDefaultMinutes int // Default snooze duration
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// TaskNotification represents a scheduled or sent notification for a task
type TaskNotification struct {
	ID               int64
	TaskID           int64
	UserID           int64
	NotificationType string // 'before_deadline', 'on_deadline', 'snooze'
	ScheduledAt      time.Time
	SentAt           *time.Time // NULL if pending
	CreatedAt        time.Time
}
