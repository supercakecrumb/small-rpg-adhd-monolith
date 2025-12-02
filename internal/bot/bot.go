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
	"small-rpg-adhd-monolith/internal/i18n"

	tele "gopkg.in/telebot.v3"
)

// Bot represents the Telegram bot
type Bot struct {
	bot           *tele.Bot
	service       *core.Service
	publicURL     string
	sessionSecret string
	token         string
	translator    *i18n.Translator
}

func normalizeLang(lang string) string {
	lang = strings.ToLower(lang)
	if strings.HasPrefix(lang, "ru") {
		return "ru"
	}
	return "en"
}

func (b *Bot) lang(ctx tele.Context, user *core.User) string {
	if user != nil && user.Language != "" {
		return normalizeLang(user.Language)
	}
	if ctx != nil && ctx.Sender() != nil && ctx.Sender().LanguageCode != "" {
		if strings.HasPrefix(strings.ToLower(ctx.Sender().LanguageCode), "ru") {
			return "ru"
		}
	}
	return "en"
}

func (b *Bot) t(lang, key string) string {
	if b.translator == nil {
		return key
	}
	return b.translator.T(lang, key)
}

func (b *Bot) languageKeyboard(lang string) *tele.ReplyMarkup {
	lang = normalizeLang(lang)
	btnEn := tele.InlineButton{Text: b.t(lang, "bot.switch.en"), Data: "lang:en"}
	btnRu := tele.InlineButton{Text: b.t(lang, "bot.switch.ru"), Data: "lang:ru"}
	markup := &tele.ReplyMarkup{InlineKeyboard: [][]tele.InlineButton{{btnEn, btnRu}}}
	return markup
}

func (b *Bot) handleLanguageSelection(c tele.Context, lang string) error {
	lang = normalizeLang(lang)
	user, err := b.service.GetUserByTelegramID(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌"})
	}
	_ = b.service.SetUserLanguage(user.ID, lang)
	user.Language = lang
	markup := b.languageKeyboard(lang)
	if err := c.Edit(b.t(lang, "bot.switch.prompt"), markup); err != nil {
		log.Printf("failed to edit language message: %v", err)
	}
	return c.Respond(&tele.CallbackResponse{})
}

// NewBot creates a new Bot instance
func NewBot(token string, service *core.Service, sessionSecret string, translator *i18n.Translator) (*Bot, error) {
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
		translator:    translator,
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
	b.bot.Handle("/switch_language", b.handleSwitchLanguage)

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
	langFromTg := b.lang(c, nil)

	// Check if user already exists
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err == nil && user != nil {
		// User exists, update profile photo
		b.updateUserPhoto(user.ID, telegramID)

		// Welcome them back
		// If language not set, adopt from Telegram
		if user.Language == "" && langFromTg != "" {
			_ = b.service.SetUserLanguage(user.ID, langFromTg)
			user.Language = langFromTg
		}
		return c.Send(fmt.Sprintf(b.t(user.Language, "bot.start.returning"), user.Username))
	}

	// User doesn't exist, create new user
	telegramIDPtr := &telegramID
	newUser, err := b.service.CreateUser(username, telegramIDPtr, langFromTg)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return c.Send("❌ Oops! Something went wrong creating your account. Try again?")
	}

	// Fetch and cache profile photo for new user
	b.updateUserPhoto(newUser.ID, telegramID)

	return c.Send(fmt.Sprintf(b.t(langFromTg, "bot.start.new"), newUser.Username))
}

// handleWeb handles the /web command
func (b *Bot) handleWeb(c tele.Context) error {
	telegramID := c.Sender().ID

	// Get user by Telegram ID
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		return c.Send(b.t(b.lang(c, nil), "bot.web.unknown"))
	}
	lang := b.lang(c, user)

	// Generate login hash
	loginHash := b.generateLoginHash(user.Username)
	loginURL := fmt.Sprintf("%s/auth?user=%s&hash=%s", b.publicURL, user.Username, loginHash)

	return c.Send(fmt.Sprintf(b.t(lang, "bot.web.access"), loginURL))
}

// generateLoginHash generates an HMAC-SHA256 hash for username
func (b *Bot) generateLoginHash(username string) string {
	h := hmac.New(sha256.New, []byte(b.sessionSecret))
	h.Write([]byte(username))
	return hex.EncodeToString(h.Sum(nil))
}

// handleHelp handles the /help command
func (b *Bot) handleHelp(c tele.Context) error {
	var user *core.User
	if u, err := b.service.GetUserByTelegramID(c.Sender().ID); err == nil {
		user = u
	}
	lang := b.lang(c, user)
	return c.Send(b.t(lang, "bot.help"))
}

