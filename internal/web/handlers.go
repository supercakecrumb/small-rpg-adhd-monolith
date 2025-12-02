package web

import (
	"net/http"
	"strconv"

	"small-rpg-adhd-monolith/internal/core"

	"github.com/go-chi/chi/v5"
)

type dashboardData struct {
	Username string
	Groups   []*core.Group
	Error    string
}

type groupViewData struct {
	Username  string
	Group     *core.Group
	Tasks     []*core.Task
	ShopItems []*core.ShopItem
	Members   []*core.User
	Balance   int
	Error     string
	Success   string
}

// handleDashboard displays the user's dashboard with their groups
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.getUserID(r)

	user, err := s.service.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Failed to load user", http.StatusInternalServerError)
		return
	}

	groups, err := s.service.GetGroupsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to load groups", http.StatusInternalServerError)
		return
	}

	data := dashboardData{
		Username: user.Username,
		Groups:   groups,
	}

	s.renderTemplate(w, "dashboard.html", data)
}

// handleCreateGroup creates a new group
func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.getUserID(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	groupName := r.FormValue("name")
	if groupName == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	group, err := s.service.CreateGroup(groupName, userID)
	if err != nil {
		http.Error(w, "Failed to create group", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/groups/"+strconv.FormatInt(group.ID, 10), http.StatusSeeOther)
}

// handleJoinGroup joins a group using an invite code
func (s *Server) handleJoinGroup(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.getUserID(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	inviteCode := r.FormValue("invite_code")
	if inviteCode == "" {
		http.Error(w, "Invite code is required", http.StatusBadRequest)
		return
	}

	group, err := s.service.JoinGroup(userID, inviteCode)
	if err != nil {
		// Redirect back to dashboard with error
		http.Redirect(w, r, "/dashboard?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/groups/"+strconv.FormatInt(group.ID, 10), http.StatusSeeOther)
}

// handleGroupView displays a group's details
func (s *Server) handleGroupView(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.getUserID(r)

	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	// Get user
	user, err := s.service.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Failed to load user", http.StatusInternalServerError)
		return
	}

	// Get group
	group, err := s.service.GetGroupByID(groupID)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	// Get tasks
	tasks, err := s.service.GetTasksByGroupID(groupID)
	if err != nil {
		http.Error(w, "Failed to load tasks", http.StatusInternalServerError)
		return
	}

	// Get shop items
	shopItems, err := s.service.GetShopItemsByGroupID(groupID)
	if err != nil {
		http.Error(w, "Failed to load shop items", http.StatusInternalServerError)
		return
	}

	// Get members - we need to add this to the service
	members, err := s.service.GetUsersByGroupID(groupID)
	if err != nil {
		http.Error(w, "Failed to load members", http.StatusInternalServerError)
		return
	}

	// Get balance
	balance, err := s.service.GetBalance(userID, groupID)
	if err != nil {
		http.Error(w, "Failed to load balance", http.StatusInternalServerError)
		return
	}

	data := groupViewData{
		Username:  user.Username,
		Group:     group,
		Tasks:     tasks,
		ShopItems: shopItems,
		Members:   members,
		Balance:   balance,
		Success:   r.URL.Query().Get("success"),
		Error:     r.URL.Query().Get("error"),
	}

	s.renderTemplate(w, "group.html", data)
}

// handleCreateTask creates a new task in a group
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	taskTypeStr := r.FormValue("task_type")
	rewardValueStr := r.FormValue("reward_value")

	if title == "" || taskTypeStr == "" || rewardValueStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	taskType := core.TaskType(taskTypeStr)
	rewardValue, err := strconv.Atoi(rewardValueStr)
	if err != nil || rewardValue <= 0 {
		http.Error(w, "Invalid reward value", http.StatusBadRequest)
		return
	}

	_, err = s.service.CreateTask(groupID, title, description, taskType, rewardValue)
	if err != nil {
		http.Redirect(w, r, "/groups/"+groupIDStr+"?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/groups/"+groupIDStr+"?success=Task created", http.StatusSeeOther)
}

// handleCompleteTask completes a task
func (s *Server) handleCompleteTask(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.getUserID(r)

	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get task to determine group
	task, err := s.service.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Handle quantity for integer tasks
	var quantity *int
	if task.TaskType == core.TaskTypeInteger {
		quantityStr := r.FormValue("quantity")
		if quantityStr != "" {
			q, err := strconv.Atoi(quantityStr)
			if err != nil || q <= 0 {
				http.Redirect(w, r, "/groups/"+strconv.FormatInt(task.GroupID, 10)+"?error=Invalid quantity", http.StatusSeeOther)
				return
			}
			quantity = &q
		}
	}

	_, err = s.service.CompleteTask(userID, taskID, quantity)
	if err != nil {
		http.Redirect(w, r, "/groups/"+strconv.FormatInt(task.GroupID, 10)+"?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/groups/"+strconv.FormatInt(task.GroupID, 10)+"?success=Task completed!", http.StatusSeeOther)
}

// handleCreateShopItem creates a new shop item in a group
func (s *Server) handleCreateShopItem(w http.ResponseWriter, r *http.Request) {
	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	costStr := r.FormValue("cost")

	if title == "" || costStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	cost, err := strconv.Atoi(costStr)
	if err != nil || cost <= 0 {
		http.Error(w, "Invalid cost", http.StatusBadRequest)
		return
	}

	_, err = s.service.CreateShopItem(groupID, title, description, cost)
	if err != nil {
		http.Redirect(w, r, "/groups/"+groupIDStr+"?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/groups/"+groupIDStr+"?success=Shop item created", http.StatusSeeOther)
}

// handleBuyItem purchases an item from the shop
func (s *Server) handleBuyItem(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.getUserID(r)

	itemIDStr := chi.URLParam(r, "itemID")
	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	// Get item to determine group
	item, err := s.service.GetShopItemByID(itemID)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	_, err = s.service.BuyItem(userID, itemID)
	if err != nil {
		http.Redirect(w, r, "/groups/"+strconv.FormatInt(item.GroupID, 10)+"?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/groups/"+strconv.FormatInt(item.GroupID, 10)+"?success=Item purchased!", http.StatusSeeOther)
}

// handleUpdateTask updates an existing task
func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Get task to determine group for redirect
	task, err := s.service.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	taskTypeStr := r.FormValue("task_type")
	rewardValueStr := r.FormValue("reward_value")

	if title == "" || taskTypeStr == "" || rewardValueStr == "" {
		http.Redirect(w, r, "/groups/"+strconv.FormatInt(task.GroupID, 10)+"?error=Missing required fields", http.StatusSeeOther)
		return
	}

	taskType := core.TaskType(taskTypeStr)
	rewardValue, err := strconv.Atoi(rewardValueStr)
	if err != nil || rewardValue <= 0 {
		http.Redirect(w, r, "/groups/"+strconv.FormatInt(task.GroupID, 10)+"?error=Invalid reward value", http.StatusSeeOther)
		return
	}

	err = s.service.UpdateTask(taskID, title, description, taskType, rewardValue)
	if err != nil {
		http.Redirect(w, r, "/groups/"+strconv.FormatInt(task.GroupID, 10)+"?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/groups/"+strconv.FormatInt(task.GroupID, 10)+"?success=Task updated", http.StatusSeeOther)
}

// handleDeleteTask deletes a task
func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Get task to determine group for redirect
	task, err := s.service.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	err = s.service.DeleteTask(taskID)
	if err != nil {
		http.Redirect(w, r, "/groups/"+strconv.FormatInt(task.GroupID, 10)+"?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/groups/"+strconv.FormatInt(task.GroupID, 10)+"?success=Task deleted", http.StatusSeeOther)
}

// handleUpdateShopItem updates an existing shop item
func (s *Server) handleUpdateShopItem(w http.ResponseWriter, r *http.Request) {
	itemIDStr := chi.URLParam(r, "itemID")
	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	// Get item to determine group for redirect
	item, err := s.service.GetShopItemByID(itemID)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	costStr := r.FormValue("cost")

	if title == "" || costStr == "" {
		http.Redirect(w, r, "/groups/"+strconv.FormatInt(item.GroupID, 10)+"?error=Missing required fields", http.StatusSeeOther)
		return
	}

	cost, err := strconv.Atoi(costStr)
	if err != nil || cost <= 0 {
		http.Redirect(w, r, "/groups/"+strconv.FormatInt(item.GroupID, 10)+"?error=Invalid cost", http.StatusSeeOther)
		return
	}

	err = s.service.UpdateShopItem(itemID, title, description, cost)
	if err != nil {
		http.Redirect(w, r, "/groups/"+strconv.FormatInt(item.GroupID, 10)+"?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/groups/"+strconv.FormatInt(item.GroupID, 10)+"?success=Shop item updated", http.StatusSeeOther)
}

// handleDeleteShopItem deletes a shop item
func (s *Server) handleDeleteShopItem(w http.ResponseWriter, r *http.Request) {
	itemIDStr := chi.URLParam(r, "itemID")
	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	// Get item to determine group for redirect
	item, err := s.service.GetShopItemByID(itemID)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	err = s.service.DeleteShopItem(itemID)
	if err != nil {
		http.Redirect(w, r, "/groups/"+strconv.FormatInt(item.GroupID, 10)+"?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/groups/"+strconv.FormatInt(item.GroupID, 10)+"?success=Shop item deleted", http.StatusSeeOther)
}

type taskHistoryData struct {
	Username string
	Group    *core.Group
	History  []*core.TaskCompletionHistory
}

// handleTaskHistory displays task completion history
func (s *Server) handleTaskHistory(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.getUserID(r)

	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	user, err := s.service.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Failed to load user", http.StatusInternalServerError)
		return
	}

	group, err := s.service.GetGroupByID(groupID)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	history, err := s.service.GetTaskCompletionHistory(userID, groupID)
	if err != nil {
		http.Error(w, "Failed to load history", http.StatusInternalServerError)
		return
	}

	data := taskHistoryData{
		Username: user.Username,
		Group:    group,
		History:  history,
	}

	s.renderTemplate(w, "task_history.html", data)
}

type purchaseHistoryData struct {
	Username string
	Group    *core.Group
	History  []*core.PurchaseHistory
}

// handlePurchaseHistory displays purchase history
func (s *Server) handlePurchaseHistory(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.getUserID(r)

	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	user, err := s.service.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Failed to load user", http.StatusInternalServerError)
		return
	}

	group, err := s.service.GetGroupByID(groupID)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	history, err := s.service.GetPurchaseHistory(userID, groupID)
	if err != nil {
		http.Error(w, "Failed to load history", http.StatusInternalServerError)
		return
	}

	data := purchaseHistoryData{
		Username: user.Username,
		Group:    group,
		History:  history,
	}

	s.renderTemplate(w, "purchase_history.html", data)
}

// handleMarkPurchaseFulfilled marks a purchase as fulfilled
func (s *Server) handleMarkPurchaseFulfilled(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.getUserID(r)

	purchaseIDStr := chi.URLParam(r, "purchaseID")
	purchaseID, err := strconv.ParseInt(purchaseIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid purchase ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	notes := r.FormValue("notes")
	groupIDStr := r.FormValue("group_id")

	err = s.service.MarkPurchaseFulfilled(purchaseID, userID, notes)
	if err != nil {
		if groupIDStr != "" {
			http.Redirect(w, r, "/groups/"+groupIDStr+"/purchases?error="+err.Error(), http.StatusSeeOther)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if groupIDStr != "" {
		http.Redirect(w, r, "/groups/"+groupIDStr+"/purchases?success=Purchase marked as fulfilled", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}
