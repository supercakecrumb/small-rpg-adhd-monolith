package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

// Store interface defines the methods required from the storage layer
type Store interface {
	// User operations
	CreateUser(username string, telegramID *int64) (*User, error)
	GetUserByID(id int64) (*User, error)
	GetUserByTelegramID(telegramID int64) (*User, error)
	GetUserByUsername(username string) (*User, error)
	GetUsersByGroupID(groupID int64) ([]*User, error)

	// Group operations
	CreateGroup(name, inviteCode string) (*Group, error)
	GetGroupByID(id int64) (*Group, error)
	GetGroupByInviteCode(inviteCode string) (*Group, error)
	GetGroupsByUserID(userID int64) ([]*Group, error)
	AddUserToGroup(userID, groupID int64) error
	IsUserInGroup(userID, groupID int64) (bool, error)

	// Task operations
	CreateTask(groupID int64, title, description string, taskType TaskType, rewardValue int, defaultQuantity int, isOneTime bool) (*Task, error)
	GetTaskByID(id int64) (*Task, error)
	GetTasksByGroupID(groupID int64) ([]*Task, error)
	UpdateTask(id int64, title, description string, taskType TaskType, rewardValue int, defaultQuantity int, isOneTime bool) error
	DeleteTask(id int64) error

	// Shop operations
	CreateShopItem(groupID int64, title, description string, cost int, isOneTime bool) (*ShopItem, error)
	GetShopItemByID(id int64) (*ShopItem, error)
	GetShopItemsByGroupID(groupID int64) ([]*ShopItem, error)
	UpdateShopItem(id int64, title, description string, cost int, isOneTime bool) error
	DeleteShopItem(id int64) error

	// Transaction operations
	CreateTransaction(userID, groupID int64, amount int, sourceType SourceType, sourceID *int64, quantity int, description, notes string) (*Transaction, error)
	GetTransactionByID(id int64) (*Transaction, error)
	GetTransactionsByUserAndGroup(userID, groupID int64) ([]*Transaction, error)
	GetBalance(userID, groupID int64) (int, error)
	GetTaskCompletionHistory(userID, groupID int64) ([]*TaskCompletionHistory, error)

	// Purchase operations
	CreatePurchase(transactionID, userID, groupID, shopItemID int64) (*Purchase, error)
	GetPurchasesByUserAndGroup(userID, groupID int64) ([]*Purchase, error)
	GetPurchaseHistoryByUserAndGroup(userID, groupID int64) ([]*PurchaseHistory, error)
	MarkPurchaseFulfilled(purchaseID, fulfilledByUserID int64, notes string) error
	CancelPurchaseByTransactionID(transactionID int64) error

	// Profile operations
	GetUserProfile(userID int64) (*UserProfile, error)
	CreateOrUpdateUserProfile(userID int64, telegramPhotoURL string, notificationEnabled bool) error
	UpdateTelegramPhoto(userID int64, photoURL string) error
	SetNotificationEnabled(userID int64, enabled bool) error

	// Notification operations
	GetNotificationSettings(userID int64) (*NotificationSettings, error)
	UpdateNotificationSettings(settings *NotificationSettings) error
	CreateNotification(notification *TaskNotification) error
	GetPendingNotifications(now time.Time) ([]*TaskNotification, error)
	MarkNotificationSent(notificationID int64) error
	DeleteNotificationsByTask(taskID int64) error
	GetNotificationByID(id int64) (*TaskNotification, error)
}

// Service provides business logic for the application
type Service struct {
	store Store
}

// NewService creates a new Service instance
func NewService(store Store) *Service {
	return &Service{
		store: store,
	}
}

// CreateUser creates a new user
func (s *Service) CreateUser(username string, telegramID *int64) (*User, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}
	return s.store.CreateUser(username, telegramID)
}

// GetUserByID retrieves a user by ID
func (s *Service) GetUserByID(id int64) (*User, error) {
	return s.store.GetUserByID(id)
}

