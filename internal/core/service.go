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
	CreateTask(groupID int64, title, description string, taskType TaskType, rewardValue int) (*Task, error)
	GetTaskByID(id int64) (*Task, error)
	GetTasksByGroupID(groupID int64) ([]*Task, error)

	// Shop operations
	CreateShopItem(groupID int64, title, description string, cost int) (*ShopItem, error)
	GetShopItemByID(id int64) (*ShopItem, error)
	GetShopItemsByGroupID(groupID int64) ([]*ShopItem, error)

	// Transaction operations
	CreateTransaction(userID, groupID int64, amount int, sourceType SourceType, sourceID *int64) (*Transaction, error)
	GetTransactionByID(id int64) (*Transaction, error)
	GetTransactionsByUserAndGroup(userID, groupID int64) ([]*Transaction, error)
	GetBalance(userID, groupID int64) (int, error)
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
func (s *Service) CreateTask(groupID int64, title, description string, taskType TaskType, rewardValue int) (*Task, error) {
	if title == "" {
		return nil, fmt.Errorf("task title cannot be empty")
	}
	if taskType != TaskTypeBoolean && taskType != TaskTypeInteger {
		return nil, fmt.Errorf("invalid task type: must be 'boolean' or 'integer'")
	}
	if rewardValue <= 0 {
		return nil, fmt.Errorf("reward value must be positive")
	}

	return s.store.CreateTask(groupID, title, description, taskType, rewardValue)
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
	switch task.TaskType {
	case TaskTypeBoolean:
		// For boolean tasks, always award the reward_value
		reward = task.RewardValue
	case TaskTypeInteger:
		// For integer tasks, require quantity and multiply
		if quantity == nil || *quantity <= 0 {
			return nil, fmt.Errorf("quantity must be provided and positive for integer tasks")
		}
		reward = task.RewardValue * (*quantity)
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
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return transaction, nil
}

// CreateShopItem creates a new shop item in a group
func (s *Service) CreateShopItem(groupID int64, title, description string, cost int) (*ShopItem, error) {
	if title == "" {
		return nil, fmt.Errorf("shop item title cannot be empty")
	}
	if cost <= 0 {
		return nil, fmt.Errorf("cost must be positive")
	}

	return s.store.CreateShopItem(groupID, title, description, cost)
}

// GetShopItemsByGroupID retrieves all shop items for a group
func (s *Service) GetShopItemsByGroupID(groupID int64) ([]*ShopItem, error) {
	return s.store.GetShopItemsByGroupID(groupID)
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
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return transaction, nil
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

// generateInviteCode generates a random invite code
func generateInviteCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
