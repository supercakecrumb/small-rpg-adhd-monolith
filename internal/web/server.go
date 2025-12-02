package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"small-rpg-adhd-monolith/internal/core"
	"small-rpg-adhd-monolith/internal/i18n"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
)

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

const sessionName = "small-rpg-session"
const sessionUserIDKey = "user_id"
const sessionLocaleKey = "locale"

// Server represents the HTTP server
type Server struct {
	service       *core.Service
	sessionStore  *sessions.CookieStore
	templates     *template.Template
	sessionSecret string
	translator    *i18n.Translator
}

// NewServer creates a new Server instance
func NewServer(service *core.Service, sessionSecret string) (*Server, error) {
	// Create session store
	store := sessions.NewCookieStore([]byte(sessionSecret))

	// Detect if running behind HTTPS by checking PUBLIC_URL environment variable
	publicURL := getEnv("PUBLIC_URL", "http://localhost:8080")
	isHTTPS := len(publicURL) >= 5 && publicURL[:5] == "https"

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   isHTTPS, // Set Secure flag for HTTPS environments
		SameSite: http.SameSiteLaxMode,
	}

	if isHTTPS {
		log.Printf("🔒 Running behind HTTPS - Secure cookie flag enabled")
	} else {
		log.Printf("🔓 Running on HTTP - Secure cookie flag disabled (local dev)")
	}

	// Note: We don't pre-parse all templates here anymore.
	// Instead, we'll parse them on-demand in renderTemplate to avoid
	// conflicts with multiple "content" blocks.

	log.Printf("Template parsing will happen on-demand for each page")

	translator, err := i18n.NewTranslator("locales", "en")
	if err != nil {
		log.Printf("⚠️ Failed to load locales: %v", err)
		translator = i18n.NewFallback("en")
	}

	return &Server{
		service:       service,
		sessionStore:  store,
		templates:     nil, // Will be nil, parse on-demand instead
		sessionSecret: sessionSecret,
		translator:    translator,
	}, nil
}

// Translator exposes the i18n translator (useful for other services like the bot).
func (s *Server) Translator() *i18n.Translator {
	return s.translator
}

// Router creates and configures the HTTP router
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Static files
	fileServer := http.FileServer(http.Dir("static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Public routes
	r.Get("/", s.handleHome)
	r.Get("/login", s.handleLoginPage)
	r.Get("/auth", s.handleHashLogin) // Hash-based login from Telegram
	r.Get("/locale", s.handleSetLocale)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Get("/dashboard", s.handleDashboard)
		r.Get("/logout", s.handleLogout)

		// Group routes
		r.Post("/groups/create", s.handleCreateGroup)
		r.Post("/groups/join", s.handleJoinGroup)
		r.Get("/groups/{groupID}", s.handleGroupView)

		// Task routes
		r.Post("/groups/{groupID}/tasks/create", s.handleCreateTask)
		r.Post("/tasks/{taskID}/complete", s.handleCompleteTask)
		r.Post("/tasks/{taskID}/update", s.handleUpdateTask)
		r.Post("/tasks/{taskID}/delete", s.handleDeleteTask)

		// Shop routes
		r.Post("/groups/{groupID}/shop/create", s.handleCreateShopItem)
		r.Post("/shop/{itemID}/buy", s.handleBuyItem)
		r.Post("/shop/{itemID}/update", s.handleUpdateShopItem)
		r.Post("/shop/{itemID}/delete", s.handleDeleteShopItem)

		// History routes
		r.Get("/groups/{groupID}/tasks/log", s.handleTaskLog)
		r.Get("/groups/{groupID}/purchases/log", s.handlePurchaseLog)
		r.Post("/purchases/{purchaseID}/fulfill", s.handleMarkPurchaseFulfilled)

		// Transaction undo route
		r.Post("/transactions/{transactionID}/undo", s.handleUndoTransaction)
	})

	return r
}

// detectLocale picks locale from session then Accept-Language with fallback to default.
func (s *Server) detectLocale(r *http.Request) string {
	if session, err := s.sessionStore.Get(r, sessionName); err == nil {
		if l, ok := session.Values[sessionLocaleKey].(string); ok && l != "" {
			return l
		}
	}
	al := r.Header.Get("Accept-Language")
	if strings.HasPrefix(strings.ToLower(al), "ru") {
		return "ru"
	}
	return "en"
}