// GetUserByTelegramID retrieves a user by Telegram ID
func (s *Service) GetUserByTelegramID(telegramID int64) (*User, error) {
	return s.store.GetUserByTelegramID(telegramID)
}

// GetUserByUsername retrieves a user by username
func (s *Service) GetUserByUsername(username string) (*User, error) {
	return s.store.GetUserByUsername(username)
}

// CreateGroup creates a new group with a generated invite code
func (s *Service) CreateGroup(name string, creatorUserID int64) (*Group, error) {
	if name == "" {
		return nil, fmt.Errorf("group name cannot be empty")
	}

	// Generate a unique invite code
	inviteCode, err := generateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite code: %w", err)
	}

	group, err := s.store.CreateGroup(name, inviteCode)
	if err != nil {
		return nil, err
	}

	// Add creator to the group
	if err := s.store.AddUserToGroup(creatorUserID, group.ID); err != nil {
		return nil, fmt.Errorf("failed to add creator to group: %w", err)
	}

	return group, nil
}

// GetGroupByID retrieves a group by ID
func (s *Service) GetGroupByID(id int64) (*Group, error) {
	return s.store.GetGroupByID(id)
}

// GetGroupsByUserID retrieves all groups for a user
func (s *Service) GetGroupsByUserID(userID int64) ([]*Group, error) {
	return s.store.GetGroupsByUserID(userID)
}

// JoinGroup adds a user to a group using an invite code
func (s *Service) JoinGroup(userID int64, inviteCode string) (*Group, error) {
	group, err := s.store.GetGroupByInviteCode(inviteCode)
	if err != nil {
		return nil, err
	}

	// Check if user is already in the group
	isMember, err := s.store.IsUserInGroup(userID, group.ID)
	if err != nil {
		return nil, err
	}
	if isMember {
		return nil, fmt.Errorf("user is already a member of this group")
	}

	if err := s.store.AddUserToGroup(userID, group.ID); err != nil {
		return nil, err
	}

	return group, nil
}

// CreateTask creates a new task in a group
func (s *Service) CreateTask(groupID int64, title, description string, taskType TaskType, rewardValue int, defaultQuantity int, isOneTime bool) (*Task, error) {
	if title == "" {
		return nil, fmt.Errorf("task title cannot be empty")
	}
	if taskType != TaskTypeBoolean && taskType != TaskTypeInteger {
		return nil, fmt.Errorf("invalid task type: must be 'boolean' or 'integer'")
	}
	if rewardValue <= 0 {
		return nil, fmt.Errorf("reward value must be positive")
	}
	if defaultQuantity <= 0 {
		defaultQuantity = 10 // Default to 10 if not provided or invalid
	}

	return s.store.CreateTask(groupID, title, description, taskType, rewardValue, defaultQuantity, isOneTime)
}

// GetTasksByGroupID retrieves all tasks for a group
func (s *Service) GetTasksByGroupID(groupID int64) ([]*Task, error) {
	return s.store.GetTasksByGroupID(groupID)
}

