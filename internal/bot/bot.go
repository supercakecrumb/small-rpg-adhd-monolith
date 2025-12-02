package bot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"small-rpg-adhd-monolith/internal/core"

	tele "gopkg.in/telebot.v3"
)

// Bot represents the Telegram bot
type Bot struct {
	bot           *tele.Bot
	service       *core.Service
	publicURL     string
	sessionSecret string
	token         string
}

// NewBot creates a new Bot instance
func NewBot(token string, service *core.Service, sessionSecret string) (*Bot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	// Get public URL from environment
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:8080"
		log.Printf("⚠️ PUBLIC_URL not set, using default: %s", publicURL)
	} else {
		log.Printf("✅ PUBLIC_URL configured: %s", publicURL)
	}

	bot := &Bot{
		bot:           b,
		service:       service,
		publicURL:     publicURL,
		sessionSecret: sessionSecret,
		token:         token,
	}

	bot.setupHandlers()
	return bot, nil
}

// Start starts the bot polling
func (b *Bot) Start() {
	log.Println("🤖 Telegram bot is now running...")
	b.bot.Start()
}

// Stop gracefully stops the bot
func (b *Bot) Stop() {
	b.bot.Stop()
}

// setupHandlers configures all command and callback handlers
func (b *Bot) setupHandlers() {
	// Command handlers
	b.bot.Handle("/start", b.handleStart)
	b.bot.Handle("/web", b.handleWeb)
	b.bot.Handle("/help", b.handleHelp)
	b.bot.Handle("/balance", b.handleBalance)
	b.bot.Handle("/tasks", b.handleTasks)
	b.bot.Handle("/notifications", b.handleNotifications)

	// Callback handlers
	b.bot.Handle(tele.OnCallback, b.handleCallback)
}

// handleStart handles the /start command
func (b *Bot) handleStart(c tele.Context) error {
	telegramID := c.Sender().ID
	username := c.Sender().Username
	if username == "" {
		username = c.Sender().FirstName
	}

	// Check if user already exists
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err == nil && user != nil {
		// User exists, update profile photo
		b.updateUserPhoto(user.ID, telegramID)

		// Welcome them back
		return c.Send(fmt.Sprintf(
			"🎮 Welcome back, %s! Ready to conquer some tasks?\n\n"+
				"Quick commands:\n"+
				"💰 /balance - Check your coins\n"+
				"📋 /tasks - Complete tasks & earn rewards\n"+
				"🌐 /web - Access the Web UI\n"+
				"🔔 /notifications - Manage notifications\n"+
				"❓ /help - Show all commands\n\n"+
				"Let's get those dopamine hits! 🚀",
			user.Username,
		))
	}

	// User doesn't exist, create new user
	telegramIDPtr := &telegramID
	newUser, err := b.service.CreateUser(username, telegramIDPtr)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return c.Send("❌ Oops! Something went wrong creating your account. Try again?")
	}

	// Fetch and cache profile photo for new user
	b.updateUserPhoto(newUser.ID, telegramID)

	return c.Send(fmt.Sprintf(
		"🎉 Welcome to the ADHD Quest System, %s!\n\n"+
			"You've just unlocked:\n"+
			"✨ A gamified way to crush your tasks\n"+
			"🪙 Coin rewards for every win\n"+
			"🎯 Group challenges with friends\n\n"+
			"💡 Quick start:\n"+
			"1. Use /web to access the Web UI\n"+
			"2. Create or join a group\n"+
			"3. Use /tasks to start earning coins\n"+
			"4. Level up your productivity! 🚀\n\n"+
			"Need help? Type /help for all commands\n"+
			"Pro tip: Small wins add up to big victories! 💪",
		newUser.Username,
	))
}

// handleWeb handles the /web command
func (b *Bot) handleWeb(c tele.Context) error {
	telegramID := c.Sender().ID

	// Get user by Telegram ID
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		return c.Send(
			"❌ I don't know you yet! Please use /start first to register.",
		)
	}

	// Generate login hash
	loginHash := b.generateLoginHash(user.Username)
	loginURL := fmt.Sprintf("%s/auth?user=%s&hash=%s", b.publicURL, user.Username, loginHash)

	return c.Send(fmt.Sprintf(
		"🌐 Web UI Access\n\n"+
			"Click the link below to log in:\n"+
			"🔗 %s\n\n"+
			"📝 This secure link will:\n"+
			"• Log you into the web interface automatically\n"+
			"• Give you access to all your groups and tasks\n"+
			"• Let you manage tasks, shop items, and more\n\n"+
			"⚠️ Security note:\n"+
			"This link is unique to you and should not be shared.\n"+
			"It will remain valid until you request a new one.\n\n"+
			"💡 Tip: Use the web UI to manage your groups,\n"+
			"then come back here to quickly complete tasks! ✨",
		loginURL,
	))
}

