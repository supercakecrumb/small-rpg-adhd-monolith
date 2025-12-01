package store

import (
	"database/sql"
	"fmt"
	"small-rpg-adhd-monolith/internal/core"
)

// CreateGroup creates a new group with an invite code
func (s *Store) CreateGroup(name, inviteCode string) (*core.Group, error) {
	result, err := s.DB.Exec(
		"INSERT INTO groups (name, invite_code) VALUES (?, ?)",
		name, inviteCode,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetGroupByID(id)
}

// GetGroupByID retrieves a group by ID
func (s *Store) GetGroupByID(id int64) (*core.Group, error) {
	group := &core.Group{}

	err := s.DB.QueryRow(
		"SELECT id, name, invite_code, created_at FROM groups WHERE id = ?",
		id,
	).Scan(&group.ID, &group.Name, &group.InviteCode, &group.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("group not found")
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	return group, nil
}

// GetGroupByInviteCode retrieves a group by invite code
func (s *Store) GetGroupByInviteCode(inviteCode string) (*core.Group, error) {
	group := &core.Group{}

	err := s.DB.QueryRow(
		"SELECT id, name, invite_code, created_at FROM groups WHERE invite_code = ?",
		inviteCode,
	).Scan(&group.ID, &group.Name, &group.InviteCode, &group.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("group not found")
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	return group, nil
}

// GetGroupsByUserID retrieves all groups a user is a member of
func (s *Store) GetGroupsByUserID(userID int64) ([]*core.Group, error) {
	rows, err := s.DB.Query(`
		SELECT g.id, g.name, g.invite_code, g.created_at
		FROM groups g
		INNER JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	defer rows.Close()

	var groups []*core.Group
	for rows.Next() {
		group := &core.Group{}
		if err := rows.Scan(&group.ID, &group.Name, &group.InviteCode, &group.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// AddUserToGroup adds a user to a group
func (s *Store) AddUserToGroup(userID, groupID int64) error {
	_, err := s.DB.Exec(
		"INSERT INTO group_members (user_id, group_id) VALUES (?, ?)",
		userID, groupID,
	)
	if err != nil {
		return fmt.Errorf("failed to add user to group: %w", err)
	}
	return nil
}

// IsUserInGroup checks if a user is a member of a group
func (s *Store) IsUserInGroup(userID, groupID int64) (bool, error) {
	var count int
	err := s.DB.QueryRow(
		"SELECT COUNT(*) FROM group_members WHERE user_id = ? AND group_id = ?",
		userID, groupID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check group membership: %w", err)
	}
	return count > 0, nil
}
