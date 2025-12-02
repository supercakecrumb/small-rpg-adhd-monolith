package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
)

type loginPageData struct {
	basePageData
	Error string
}

// handleLoginPage displays the login page
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔑 handleLoginPage called: %s %s", r.Method, r.URL.Path)
	log.Printf("   Remote Address: %s", r.RemoteAddr)
	log.Printf("   User-Agent: %s", r.UserAgent())
	locale := s.detectLocale(r)

	// Check if user is already logged in
	if userID, ok := s.getUserID(r); ok {
		log.Printf("   ✅ User already authenticated (ID: %d), redirecting to /dashboard", userID)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	log.Printf("   ❌ User not authenticated, rendering login page")
	s.renderTemplate(w, "login.html", loginPageData{basePageData: basePageData{Locale: locale}})
}

// handleHashLogin processes hash-based login from Telegram bot
// URL format: /login?user=<username>&hash=<hmac>
func (s *Server) handleHashLogin(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔐 handleHashLogin called: %s %s", r.Method, r.URL.Path)
	locale := s.detectLocale(r)

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
		s.renderTemplate(w, "login.html", loginPageData{basePageData: basePageData{Locale: locale}, Error: "Invalid login link. Please use the link from Telegram bot."})
		return
	}

	// Verify the hash
	expectedHash := s.generateLoginHash(username)
	if !hmac.Equal([]byte(providedHash), []byte(expectedHash)) {
		log.Printf("   ❌ Invalid hash for user: %s", username)
		s.renderTemplate(w, "login.html", loginPageData{basePageData: basePageData{Locale: locale}, Error: "Invalid or expired login link. Please request a new one from the Telegram bot."})
		return
	}

	// Find user by username
	user, err := s.service.GetUserByUsername(username)
	if err != nil {
		log.Printf("   ❌ User not found: %s, error: %v", username, err)
		s.renderTemplate(w, "login.html", loginPageData{basePageData: basePageData{Locale: locale}, Error: "User not found. Please start the Telegram bot first with /start"})
		return
	}

	// Set session
	if err := s.setUserID(w, r, user.ID); err != nil {
		log.Printf("   ❌ Failed to create session: %v", err)
		s.renderTemplate(w, "login.html", loginPageData{basePageData: basePageData{Locale: locale}, Error: "Failed to create session"})
		return
	}

	log.Printf("   ✅ User %s logged in successfully via hash", username)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Login Successful</title>
<style>
body { background:#05070f; color:#eaf1ff; font-family: 'Space Grotesk', 'Segoe UI', sans-serif; display:flex; align-items:center; justify-content:center; min-height:100vh; }
.card { background: linear-gradient(145deg, rgba(255,255,255,0.04), rgba(94,232,233,0.08)); padding:24px 28px; border-radius:14px; border:1px solid rgba(123,243,242,0.25); box-shadow:0 14px 36px rgba(0,0,0,0.45); text-align:center; max-width:360px; }
.card h1 { margin:0 0 10px 0; font-size:22px; }
.card p { margin:0 0 16px 0; color:#a5b4d4; }
.btn { display:inline-block; padding:10px 16px; border-radius:10px; border:1px solid rgba(123,243,242,0.35); color:#021014; background:linear-gradient(135deg, #5ee8e9, #36d3d4); text-decoration:none; font-weight:700; }
.hint { font-size:13px; color:#7f8bad; margin-top:8px; }
</style>
</head>
<body>
  <div class="card">
    <h1>Login complete</h1>
    <p>You can continue to the Burrow.</p>
    <a class="btn" href="/dashboard">Go to dashboard</a>
    <div class="hint">If the button doesn't work, copy the link into your browser.</div>
  </div>
</body>
</html>
`))
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