// generateLoginHash generates an HMAC-SHA256 hash for username
func (b *Bot) generateLoginHash(username string) string {
	h := hmac.New(sha256.New, []byte(b.sessionSecret))
	h.Write([]byte(username))
	return hex.EncodeToString(h.Sum(nil))
}

// handleHelp handles the /help command
func (b *Bot) handleHelp(c tele.Context) error {
	return c.Send(
		"🤖 RatPG - Command Guide\n\n" +
			"Basic Commands:\n" +
			"🏁 /start - Register & get started\n" +
			"❓ /help - Show this help message\n" +
			"🌐 /web - Get Web UI access link\n\n" +
			"Game Commands:\n" +
			"💰 /balance - Check your coin balance\n" +
			"📋 /tasks - Browse & complete tasks\n" +
			"🔔 /notifications - Manage notifications\n\n" +
			"How it works:\n" +
			"1. Create or join groups via the Web UI\n" +
			"2. Tasks and shop items are managed on the web\n" +
			"3. Use the bot for quick task completion\n" +
			"4. Earn coins and spend them in the shop!\n\n" +
			"Need more help? Visit the Web UI for full features! 🚀",
	)
}

// handleBalance handles the /balance command
func (b *Bot) handleBalance(c tele.Context) error {
	telegramID := c.Sender().ID

	// Get user
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		return c.Send("❌ Hmm, I don't know you yet! Use /start to get registered.")
	}

	// Get all groups for this user
	groups, err := b.service.GetGroupsByUserID(user.ID)
	if err != nil {
		log.Printf("Error getting groups: %v", err)
		return c.Send("❌ Couldn't fetch your groups. Try again?")
	}

	if len(groups) == 0 {
		return c.Send(fmt.Sprintf(
			"🏜️ You're not in any groups yet!\n\n"+
				"Head over to the Web UI to:\n"+
				"• Create your own group\n"+
				"• Join existing groups with invite codes\n\n"+
				"Access the web at:\n"+
				"🔗 %s\n\n"+
				"Then come back here to start earning those coins! 💰\n\n"+
				"Type /web for more info about the Web UI",
			b.publicURL,
		))
	}

	// Build balance message
	var msg strings.Builder
	msg.WriteString("💰 Your Coin Balance:\n\n")

	totalCoins := 0
	for _, group := range groups {
		balance, err := b.service.GetBalance(user.ID, group.ID)
		if err != nil {
			log.Printf("Error getting balance for group %d: %v", group.ID, err)
			continue
		}
		totalCoins += balance
		msg.WriteString(fmt.Sprintf("🏷️ %s: %d coins\n", group.Name, balance))
	}

	msg.WriteString(fmt.Sprintf("\n✨ Total: %d coins across all groups!\n", totalCoins))

	if totalCoins == 0 {
		msg.WriteString("\n💡 Complete tasks with /tasks to start earning!")
	} else if totalCoins >= 100 {
		msg.WriteString("\n🎉 Wow! You're crushing it! Keep going!")
	}

	return c.Send(msg.String())
}

// handleTasks handles the /tasks command
func (b *Bot) handleTasks(c tele.Context) error {
	telegramID := c.Sender().ID

	// Get user
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		return c.Send("❌ I don't know you yet! Use /start to get registered.")
	}

	// Get all groups for this user
	groups, err := b.service.GetGroupsByUserID(user.ID)
	if err != nil {
		log.Printf("Error getting groups: %v", err)
		return c.Send("❌ Couldn't fetch your groups. Try again?")
	}

	if len(groups) == 0 {
		return c.Send(fmt.Sprintf(
			"🏜️ No groups yet!\n\n"+
				"Access the Web UI at:\n"+
				"🔗 %s\n\n"+
				"Join or create a group, then come back here to complete tasks! 🎯\n\n"+
				"Type /web for more info",
			b.publicURL,
		))
	}

	// Create inline keyboard with group buttons
	var rows [][]tele.InlineButton
	for _, group := range groups {
		btn := tele.InlineButton{
			Text: fmt.Sprintf("📁 %s", group.Name),
			Data: fmt.Sprintf("group:%d", group.ID),
		}
		rows = append(rows, []tele.InlineButton{btn})
	}

	markup := &tele.ReplyMarkup{InlineKeyboard: rows}

	return c.Send(
		"🎯 Choose a group to see available tasks:\n\n"+
			"Pick one and let's earn some coins! 💪",
		markup,
	)
}

