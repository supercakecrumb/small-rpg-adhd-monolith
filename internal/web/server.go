package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"small-rpg-adhd-monolith/internal/core"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
)

const sessionName = "small-rpg-session"
const sessionUserIDKey = "user_id"

// Server represents the HTTP server
type Server struct {
	service      *core.Service
	sessionStore *sessions.CookieStore
	templates    *template.Template
}

// NewServer creates a new Server instance
func NewServer(service *core.Service, sessionSecret string) (*Server, error) {
	// Create session store
	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	// Note: We don't pre-parse all templates here anymore.
	// Instead, we'll parse them on-demand in renderTemplate to avoid
	// conflicts with multiple "content" blocks.

	log.Printf("Template parsing will happen on-demand for each page")

	return &Server{
		service:      service,
		sessionStore: store,
		templates:    nil, // Will be nil, parse on-demand instead
	}, nil
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
	r.Post("/login", s.handleLogin)
	r.Get("/register", s.handleRegisterPage)
	r.Post("/register", s.handleRegister)

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

		// Shop routes
		r.Post("/groups/{groupID}/shop/create", s.handleCreateShopItem)
		r.Post("/shop/{itemID}/buy", s.handleBuyItem)
	})

	return r
}

// getUserID retrieves the user ID from the session
func (s *Server) getUserID(r *http.Request) (int64, bool) {
	session, err := s.sessionStore.Get(r, sessionName)
	if err != nil {
		return 0, false
	}

	userID, ok := session.Values[sessionUserIDKey].(int64)
	if !ok {
		return 0, false
	}

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
		if _, ok := s.getUserID(r); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
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

	tmpl, err := template.ParseFiles(layoutPath, pagePath)
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
	if _, ok := s.getUserID(r); ok {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