// CompleteTask handles task completion logic
// For boolean tasks: awards the reward_value
// For integer tasks: awards reward_value * quantity
func (s *Service) CompleteTask(userID, taskID int64, quantity *int) (*Transaction, error) {
	task, err := s.store.GetTaskByID(taskID)
	if err != nil {
		return nil, err
	}

	// Verify user is in the group
	isMember, err := s.store.IsUserInGroup(userID, task.GroupID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("user is not a member of this group")
	}

	// Calculate reward based on task type
	var reward int
	var finalQuantity int
	switch task.TaskType {
	case TaskTypeBoolean:
		// For boolean tasks, always award the reward_value
		reward = task.RewardValue
		finalQuantity = 1
	case TaskTypeInteger:
		// For integer tasks, require quantity and multiply
		if quantity == nil || *quantity <= 0 {
			return nil, fmt.Errorf("quantity must be provided and positive for integer tasks")
		}
		reward = task.RewardValue * (*quantity)
		finalQuantity = *quantity
	default:
		return nil, fmt.Errorf("unknown task type: %s", task.TaskType)
	}

	// Create transaction with task details stored
	transaction, err := s.store.CreateTransaction(
		userID,
		task.GroupID,
		reward,
		SourceTypeTask,
		&task.ID,
		finalQuantity,
		task.Title,       // Store task title as description
		task.Description, // Store task description as notes
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// If task is one-time, delete it after completion
	if task.IsOneTime {
		if err := s.store.DeleteTask(task.ID); err != nil {
			// Log error but don't fail the transaction
			// The transaction was successful, deletion is secondary
			_ = err
		}
	}

	return transaction, nil
}

// UpdateTask updates an existing task
func (s *Service) UpdateTask(id int64, title, description string, taskType TaskType, rewardValue int, defaultQuantity int, isOneTime bool) error {
	if title == "" {
		return fmt.Errorf("task title cannot be empty")
	}
	if taskType != TaskTypeBoolean && taskType != TaskTypeInteger {
		return fmt.Errorf("invalid task type: must be 'boolean' or 'integer'")
	}
	if rewardValue <= 0 {
		return fmt.Errorf("reward value must be positive")
	}
	if defaultQuantity <= 0 {
		defaultQuantity = 10 // Default to 10 if not provided or invalid
	}

	return s.store.UpdateTask(id, title, description, taskType, rewardValue, defaultQuantity, isOneTime)
}

// DeleteTask deletes a task
func (s *Service) DeleteTask(id int64) error {
	// Cancel any pending notifications before deleting the task
	if err := s.CancelNotificationsForTask(id); err != nil {
		log.Printf("Warning: failed to cancel notifications for task %d: %v", id, err)
	}

	return s.store.DeleteTask(id)
}

// CreateShopItem creates a new shop item in a group
func (s *Service) CreateShopItem(groupID int64, title, description string, cost int, isOneTime bool) (*ShopItem, error) {
	if title == "" {
		return nil, fmt.Errorf("shop item title cannot be empty")
	}
	if cost <= 0 {
		return nil, fmt.Errorf("cost must be positive")
	}

	return s.store.CreateShopItem(groupID, title, description, cost, isOneTime)
}

// GetShopItemsByGroupID retrieves all shop items for a group
func (s *Service) GetShopItemsByGroupID(groupID int64) ([]*ShopItem, error) {
	return s.store.GetShopItemsByGroupID(groupID)
}

// UpdateShopItem updates an existing shop item
func (s *Service) UpdateShopItem(id int64, title, description string, cost int, isOneTime bool) error {
	if title == "" {
		return fmt.Errorf("shop item title cannot be empty")
	}
	if cost <= 0 {
		return fmt.Errorf("cost must be positive")
	}

	return s.store.UpdateShopItem(id, title, description, cost, isOneTime)
}

// DeleteShopItem deletes a shop item
func (s *Service) DeleteShopItem(id int64) error {
	return s.store.DeleteShopItem(id)
}

// BuyItem handles purchasing an item from the shop
func (s *Service) BuyItem(userID, itemID int64) (*Transaction, error) {
	item, err := s.store.GetShopItemByID(itemID)
	if err != nil {
		return nil, err
	}

	// Verify user is in the group
	isMember, err := s.store.IsUserInGroup(userID, item.GroupID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("user is not a member of this group")
	}

	// Check if user has enough balance
	balance, err := s.store.GetBalance(userID, item.GroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if balance < item.Cost {
		return nil, fmt.Errorf("insufficient balance: have %d, need %d", balance, item.Cost)
	}

	// Create negative transaction for the purchase with item details stored
	transaction, err := s.store.CreateTransaction(
		userID,
		item.GroupID,
		-item.Cost,
		SourceTypeShopItem,
		&item.ID,
		1,                // Always quantity 1 for shop purchases
		item.Title,       // Store shop item title as description
		item.Description, // Store shop item description as notes
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Create purchase record for tracking
	purchase, err := s.store.CreatePurchase(transaction.ID, userID, item.GroupID, item.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create purchase record: %w", err)
	}
	_ = purchase // Purchase record created successfully

	// If item is one-time, delete it after purchase
	if item.IsOneTime {
		if err := s.store.DeleteShopItem(item.ID); err != nil {
			// Log error but don't fail the transaction
			// The transaction was successful, deletion is secondary
			_ = err
		}
	}

	return transaction, nil
}

// GetTaskCompletionHistory retrieves task completion history
func (s *Service) GetTaskCompletionHistory(userID, groupID int64) ([]*TaskCompletionHistory, error) {
	return s.store.GetTaskCompletionHistory(userID, groupID)
}

// GetPurchaseHistory retrieves purchase history
func (s *Service) GetPurchaseHistory(userID, groupID int64) ([]*PurchaseHistory, error) {
	return s.store.GetPurchaseHistoryByUserAndGroup(userID, groupID)
}

// MarkPurchaseFulfilled marks a purchase as fulfilled
func (s *Service) MarkPurchaseFulfilled(purchaseID, fulfilledByUserID int64, notes string) error {
	return s.store.MarkPurchaseFulfilled(purchaseID, fulfilledByUserID, notes)
}

// GetUserProfile retrieves a user's profile
func (s *Service) GetUserProfile(userID int64) (*UserProfile, error) {
	return s.store.GetUserProfile(userID)
}

// UpdateTelegramPhoto updates a user's Telegram photo
func (s *Service) UpdateTelegramPhoto(userID int64, photoURL string) error {
	return s.store.UpdateTelegramPhoto(userID, photoURL)
}

// SetNotificationEnabled sets notification preferences
func (s *Service) SetNotificationEnabled(userID int64, enabled bool) error {
	return s.store.SetNotificationEnabled(userID, enabled)
}

// GetBalance retrieves the current balance for a user in a group
func (s *Service) GetBalance(userID, groupID int64) (int, error) {
	return s.store.GetBalance(userID, groupID)
}

// GetTransactionHistory retrieves transaction history for a user in a group
func (s *Service) GetTransactionHistory(userID, groupID int64) ([]*Transaction, error) {
	return s.store.GetTransactionsByUserAndGroup(userID, groupID)
}

// GetUsersByGroupID retrieves all users in a group
func (s *Service) GetUsersByGroupID(groupID int64) ([]*User, error) {
	return s.store.GetUsersByGroupID(groupID)
}

// GetTaskByID retrieves a task by ID
func (s *Service) GetTaskByID(id int64) (*Task, error) {
	return s.store.GetTaskByID(id)
}

// GetShopItemByID retrieves a shop item by ID
func (s *Service) GetShopItemByID(id int64) (*ShopItem, error) {
	return s.store.GetShopItemByID(id)
}

// UndoTransaction creates a reversal transaction to undo a completed task or purchase
func (s *Service) UndoTransaction(userID, transactionID int64) error {
	// Get the original transaction
	transaction, err := s.store.GetTransactionByID(transactionID)
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	// Verify the transaction belongs to the user
	if transaction.UserID != userID {
		return fmt.Errorf("transaction does not belong to this user")
	}

	// Verify user is still in the group
	isMember, err := s.store.IsUserInGroup(userID, transaction.GroupID)
	if err != nil {
		return err
	}
	if !isMember {
		return fmt.Errorf("user is not a member of this group")
	}

	// Create reversal transaction (negative of original amount)
	// Keep the same description and notes for consistency
	reversalAmount := -transaction.Amount
	_, err = s.store.CreateTransaction(
		transaction.UserID,
		transaction.GroupID,
		reversalAmount,
		transaction.SourceType,
		transaction.SourceID,
		transaction.Quantity,
		transaction.Description, // Keep original description
		transaction.Notes,       // Keep original notes
	)
	if err != nil {
		return fmt.Errorf("failed to create reversal transaction: %w", err)
	}

	// If this was a purchase transaction, mark the purchase as cancelled
	if transaction.SourceType == SourceTypeShopItem && transaction.Amount < 0 {
		// Find the purchase record for this transaction
		if err := s.store.CancelPurchaseByTransactionID(transactionID); err != nil {
			// Log error but don't fail - the reversal transaction was successful
			// The cancelled_at field helps track that this was undone
			_ = err
		}
	}

	return nil
}

// ScheduleNotificationsForTask creates notification records when a task has a due date
// Creates two notifications:
// - One "on_deadline" notification scheduled at due_at
// - One "before_deadline" notification scheduled at due_at minus user's reminder_delta_minutes
func (s *Service) ScheduleNotificationsForTask(task *Task) error {
	// Skip if task has no due date
	if task.DueAt == nil {
		return nil
	}

	// Skip if due date is in the past
	if task.DueAt.Before(time.Now()) {
		return nil
	}

	// Get all members of the group to schedule notifications for them
	members, err := s.store.GetUsersByGroupID(task.GroupID)
	if err != nil {
		return fmt.Errorf("failed to get group members: %w", err)
	}

	// Schedule notifications for each member
	for _, member := range members {
		// Get user's notification settings
		settings, err := s.store.GetNotificationSettings(member.ID)
		if err != nil {
			return fmt.Errorf("failed to get notification settings for user %d: %w", member.ID, err)
		}

		// Schedule "on_deadline" notification
		onDeadlineNotif := &TaskNotification{
			TaskID:           task.ID,
			UserID:           member.ID,
			NotificationType: "on_deadline",
			ScheduledAt:      *task.DueAt,
		}
		if err := s.store.CreateNotification(onDeadlineNotif); err != nil {
			return fmt.Errorf("failed to create on_deadline notification: %w", err)
		}

		// Schedule "before_deadline" notification
		beforeDeadline := task.DueAt.Add(-time.Duration(settings.ReminderDeltaMinutes) * time.Minute)
		// Only schedule if it's still in the future
		if beforeDeadline.After(time.Now()) {
			beforeDeadlineNotif := &TaskNotification{
				TaskID:           task.ID,
				UserID:           member.ID,
				NotificationType: "before_deadline",
				ScheduledAt:      beforeDeadline,
			}
			if err := s.store.CreateNotification(beforeDeadlineNotif); err != nil {
				return fmt.Errorf("failed to create before_deadline notification: %w", err)
			}
		}
	}

	return nil
}

// RescheduleNotificationsForTask updates notifications when a task's due date changes
// Deletes existing pending notifications and creates new ones if newDueAt is not nil
func (s *Service) RescheduleNotificationsForTask(taskID int64, newDueAt *time.Time) error {
	// Delete existing pending notifications
	if err := s.store.DeleteNotificationsByTask(taskID); err != nil {
		return fmt.Errorf("failed to delete existing notifications: %w", err)
	}

	// If newDueAt is nil, we're done (just cancelled notifications)
	if newDueAt == nil {
		return nil
	}

	// Get the task to schedule new notifications
	task, err := s.store.GetTaskByID(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update the task's due_at field for scheduling
	task.DueAt = newDueAt

	// Schedule new notifications
	return s.ScheduleNotificationsForTask(task)
}

// CancelNotificationsForTask deletes all pending notifications for a task
// This should be called when a task is marked done or deleted
func (s *Service) CancelNotificationsForTask(taskID int64) error {
	return s.store.DeleteNotificationsByTask(taskID)
}

// StartNotificationWorker runs a background goroutine that checks for pending notifications
// and sends them via Telegram bot. It runs with a 1-minute ticker and handles graceful shutdown.
//
// TODO: Future improvements:
// - Add retry logic for failed notification sends
// - Implement batch notification processing for better performance
// - Add metrics/monitoring for notification delivery
// - Consider using a proper job queue system for high-scale deployments
func (s *Service) StartNotificationWorker(ctx context.Context, bot BotNotifier) {
	// Use a ticker that fires every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log := func(format string, args ...interface{}) {
		// Simple logging helper
		fmt.Printf("[NotificationWorker] "+format+"\n", args...)
	}

	log("Starting notification worker...")

	for {
		select {
		case <-ctx.Done():
			log("Shutdown signal received, stopping notification worker...")
			return

		case <-ticker.C:
			// Get pending notifications
			now := time.Now()
			notifications, err := s.store.GetPendingNotifications(now)
			if err != nil {
				log("Error fetching pending notifications: %v", err)
				continue
			}

			if len(notifications) == 0 {
				continue // No notifications to send
			}

			log("Found %d pending notification(s) to send", len(notifications))

			// Process each notification
			for _, notif := range notifications {
				if err := s.sendNotification(notif, bot); err != nil {
					log("Error sending notification %d: %v", notif.ID, err)
					// Don't mark as sent if there was an error
					// TODO: Implement retry logic with exponential backoff
					continue
				}

				// Mark as sent
				if err := s.store.MarkNotificationSent(notif.ID); err != nil {
					log("Error marking notification %d as sent: %v", notif.ID, err)
					// Continue anyway - we sent the notification successfully
				}
			}
		}
	}
}

// BotNotifier interface defines the method needed to send notifications via Telegram
type BotNotifier interface {
	SendNotification(chatID int64, message string, buttons map[string]string) error
}

// sendNotification sends a single notification via Telegram
func (s *Service) sendNotification(notif *TaskNotification, bot BotNotifier) error {
	// Get task details
	task, err := s.store.GetTaskByID(notif.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Get user to get their Telegram ID
	user, err := s.store.GetUserByID(notif.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Skip if user has no Telegram ID
	if user.TelegramID == nil {
		return fmt.Errorf("user has no Telegram ID")
	}

	// Check if user has notifications enabled
	profile, err := s.store.GetUserProfile(notif.UserID)
	if err == nil && profile != nil && !profile.NotificationEnabled {
		// User has disabled notifications, skip
		return nil
	}

	// Build notification message
	var message string
	if task.DueAt != nil {
		dueTimeStr := task.DueAt.Format("Mon, 02 Jan 2006 at 15:04")
		message = fmt.Sprintf("⏰ Task Reminder\n\nTask: %s\nDue: %s", task.Title, dueTimeStr)
	} else {
		message = fmt.Sprintf("⏰ Task Reminder\n\nTask: %s", task.Title)
	}

	if task.Description != "" {
		message += fmt.Sprintf("\n\n%s", task.Description)
	}

	// Create inline buttons
	// Callback data format: "notify_done_{notif_id}", "notify_snooze_{notif_id}", "notify_later_{notif_id}"
	buttons := map[string]string{
		"✅ Done":               fmt.Sprintf("notify_done_%d", notif.ID),
		"⏰ Will do in 15 mins": fmt.Sprintf("notify_snooze_%d", notif.ID),
		"🔔 Remind later":       fmt.Sprintf("notify_later_%d", notif.ID),
	}

	// Send via bot
	return bot.SendNotification(*user.TelegramID, message, buttons)
}

// GetNotificationByID retrieves a notification by its ID
func (s *Service) GetNotificationByID(id int64) (*TaskNotification, error) {
	return s.store.GetNotificationByID(id)
}

// GetNotificationSettings retrieves notification settings for a user
func (s *Service) GetNotificationSettings(userID int64) (*NotificationSettings, error) {
	return s.store.GetNotificationSettings(userID)
}

// CreateNotification creates a new notification (used for snoozing)
func (s *Service) CreateNotification(notification *TaskNotification) error {
	return s.store.CreateNotification(notification)
}

// generateInviteCode generates a random invite code
func generateInviteCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