// handleCallback handles all inline button callbacks
func (b *Bot) handleCallback(c tele.Context) error {
	data := c.Callback().Data

	// Parse callback data
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Invalid action"})
	}

	action := parts[0]
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Invalid ID"})
	}

	switch action {
	case "group":
		return b.handleGroupSelection(c, id)
	case "task":
		return b.handleTaskCompletion(c, id)
	case "back_tasks":
		return b.handleTasks(c)
	case "notif":
		return b.handleNotificationToggle(c, parts[1])
	default:
		return c.Respond(&tele.CallbackResponse{Text: "❌ Unknown action"})
	}
}

// handleNotificationToggle handles notification preference toggling
func (b *Bot) handleNotificationToggle(c tele.Context, action string) error {
	telegramID := c.Sender().ID

	// Get user
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ User not found"})
	}

	// Set notification preference
	enabled := action == "enable"
	err = b.service.SetNotificationEnabled(user.ID, enabled)
	if err != nil {
		log.Printf("Failed to update notification preference: %v", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Failed to update settings"})
	}

	// Update message
	status := "disabled"
	emoji := "🔕"
	if enabled {
		status = "enabled"
		emoji = "✅"
	}

	c.Edit(fmt.Sprintf(
		"%s Notifications %s!\n\n"+
			"You can change this anytime with /notifications",
		emoji,
		status,
	))

	return c.Respond(&tele.CallbackResponse{
		Text: fmt.Sprintf("Notifications %s!", status),
	})
}

// handleGroupSelection shows tasks for a selected group
func (b *Bot) handleGroupSelection(c tele.Context, groupID int64) error {
	telegramID := c.Sender().ID

	// Get user
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ User not found"})
	}

	// Get group
	group, err := b.service.GetGroupByID(groupID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Group not found"})
	}

	// Get tasks for this group
	tasks, err := b.service.GetTasksByGroupID(groupID)
	if err != nil {
		log.Printf("Error getting tasks: %v", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Couldn't fetch tasks"})
	}

	if len(tasks) == 0 {
		return c.Edit(fmt.Sprintf(
			"📭 No tasks in %s yet!\n\n"+
				"Create some tasks via the Web UI:\n"+
				"🔗 %s\n\n"+
				"Type /web for more info",
			group.Name,
			b.publicURL,
		))
	}

	// Create inline keyboard with task buttons
	var rows [][]tele.InlineButton
	for _, task := range tasks {
		rewardEmoji := "🪙"
		if task.RewardValue >= 10 {
			rewardEmoji = "💰"
		}

		btn := tele.InlineButton{
			Text: fmt.Sprintf("%s %s (+%d)", rewardEmoji, task.Title, task.RewardValue),
			Data: fmt.Sprintf("task:%d", task.ID),
		}
		rows = append(rows, []tele.InlineButton{btn})
	}

	// Add back button
	backBtn := tele.InlineButton{
		Text: "⬅️ Back to Groups",
		Data: "back_tasks:0",
	}
	rows = append(rows, []tele.InlineButton{backBtn})

	markup := &tele.ReplyMarkup{InlineKeyboard: rows}

	// Get current balance for this group
	balance, _ := b.service.GetBalance(user.ID, groupID)

	return c.Edit(
		fmt.Sprintf(
			"📋 Tasks in %s\n"+
				"💰 Current balance: %d coins\n\n"+
				"Click a task to complete it and earn coins! 🚀",
			group.Name,
			balance,
		),
		markup,
	)
}

