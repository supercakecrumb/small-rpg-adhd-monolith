package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"small-rpg-adhd-monolith/internal/core"

	tele "gopkg.in/telebot.v3"
)

// Bot represents the Telegram bot
type Bot struct {
	bot     *tele.Bot
	service *core.Service
}

// NewBot creates a new Bot instance
func NewBot(token string, service *core.Service) (*Bot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	bot := &Bot{
		bot:     b,
		service: service,
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
	b.bot.Handle("/balance", b.handleBalance)
	b.bot.Handle("/tasks", b.handleTasks)

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
		// User exists, welcome them back
		return c.Send(fmt.Sprintf(
			"🎮 Welcome back, %s! Ready to conquer some tasks?\n\n"+
				"💰 Use /balance to check your coins\n"+
				"📋 Use /tasks to complete tasks and earn rewards!\n\n"+
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

	return c.Send(fmt.Sprintf(
		"🎉 Welcome to the ADHD Quest System, %s!\n\n"+
			"You've just unlocked:\n"+
			"✨ A gamified way to crush your tasks\n"+
			"🪙 Coin rewards for every win\n"+
			"🎯 Group challenges with friends\n\n"+
			"💡 Quick start:\n"+
			"1. Join a group via the web interface\n"+
			"2. Use /tasks to start earning coins\n"+
			"3. Level up your productivity! 🚀\n\n"+
			"Pro tip: Small wins add up to big victories! 💪",
		newUser.Username,
	))
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
		return c.Send(
			"🏜️ You're not in any groups yet!\n\n" +
				"Head over to the web interface to:\n" +
				"• Create your own group\n" +
				"• Join existing groups with invite codes\n\n" +
				"Then come back here to start earning those coins! 💰",
		)
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
		return c.Send(
			"🏜️ No groups yet!\n\n" +
				"Join a group via the web interface first, then come back here to complete tasks! 🎯",
		)
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
	default:
		return c.Respond(&tele.CallbackResponse{Text: "❌ Unknown action"})
	}
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
		return c.Edit(
			fmt.Sprintf("📭 No tasks in %s yet!\n\nCreate some tasks via the web interface first.", group.Name),
		)
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
	return c.Respond(&tele.CallbackResponse{
		Text: fmt.Sprintf("✨ +%d coins!", transaction.Amount),
	})
}
