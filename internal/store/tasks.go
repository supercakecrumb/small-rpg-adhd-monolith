package store

import (
	"database/sql"
	"fmt"
	"small-rpg-adhd-monolith/internal/core"
	"sync"
	"time"
)

// deletedItem holds a deleted item with timestamp for undo functionality
type deletedItem struct {
	data      interface{}
	deletedAt time.Time
}

// undoCache holds recently deleted items for undo functionality
type undoCache struct {
	tasks     map[int64]*deletedItem
	shopItems map[int64]*deletedItem
	mu        sync.RWMutex
	ttl       time.Duration
}

// newUndoCache creates a new undo cache with 30 minute TTL
func newUndoCache() *undoCache {
	cache := &undoCache{
		tasks:     make(map[int64]*deletedItem),
		shopItems: make(map[int64]*deletedItem),
		ttl:       30 * time.Minute,
	}
	// Start cleanup goroutine
	go cache.cleanup()
	return cache
}

// cleanup removes expired items from cache
func (c *undoCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()

		// Clean expired tasks
		for id, item := range c.tasks {
			if now.Sub(item.deletedAt) > c.ttl {
				delete(c.tasks, id)
			}
		}

		// Clean expired shop items
		for id, item := range c.shopItems {
			if now.Sub(item.deletedAt) > c.ttl {
				delete(c.shopItems, id)
			}
		}

		c.mu.Unlock()
	}
}

// cacheTask stores a deleted task
func (c *undoCache) cacheTask(task *core.Task) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tasks[task.ID] = &deletedItem{
		data:      task,
		deletedAt: time.Now(),
	}
}

// cacheShopItem stores a deleted shop item
func (c *undoCache) cacheShopItem(item *core.ShopItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shopItems[item.ID] = &deletedItem{
		data:      item,
		deletedAt: time.Now(),
	}
}

// getTask retrieves a cached task if not expired
func (c *undoCache) getTask(id int64) (*core.Task, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found in undo cache")
	}

	if time.Since(item.deletedAt) > c.ttl {
		return nil, fmt.Errorf("undo period expired")
	}

	task, ok := item.data.(*core.Task)
	if !ok {
		return nil, fmt.Errorf("invalid task data in cache")
	}

	return task, nil
}

// getShopItem retrieves a cached shop item if not expired
func (c *undoCache) getShopItem(id int64) (*core.ShopItem, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.shopItems[id]
	if !exists {
		return nil, fmt.Errorf("shop item not found in undo cache")
	}

	if time.Since(item.deletedAt) > c.ttl {
		return nil, fmt.Errorf("undo period expired")
	}

	shopItem, ok := item.data.(*core.ShopItem)
	if !ok {
		return nil, fmt.Errorf("invalid shop item data in cache")
	}

	return shopItem, nil
}

// removeTask removes a task from cache
func (c *undoCache) removeTask(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.tasks, id)
}

// removeShopItem removes a shop item from cache
func (c *undoCache) removeShopItem(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.shopItems, id)
}