// handleTaskCompletion handles task completion
func (b *Bot) handleTaskCompletion(c tele.Context, taskID int64) error {
	telegramID := c.Sender().ID

	// Get user
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ User not found"})
	}

	// Get task details
	task, err := b.service.GetTaskByID(taskID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Task not found"})
	}

	// Complete the task (for boolean tasks, no quantity needed)
	transaction, err := b.service.CompleteTask(user.ID, taskID, nil)
	if err != nil {
		log.Printf("Error completing task: %v", err)
		return c.Respond(&tele.CallbackResponse{
			Text: fmt.Sprintf("❌ Error: %v", err),
		})
	}

	// Get updated balance
	balance, _ := b.service.GetBalance(user.ID, task.GroupID)

	// Success response
	responseMsg := fmt.Sprintf(
		"🎉 Task completed!\n\n"+
			"✅ %s\n"+
			"💰 +%d coins earned!\n"+
			"🏆 New balance: %d coins\n\n"+
			"Keep it up! Every task is a victory! 💪",
		task.Title,
		transaction.Amount,
		balance,
	)

	// Update the message
	c.Edit(responseMsg)

	// Send callback response
	err = c.Respond(&tele.CallbackResponse{
		Text: fmt.Sprintf("✨ +%d coins!", transaction.Amount),
	})

	// Send notifications to other group members
	b.notifyGroupMembers(task.GroupID, user.ID, fmt.Sprintf(
		"🎉 %s completed: %s (+%d coins)",
		user.Username,
		task.Title,
		transaction.Amount,
	))

	return err
}

// updateUserPhoto fetches and caches the user's Telegram profile photo
func (b *Bot) updateUserPhoto(userID int64, telegramID int64) {
	// Note: Profile photo fetching requires direct API access
	// For now, we skip this functionality as telebot v3 doesn't provide ProfilePhotos method
	// TODO: Implement using direct Telegram Bot API HTTP calls if needed
	log.Printf("Profile photo fetching not yet implemented for user %d", userID)
}

// handleNotifications handles the /notifications command
func (b *Bot) handleNotifications(c tele.Context) error {
	telegramID := c.Sender().ID

	// Get user
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		return c.Send("❌ I don't know you yet! Use /start to get registered.")
	}

	// Get current notification status
	profile, err := b.service.GetUserProfile(user.ID)
	currentStatus := "disabled"
	if err == nil && profile != nil && profile.NotificationEnabled {
		currentStatus = "enabled"
	}

	// Create inline keyboard for notification toggle
	btnEnable := tele.InlineButton{
		Text: "✅ Enable Notifications",
		Data: "notif:enable",
	}
	btnDisable := tele.InlineButton{
		Text: "🔕 Disable Notifications",
		Data: "notif:disable",
	}

	markup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{btnEnable},
			{btnDisable},
		},
	}

	return c.Send(
		fmt.Sprintf("🔔 Notification Settings\n\n"+
			"Current status: %s\n\n"+
			"When enabled, you'll receive notifications about:\n"+
			"• Task completions by group members\n"+
			"• Shop purchases in your groups\n"+
			"• Activity updates\n\n"+
			"Choose your preference:",
			currentStatus,
		),
		markup,
	)
}

// notifyGroupMembers sends notifications to all group members (except the actor)
func (b *Bot) notifyGroupMembers(groupID, actorUserID int64, message string) {
	// Get all members of the group
	members, err := b.service.GetUsersByGroupID(groupID)
	if err != nil {
		log.Printf("Failed to get group members for notification: %v", err)
		return
	}

	// Send notification to each member (except the actor)
	for _, member := range members {
		if member.ID == actorUserID {
			continue // Skip the person who performed the action
		}

		// Check if user has notifications enabled
		profile, err := b.service.GetUserProfile(member.ID)
		if err != nil || profile == nil || !profile.NotificationEnabled {
			continue // Skip if notifications disabled
		}

		// Check if user has a Telegram ID
		if member.TelegramID == nil {
			continue
		}

		// Send notification
		_, err = b.bot.Send(&tele.User{ID: *member.TelegramID}, message)
		if err != nil {
			log.Printf("Failed to send notification to user %d: %v", member.ID, err)
		}
	}
}

// NotifyPurchase sends a notification about a purchase
func (b *Bot) NotifyPurchase(groupID, buyerUserID int64, itemTitle string, cost int) {
	buyer, err := b.service.GetUserByID(buyerUserID)
	if err != nil {
		return
	}

	message := fmt.Sprintf(
		"🛍️ %s bought: %s (🪙 %d coins)",
		buyer.Username,
		itemTitle,
		cost,
	)

	b.notifyGroupMembers(groupID, buyerUserID, message)
}