// handleSetLocale stores locale in session and redirects back.
func (s *Server) handleSetLocale(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang != "ru" && lang != "en" {
		lang = "en"
	}
	_ = s.setLocale(w, r, lang)
	ref := r.Header.Get("Referer")
	if ref == "" {
		ref = "/"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

// getUserID retrieves the user ID from the session
func (s *Server) getUserID(r *http.Request) (int64, bool) {
	log.Printf("🔍 getUserID called for %s %s", r.Method, r.URL.Path)
	session, err := s.sessionStore.Get(r, sessionName)
	if err != nil {
		log.Printf("   ❌ Session retrieval error: %v", err)
		return 0, false
	}

	userID, ok := session.Values[sessionUserIDKey].(int64)
	if !ok {
		log.Printf("   ❌ No user_id in session or invalid type")
		return 0, false
	}

	log.Printf("   ✅ User ID found: %d", userID)
	return userID, true
}

// setUserID sets the user ID in the session
func (s *Server) setUserID(w http.ResponseWriter, r *http.Request, userID int64) error {
	session, err := s.sessionStore.Get(r, sessionName)
	if err != nil {
		return err
	}

	session.Values[sessionUserIDKey] = userID
	return session.Save(r, w)
}

// setLocale sets the preferred locale in session.
func (s *Server) setLocale(w http.ResponseWriter, r *http.Request, locale string) error {
	session, err := s.sessionStore.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Values[sessionLocaleKey] = locale
	return session.Save(r, w)
}

// clearSession clears the session
func (s *Server) clearSession(w http.ResponseWriter, r *http.Request) error {
	session, err := s.sessionStore.Get(r, sessionName)
	if err != nil {
		return err
	}

	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// requireAuth is middleware that ensures the user is authenticated
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🔐 requireAuth middleware for: %s %s", r.Method, r.URL.Path)
		if _, ok := s.getUserID(r); !ok {
			log.Printf("   ❌ Not authenticated, rendering login (no redirect to avoid loops)")
			// Render login page directly to avoid redirect loops when cookies are blocked
			s.handleLoginPage(w, r)
			return
		}
		log.Printf("   ✅ Authenticated, proceeding to handler")
		next.ServeHTTP(w, r)
	})
}

// renderTemplate renders a template with the given data
// It parses layout.html together with the specific page template to ensure
// the correct "content" block is used for each page
func (s *Server) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	log.Printf("Attempting to render template: %s", name)

	// Parse layout.html and the specific page template together
	layoutPath := filepath.Join("templates", "layout.html")
	pagePath := filepath.Join("templates", name)

	funcMap := template.FuncMap{
		"t": func(locale, key string) string {
			if s.translator == nil {
				return key
			}
			return s.translator.T(locale, key)
		},
	}

	tmpl, err := template.New(filepath.Base(layoutPath)).Funcs(funcMap).ParseFiles(layoutPath, pagePath)
	if err != nil {
		log.Printf("ERROR parsing templates for %s: %v", name, err)
		http.Error(w, fmt.Sprintf("Template parsing error: %v", err), http.StatusInternalServerError)
		return
	}

	// Execute layout.html which includes the {{template "content" .}} directive
	// The page template defines the "content" block
	err = tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Printf("ERROR rendering template %s: %v", name, err)
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully rendered template: %s using layout.html", name)
}

// handleHome redirects to dashboard if logged in, otherwise to login
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	log.Printf("🏠 handleHome called: %s %s", r.Method, r.URL.Path)
	log.Printf("   Remote Address: %s", r.RemoteAddr)
	log.Printf("   User-Agent: %s", r.UserAgent())

	if userID, ok := s.getUserID(r); ok {
		log.Printf("   ✅ User authenticated (ID: %d), redirecting to /dashboard", userID)
		// Render dashboard directly to avoid redirect loops in some environments
		s.handleDashboard(w, r)
		return
	}
	log.Printf("   ❌ User not authenticated, rendering login")
	s.handleLoginPage(w, r)
}
