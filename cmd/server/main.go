package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"small-rpg-adhd-monolith/internal/bot"
	"small-rpg-adhd-monolith/internal/core"
	"small-rpg-adhd-monolith/internal/store"
	"small-rpg-adhd-monolith/internal/web"
)

func main() {
	// Get database path from environment or use default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "small-rpg.db"
	}

	// Get session secret from environment or use default (change in production!)
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "dev-secret-change-in-production"
		log.Println("Warning: Using default session secret. Set SESSION_SECRET environment variable in production!")
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Get Telegram bot token from environment
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	// Initialize the database store
	log.Println("Initializing database...")
	db, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Database initialized successfully")

	// Initialize the core service
	log.Println("Initializing service layer...")
	service := core.NewService(db)
	if service == nil {
		log.Fatal("Failed to initialize service")
	}

	log.Println("Service layer initialized successfully")

	// Initialize the web server
	log.Println("Initializing web server...")
	server, err := web.NewServer(service, sessionSecret)
	if err != nil {
		log.Fatalf("Failed to initialize web server: %v", err)
	}

	router := server.Router()

	// Initialize and start Telegram bot if token is provided
	if botToken != "" {
		log.Println("Initializing Telegram bot...")
		telegramBot, err := bot.NewBot(botToken, service)
		if err != nil {
			log.Printf("Warning: Failed to initialize Telegram bot: %v", err)
			log.Println("Continuing without Telegram bot...")
		} else {
			log.Println("Telegram bot initialized successfully")
			// Start bot in a goroutine
			go telegramBot.Start()
		}
	} else {
		log.Println("TELEGRAM_BOT_TOKEN not set, Telegram bot will not be started")
		log.Println("Set TELEGRAM_BOT_TOKEN environment variable to enable Telegram integration")
	}

	// Print startup information
	fmt.Println("\n✓ All components initialized successfully!")
	fmt.Println("✓ Database connection established")
	fmt.Println("✓ Schema migrations completed")
	fmt.Println("✓ Core service ready")
	fmt.Println("✓ Web server ready")
	if botToken != "" {
		fmt.Println("✓ Telegram bot ready")
	}
	fmt.Printf("\n🚀 Server starting on http://localhost:%s\n", port)
	fmt.Println("📝 Access the application at: http://localhost:" + port)
	if botToken != "" {
		fmt.Println("🤖 Telegram bot is running and ready to receive commands")
	}
	fmt.Println("\nPress Ctrl+C to stop the server")

	// Setup HTTP server with graceful shutdown
	addr := ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal is received
	sig := <-quit
	log.Printf("\nReceived signal %v, initiating graceful shutdown...", sig)

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	log.Println("✓ HTTP server stopped")
	log.Println("✓ Database connection closed")
	log.Println("Shutdown complete")
}