func (b *Bot) handleSwitchLanguage(c tele.Context) error {
	user, err := b.service.GetUserByTelegramID(c.Sender().ID)
	if err != nil {
		return c.Send("❌")
	}
	lang := b.lang(c, user)
	markup := b.languageKeyboard(lang)
	return c.Send(b.t(lang, "bot.switch.prompt"), markup)
}

// handleBalance handles the /balance command
func (b *Bot) handleBalance(c tele.Context) error {
	telegramID := c.Sender().ID

	// Get user
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		return c.Send(b.t("en", "bot.web.unknown"))
	}
	lang := b.lang(c, user)

	// Get all groups for this user
	groups, err := b.service.GetGroupsByUserID(user.ID)
	if err != nil {
		log.Printf("Error getting groups: %v", err)
		return c.Send(b.t(lang, "bot.error.groups"))
	}

	if len(groups) == 0 {
		return c.Send(fmt.Sprintf(b.t(lang, "bot.balance.empty"), b.publicURL))
	}

	// Build balance message
	var msg strings.Builder
	msg.WriteString(b.t(lang, "bot.balance.header"))
	msg.WriteString("\n\n")

	totalCoins := 0
	for _, group := range groups {
		balance, err := b.service.GetBalance(user.ID, group.ID)
		if err != nil {
			log.Printf("Error getting balance for group %d: %v", group.ID, err)
			continue
		}
		totalCoins += balance
		msg.WriteString(fmt.Sprintf(b.t(lang, "bot.balance.line"), group.Name, balance))
		msg.WriteString("\n")
	}

	msg.WriteString(fmt.Sprintf("\n"+b.t(lang, "bot.balance.total"), totalCoins))

	if totalCoins == 0 {
		msg.WriteString("\n\n" + b.t(lang, "bot.balance.tip"))
	} else if totalCoins >= 100 {
		msg.WriteString("\n\n" + b.t(lang, "bot.balance.celebrate"))
	}

	return c.Send(msg.String())
}

// handleTasks handles the /tasks command
func (b *Bot) handleTasks(c tele.Context) error {
	telegramID := c.Sender().ID

	// Get user
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		return c.Send(b.t("en", "bot.web.unknown"))
	}
	lang := b.lang(c, user)

	// Get all groups for this user
	groups, err := b.service.GetGroupsByUserID(user.ID)
	if err != nil {
		log.Printf("Error getting groups: %v", err)
		return c.Send(b.t(lang, "bot.error.groups"))
	}

	if len(groups) == 0 {
		return c.Send(fmt.Sprintf(b.t(lang, "bot.tasks.empty"), b.publicURL))
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

	return c.Send(b.t(lang, "bot.tasks.choose"), markup)
}

// handleCallback handles all inline button callbacks
func (b *Bot) handleCallback(c tele.Context) error {
	data := c.Callback().Data

	if strings.HasPrefix(data, "lang:") {
		return b.handleLanguageSelection(c, strings.TrimPrefix(data, "lang:"))
	}

	// Handle notification action buttons (format: notify_action_id or notify_reschedule_id_minutes)
	if strings.HasPrefix(data, "notify_") {
		return b.handleNotificationCallback(c, data)
	}

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

// handleNotificationCallback routes notification action button callbacks
func (b *Bot) handleNotificationCallback(c tele.Context, data string) error {
	// Parse callback data: notify_action_id or notify_reschedule_id_minutes
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		log.Printf("Invalid notification callback data: %s", data)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Invalid notification action"})
	}

	action := parts[1] // done, snooze, later, reschedule
	notifIDStr := parts[2]
	notifID, err := strconv.ParseInt(notifIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid notification ID: %s", notifIDStr)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Invalid notification ID"})
	}

	// Route to appropriate handler
	switch action {
	case "done":
		return b.handleNotifyDone(c, notifID)
	case "snooze":
		return b.handleNotifySnooze(c, notifID)
	case "later":
		return b.handleNotifyLater(c, notifID)
	case "reschedule":
		// For reschedule, we need the minutes parameter
		if len(parts) < 4 {
			log.Printf("Missing minutes parameter for reschedule")
			return c.Respond(&tele.CallbackResponse{Text: "❌ Invalid reschedule parameters"})
		}
		minutesStr := parts[3]
		minutes, err := strconv.Atoi(minutesStr)
		if err != nil {
			log.Printf("Invalid minutes value: %s", minutesStr)
			return c.Respond(&tele.CallbackResponse{Text: "❌ Invalid time duration"})
		}
		return b.handleNotifyReschedule(c, notifID, minutes)
	default:
		log.Printf("Unknown notification action: %s", action)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Unknown notification action"})
	}
}

