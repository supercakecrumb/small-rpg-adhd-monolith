package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
)

type loginPageData struct {
	Username string
	Error    string
}

// handleLoginPage displays the login page
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔑 handleLoginPage called: %s %s", r.Method, r.URL.Path)
	log.Printf("   Remote Address: %s", r.RemoteAddr)
	log.Printf("   User-Agent: %s", r.UserAgent())

	// Check if user is already logged in
	if userID, ok := s.getUserID(r); ok {
		log.Printf("   ✅ User already authenticated (ID: %d), redirecting to /dashboard", userID)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	log.Printf("   ❌ User not authenticated, rendering login page")
	s.renderTemplate(w, "login.html", loginPageData{})
}

// handleHashLogin processes hash-based login from Telegram bot
// URL format: /login?user=<username>&hash=<hmac>
func (s *Server) handleHashLogin(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔐 handleHashLogin called: %s %s", r.Method, r.URL.Path)

	// Check if user is already logged in
	if userID, ok := s.getUserID(r); ok {
		log.Printf("   ✅ User already authenticated (ID: %d), redirecting to /dashboard", userID)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	// Get username and hash from query parameters
	username := r.URL.Query().Get("user")
	providedHash := r.URL.Query().Get("hash")

	if username == "" || providedHash == "" {
		log.Printf("   ❌ Missing username or hash")
		s.renderTemplate(w, "login.html", loginPageData{Error: "Invalid login link. Please use the link from Telegram bot."})
		return
	}

	// Verify the hash
	expectedHash := s.generateLoginHash(username)
	if !hmac.Equal([]byte(providedHash), []byte(expectedHash)) {
		log.Printf("   ❌ Invalid hash for user: %s", username)
		s.renderTemplate(w, "login.html", loginPageData{Error: "Invalid or expired login link. Please request a new one from the Telegram bot."})
		return
	}

	// Find user by username
	user, err := s.service.GetUserByUsername(username)
	if err != nil {
		log.Printf("   ❌ User not found: %s, error: %v", username, err)
		s.renderTemplate(w, "login.html", loginPageData{Error: "User not found. Please start the Telegram bot first with /start"})
		return
	}

	// Set session
	if err := s.setUserID(w, r, user.ID); err != nil {
		log.Printf("   ❌ Failed to create session: %v", err)
		s.renderTemplate(w, "login.html", loginPageData{Error: "Failed to create session"})
		return
	}

	log.Printf("   ✅ User %s logged in successfully via hash", username)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// generateLoginHash generates an HMAC-SHA256 hash for username
func (s *Server) generateLoginHash(username string) string {
	h := hmac.New(sha256.New, []byte(s.sessionSecret))
	h.Write([]byte(username))
	return hex.EncodeToString(h.Sum(nil))
}

// handleLogout logs out the user
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	log.Printf("🚪 handleLogout called: %s %s", r.Method, r.URL.Path)
	s.clearSession(w, r)
	log.Printf("✅ Session cleared, redirecting to /login")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
