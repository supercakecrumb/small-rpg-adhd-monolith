package store

import (
	"database/sql"
	"fmt"
	"small-rpg-adhd-monolith/internal/core"
)

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

// DeleteTask deletes a task
func (s *Store) DeleteTask(id int64) error {
	query := `DELETE FROM tasks WHERE id = ?`

	_, err := s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
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

// DeleteShopItem deletes a shop item
func (s *Store) DeleteShopItem(id int64) error {
	query := `DELETE FROM shop_items WHERE id = ?`

	_, err := s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete shop item: %w", err)
	}

	return nil
}