// handleNotifyDone handles the "Done" button - marks task as completed
func (b *Bot) handleNotifyDone(c tele.Context, notificationID int64) error {
	telegramID := c.Sender().ID

	// Get user
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("Error getting user for notification done: %v", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ User not found"})
	}

	// Get notification from DB
	notification, err := b.service.GetNotificationByID(notificationID)
	if err != nil {
		log.Printf("Error getting notification %d: %v", notificationID, err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Notification not found"})
	}

	// Get task from DB
	task, err := b.service.GetTaskByID(notification.TaskID)
	if err != nil {
		log.Printf("Error getting task %d: %v", notification.TaskID, err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Task not found"})
	}

	// Complete the task
	transaction, err := b.service.CompleteTask(user.ID, task.ID, nil)
	if err != nil {
		log.Printf("Error completing task %d: %v", task.ID, err)
		return c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("❌ %v", err)})
	}

	// Edit message to show completion (remove keyboard)
	successMsg := fmt.Sprintf(
		"✅ Task completed: %s\n\n"+
			"💰 +%d coins earned!\n\n"+
			"Great job! Keep up the momentum! 💪",
		task.Title,
		transaction.Amount,
	)

	err = c.Edit(successMsg)
	if err != nil {
		log.Printf("Error editing message after task completion: %v", err)
	}

	// Answer callback query with success message
	return c.Respond(&tele.CallbackResponse{
		Text: fmt.Sprintf("✅ +%d coins!", transaction.Amount),
	})
}