// CreateTask creates a new task in a group
func (s *Store) CreateTask(groupID int64, title, description string, taskType core.TaskType, rewardValue int, defaultQuantity int, isOneTime bool) (*core.Task, error) {
	result, err := s.DB.Exec(
		"INSERT INTO tasks (group_id, title, description, task_type, reward_value, default_quantity, is_one_time, due_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		groupID, title, description, string(taskType), rewardValue, defaultQuantity, isOneTime, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetTaskByID(id)
}

// GetTaskByID retrieves a task by ID
func (s *Store) GetTaskByID(id int64) (*core.Task, error) {
	task := &core.Task{}
	var taskType string

	err := s.DB.QueryRow(
		"SELECT id, group_id, title, description, task_type, reward_value, default_quantity, is_one_time, created_at FROM tasks WHERE id = ?",
		id,
	).Scan(&task.ID, &task.GroupID, &task.Title, &task.Description, &taskType, &task.RewardValue, &task.DefaultQuantity, &task.IsOneTime, &task.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	task.TaskType = core.TaskType(taskType)
	return task, nil
}

// GetTasksByGroupID retrieves all tasks for a group
func (s *Store) GetTasksByGroupID(groupID int64) ([]*core.Task, error) {
	rows, err := s.DB.Query(
		"SELECT id, group_id, title, description, task_type, reward_value, default_quantity, is_one_time, created_at FROM tasks WHERE group_id = ?",
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*core.Task
	for rows.Next() {
		task := &core.Task{}
		var taskType string

		if err := rows.Scan(&task.ID, &task.GroupID, &task.Title, &task.Description, &taskType, &task.RewardValue, &task.DefaultQuantity, &task.IsOneTime, &task.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		task.TaskType = core.TaskType(taskType)
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// CreateShopItem creates a new shop item in a group
func (s *Store) CreateShopItem(groupID int64, title, description string, cost int, isOneTime bool) (*core.ShopItem, error) {
	result, err := s.DB.Exec(
		"INSERT INTO shop_items (group_id, title, description, cost, is_one_time) VALUES (?, ?, ?, ?, ?)",
		groupID, title, description, cost, isOneTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop item: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetShopItemByID(id)
}

// GetShopItemByID retrieves a shop item by ID
func (s *Store) GetShopItemByID(id int64) (*core.ShopItem, error) {
	item := &core.ShopItem{}

	err := s.DB.QueryRow(
		"SELECT id, group_id, title, description, cost, is_one_time, created_at FROM shop_items WHERE id = ?",
		id,
	).Scan(&item.ID, &item.GroupID, &item.Title, &item.Description, &item.Cost, &item.IsOneTime, &item.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("shop item not found")
		}
		return nil, fmt.Errorf("failed to get shop item: %w", err)
	}

	return item, nil
}

// GetShopItemsByGroupID retrieves all shop items for a group
func (s *Store) GetShopItemsByGroupID(groupID int64) ([]*core.ShopItem, error) {
	rows, err := s.DB.Query(
		"SELECT id, group_id, title, description, cost, is_one_time, created_at FROM shop_items WHERE group_id = ?",
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query shop items: %w", err)
	}
	defer rows.Close()

	var items []*core.ShopItem
	for rows.Next() {
		item := &core.ShopItem{}
		if err := rows.Scan(&item.ID, &item.GroupID, &item.Title, &item.Description, &item.Cost, &item.IsOneTime, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan shop item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// UpdateTask updates a task's details
// Note: This method currently doesn't update due_at field
// TODO: Add UpdateTaskWithDueDate method or extend this to handle due_at when UI is implemented
func (s *Store) UpdateTask(id int64, title, description string, taskType core.TaskType, rewardValue int, defaultQuantity int, isOneTime bool) error {
	query := `
		UPDATE tasks
		SET title = ?, description = ?, task_type = ?, reward_value = ?, default_quantity = ?, is_one_time = ?
		WHERE id = ?
	`

	_, err := s.DB.Exec(query, title, description, string(taskType), rewardValue, defaultQuantity, isOneTime, id)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// DeleteTask deletes a task and caches it for undo
func (s *Store) DeleteTask(id int64) error {
	// Get task before deletion for caching
	task, err := s.GetTaskByID(id)
	if err != nil {
		return fmt.Errorf("failed to get task before deletion: %w", err)
	}

	// Delete the task
	query := `DELETE FROM tasks WHERE id = ?`
	_, err = s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	// Cache for undo
	s.undoCache.cacheTask(task)

	return nil
}

// UndoTaskDeletion restores a deleted task from cache
func (s *Store) UndoTaskDeletion(id int64) (*core.Task, error) {
	// Get task from cache
	task, err := s.undoCache.getTask(id)
	if err != nil {
		return nil, err
	}

	// Re-insert the task with the same ID
	query := `INSERT INTO tasks (id, group_id, title, description, task_type, reward_value, default_quantity, is_one_time, due_at, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.DB.Exec(query, task.ID, task.GroupID, task.Title, task.Description, string(task.TaskType),
		task.RewardValue, task.DefaultQuantity, task.IsOneTime, task.DueAt, task.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to restore task: %w", err)
	}

	// Remove from cache
	s.undoCache.removeTask(id)

	return task, nil
}

// UpdateShopItem updates a shop item's details
func (s *Store) UpdateShopItem(id int64, title, description string, cost int, isOneTime bool) error {
	query := `
		UPDATE shop_items
		SET title = ?, description = ?, cost = ?, is_one_time = ?
		WHERE id = ?
	`

	_, err := s.DB.Exec(query, title, description, cost, isOneTime, id)
	if err != nil {
		return fmt.Errorf("failed to update shop item: %w", err)
	}

	return nil
}

// DeleteShopItem deletes a shop item and caches it for undo
func (s *Store) DeleteShopItem(id int64) error {
	// Get shop item before deletion for caching
	item, err := s.GetShopItemByID(id)
	if err != nil {
		return fmt.Errorf("failed to get shop item before deletion: %w", err)
	}

	// Delete the shop item
	query := `DELETE FROM shop_items WHERE id = ?`
	_, err = s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete shop item: %w", err)
	}

	// Cache for undo
	s.undoCache.cacheShopItem(item)

	return nil
}

// UndoShopItemDeletion restores a deleted shop item from cache
func (s *Store) UndoShopItemDeletion(id int64) (*core.ShopItem, error) {
	// Get shop item from cache
	item, err := s.undoCache.getShopItem(id)
	if err != nil {
		return nil, err
	}

	// Re-insert the shop item with the same ID
	query := `INSERT INTO shop_items (id, group_id, title, description, cost, is_one_time, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = s.DB.Exec(query, item.ID, item.GroupID, item.Title, item.Description, item.Cost, item.IsOneTime, item.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to restore shop item: %w", err)
	}

	// Remove from cache
	s.undoCache.removeShopItem(id)

	return item, nil
}
