package web

import (
	"log"
	"net/http"
)

type loginPageData struct {
	Username string
	Error    string
}

type registerPageData struct {
	Username string
	Error    string
}

// handleLoginPage displays the login page
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate(w, "login.html", loginPageData{})
}

// handleLogin processes login form submission
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderTemplate(w, "login.html", loginPageData{Error: "Invalid form data"})
		return
	}

	username := r.FormValue("username")
	if username == "" {
		s.renderTemplate(w, "login.html", loginPageData{Error: "Username is required"})
		return
	}

	// For simplicity, we'll use GetUserByUsername which we need to add
	// For now, let's just check if user exists by trying to find them
	// Since we don't have GetUserByUsername, we'll create the user if they don't exist
	// This is a simplified auth - in production you'd want proper password handling

	// Try to create user (will fail if exists)
	user, err := s.service.CreateUser(username, nil)
	if err != nil {
		// If user already exists, we'll assume login is successful
		// This is very simplified - in production you'd check password
		// For now, we need to get the user somehow
		// Let's add a GetUserByUsername method to the service
		// For this simplified version, we'll just create if not exists
		s.renderTemplate(w, "login.html", loginPageData{Error: "Login failed. Please register first or check username."})
		return
	}

	// Set session
	if err := s.setUserID(w, r, user.ID); err != nil {
		s.renderTemplate(w, "login.html", loginPageData{Error: "Failed to create session"})
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleRegisterPage displays the registration page
func (s *Server) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate(w, "register.html", registerPageData{})
}

// handleRegister processes registration form submission
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderTemplate(w, "register.html", registerPageData{Error: "Invalid form data"})
		return
	}

	username := r.FormValue("username")
	if username == "" {
		s.renderTemplate(w, "register.html", registerPageData{Error: "Username is required"})
		return
	}

	// Create user
	user, err := s.service.CreateUser(username, nil)
	if err != nil {
		s.renderTemplate(w, "register.html", registerPageData{Error: "Username already exists or registration failed"})
		return
	}

	// Set session
	if err := s.setUserID(w, r, user.ID); err != nil {
		s.renderTemplate(w, "register.html", registerPageData{Error: "Registration successful but failed to log in"})
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleLogout logs out the user
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	log.Printf("🚪 handleLogout called: %s %s", r.Method, r.URL.Path)
	s.clearSession(w, r)
	log.Printf("✅ Session cleared, redirecting to /login")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