// handleNotifySnooze handles the "Snooze" button - reschedules notification for default snooze duration
func (b *Bot) handleNotifySnooze(c tele.Context, notificationID int64) error {
	telegramID := c.Sender().ID

	// Get user
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("Error getting user for notification snooze: %v", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ User not found"})
	}

	// Get notification from DB
	notification, err := b.service.GetNotificationByID(notificationID)
	if err != nil {
		log.Printf("Error getting notification %d: %v", notificationID, err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Notification not found"})
	}

	// Get task from DB
	task, err := b.service.GetTaskByID(notification.TaskID)
	if err != nil {
		log.Printf("Error getting task %d: %v", notification.TaskID, err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Task not found"})
	}

	// Get user's notification settings to get snooze_default_minutes
	// TODO: Load duration options from user's notification settings once user settings UI is implemented
	settings, err := b.service.GetNotificationSettings(user.ID)
	if err != nil {
		log.Printf("Error getting notification settings: %v", err)
		// Use default if can't get settings
		settings = &core.NotificationSettings{
			SnoozeDefaultMinutes: 15,
		}
	}

	// Create snooze notification helper
	if err := b.createSnoozeNotification(task.ID, user.ID, settings.SnoozeDefaultMinutes); err != nil {
		log.Printf("Error creating snooze notification: %v", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Failed to snooze notification"})
	}

	// Edit message to show snooze confirmation (remove keyboard)
	snoozeMsg := fmt.Sprintf(
		"⏰ Snoozed for %d minutes\n\n"+
			"Task: %s\n\n"+
			"I'll remind you again soon! 💤",
		settings.SnoozeDefaultMinutes,
		task.Title,
	)

	err = c.Edit(snoozeMsg)
	if err != nil {
		log.Printf("Error editing message after snooze: %v", err)
	}

	// Answer callback query
	return c.Respond(&tele.CallbackResponse{
		Text: fmt.Sprintf("⏰ Snoozed for %d minutes", settings.SnoozeDefaultMinutes),
	})
}

// handleNotifyLater handles the "Remind Later" button - shows duration selection menu
func (b *Bot) handleNotifyLater(c tele.Context, notificationID int64) error {
	// Get notification from DB
	notification, err := b.service.GetNotificationByID(notificationID)
	if err != nil {
		log.Printf("Error getting notification %d: %v", notificationID, err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Notification not found"})
	}

	// Get task from DB
	task, err := b.service.GetTaskByID(notification.TaskID)
	if err != nil {
		log.Printf("Error getting task %d: %v", notification.TaskID, err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Task not found"})
	}

	// TODO: Load duration options from user's notification settings once user settings UI is implemented
	// For now, use hardcoded default durations
	durationOptions := []struct {
		label   string
		minutes int
	}{
		{"30 minutes", 30},
		{"1 hour", 60},
		{"2 hours", 120},
		{"4 hours", 240},
		{"Tomorrow (24h)", 1440},
	}

	// Create inline keyboard with duration options
	var rows [][]tele.InlineButton
	for _, opt := range durationOptions {
		btn := tele.InlineButton{
			Text: opt.label,
			Data: fmt.Sprintf("notify_reschedule_%d_%d", notificationID, opt.minutes),
		}
		rows = append(rows, []tele.InlineButton{btn})
	}

	markup := &tele.ReplyMarkup{InlineKeyboard: rows}

	// Edit message to show duration selection (keep task info, change keyboard)
	laterMsg := fmt.Sprintf(
		"🔔 Remind me later about:\n\n"+
			"%s\n\n"+
			"When should I remind you?",
		task.Title,
	)

	err = c.Edit(laterMsg, markup)
	if err != nil {
		log.Printf("Error editing message for later options: %v", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Failed to show options"})
	}

	// Answer callback query
	return c.Respond(&tele.CallbackResponse{
		Text: "Choose a time",
	})
}

// handleNotifyReschedule handles duration selection from "Remind Later" menu
func (b *Bot) handleNotifyReschedule(c tele.Context, notificationID int64, minutes int) error {
	telegramID := c.Sender().ID

	// Get user
	user, err := b.service.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("Error getting user for notification reschedule: %v", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ User not found"})
	}

	// Get notification from DB
	notification, err := b.service.GetNotificationByID(notificationID)
	if err != nil {
		log.Printf("Error getting notification %d: %v", notificationID, err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Notification not found"})
	}

	// Get task from DB
	task, err := b.service.GetTaskByID(notification.TaskID)
	if err != nil {
		log.Printf("Error getting task %d: %v", notification.TaskID, err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Task not found"})
	}

	// Create snooze notification with specified duration
	if err := b.createSnoozeNotification(task.ID, user.ID, minutes); err != nil {
		log.Printf("Error creating snooze notification: %v", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ Failed to schedule reminder"})
	}

	// Format duration for display
	durationStr := formatDuration(minutes)

	// Edit message to show confirmation (remove keyboard)
	rescheduleMsg := fmt.Sprintf(
		"🔔 Reminder scheduled!\n\n"+
			"Task: %s\n\n"+
			"I'll remind you in %s ⏰",
		task.Title,
		durationStr,
	)

	err = c.Edit(rescheduleMsg)
	if err != nil {
		log.Printf("Error editing message after reschedule: %v", err)
	}

	// Answer callback query
	return c.Respond(&tele.CallbackResponse{
		Text: fmt.Sprintf("🔔 Reminder set for %s", durationStr),
	})
}

// createSnoozeNotification creates a snooze notification for a task
// This is a helper method used by both snooze and reschedule handlers
func (b *Bot) createSnoozeNotification(taskID, userID int64, delayMinutes int) error {
	scheduledAt := time.Now().Add(time.Duration(delayMinutes) * time.Minute)

	notification := &core.TaskNotification{
		TaskID:           taskID,
		UserID:           userID,
		NotificationType: "snooze",
		ScheduledAt:      scheduledAt,
	}

	return b.service.CreateNotification(notification)
}

// formatDuration formats minutes into a human-readable string
func formatDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	} else if minutes < 1440 {
		hours := minutes / 60
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	} else {
		days := minutes / 1440
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
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
		return c.Send(b.t("en", "bot.web.unknown"))
	}
	lang := b.lang(c, user)

	// Get current notification status
	profile, err := b.service.GetUserProfile(user.ID)
	currentStatus := b.t(lang, "bot.notifications.status.disabled")
	if err == nil && profile != nil && profile.NotificationEnabled {
		currentStatus = b.t(lang, "bot.notifications.status.enabled")
	}

	// Create inline keyboard for notification toggle
	btnEnable := tele.InlineButton{
		Text: b.t(lang, "bot.notifications.enable"),
		Data: "notif:enable",
	}
	btnDisable := tele.InlineButton{
		Text: b.t(lang, "bot.notifications.disable"),
		Data: "notif:disable",
	}

	markup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{btnEnable},
			{btnDisable},
		},
	}

	return c.Send(fmt.Sprintf(b.t(lang, "bot.notifications.header"), currentStatus), markup)
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

// SendNotification sends a notification message with inline buttons to a Telegram user
// This implements the core.BotNotifier interface
func (b *Bot) SendNotification(chatID int64, message string, buttons map[string]string) error {
	// Create inline keyboard from buttons map
	var rows [][]tele.InlineButton

	// Add buttons in the order: Done, Snooze, Later (left to right)
	buttonOrder := []string{"✅ Done", "⏰ Will do in 15 mins", "🔔 Remind later"}
	var row []tele.InlineButton

	for _, btnText := range buttonOrder {
		if callbackData, exists := buttons[btnText]; exists {
			btn := tele.InlineButton{
				Text: btnText,
				Data: callbackData,
			}
			row = append(row, btn)
		}
	}

	if len(row) > 0 {
		rows = append(rows, row)
	}

	markup := &tele.ReplyMarkup{InlineKeyboard: rows}

	// Send message to user
	_, err := b.bot.Send(&tele.User{ID: chatID}, message, markup)
	return err
}
