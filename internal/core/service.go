package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
	CreateTask(groupID int64, title, description string, taskType TaskType, rewardValue int, isOneTime bool) (*Task, error)
	GetTaskByID(id int64) (*Task, error)
	GetTasksByGroupID(groupID int64) ([]*Task, error)
	UpdateTask(id int64, title, description string, taskType TaskType, rewardValue int, isOneTime bool) error
	DeleteTask(id int64) error

	// Shop operations
	CreateShopItem(groupID int64, title, description string, cost int, isOneTime bool) (*ShopItem, error)
	GetShopItemByID(id int64) (*ShopItem, error)
	GetShopItemsByGroupID(groupID int64) ([]*ShopItem, error)
	UpdateShopItem(id int64, title, description string, cost int, isOneTime bool) error
	DeleteShopItem(id int64) error

	// Transaction operations
	CreateTransaction(userID, groupID int64, amount int, sourceType SourceType, sourceID *int64, quantity int) (*Transaction, error)
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
func (s *Service) CreateTask(groupID int64, title, description string, taskType TaskType, rewardValue int, isOneTime bool) (*Task, error) {
	if title == "" {
		return nil, fmt.Errorf("task title cannot be empty")
	}
	if taskType != TaskTypeBoolean && taskType != TaskTypeInteger {
		return nil, fmt.Errorf("invalid task type: must be 'boolean' or 'integer'")
	}
	if rewardValue <= 0 {
		return nil, fmt.Errorf("reward value must be positive")
	}

	return s.store.CreateTask(groupID, title, description, taskType, rewardValue, isOneTime)
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

	// Create transaction
	transaction, err := s.store.CreateTransaction(
		userID,
		task.GroupID,
		reward,
		SourceTypeTask,
		&task.ID,
		finalQuantity,
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
func (s *Service) UpdateTask(id int64, title, description string, taskType TaskType, rewardValue int, isOneTime bool) error {
	if title == "" {
		return fmt.Errorf("task title cannot be empty")
	}
	if taskType != TaskTypeBoolean && taskType != TaskTypeInteger {
		return fmt.Errorf("invalid task type: must be 'boolean' or 'integer'")
	}
	if rewardValue <= 0 {
		return fmt.Errorf("reward value must be positive")
	}

	return s.store.UpdateTask(id, title, description, taskType, rewardValue, isOneTime)
}

// DeleteTask deletes a task
func (s *Service) DeleteTask(id int64) error {
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

	// Create negative transaction for the purchase
	transaction, err := s.store.CreateTransaction(
		userID,
		item.GroupID,
		-item.Cost,
		SourceTypeShopItem,
		&item.ID,
		1, // Always quantity 1 for shop purchases
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
	reversalAmount := -transaction.Amount
	_, err = s.store.CreateTransaction(
		transaction.UserID,
		transaction.GroupID,
		reversalAmount,
		transaction.SourceType,
		transaction.SourceID,
		transaction.Quantity,
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

// generateInviteCode generates a random invite code
func generateInviteCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
