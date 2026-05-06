package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"

	"tomatogether/backend/internal/api"
	"tomatogether/backend/internal/middleware"
	"tomatogether/backend/internal/repository"
	"tomatogether/backend/internal/service"
	"tomatogether/backend/internal/sse"
)

func main() {
	godotenv.Load()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/tomatogether.db"
	}

	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Enable WAL mode for better concurrent read/write performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("Warning: Failed to enable WAL mode: %v", err)
	}

	// Set busy timeout to reduce "database is locked" errors
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		log.Printf("Warning: Failed to set busy timeout: %v", err)
	}

	// Run database migrations
	if err := runMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	repo := repository.New(db)
	svc := service.New(repo)

	// Log JWT status
	if svc.JWTEnabled() {
		log.Println("JWT authentication enabled (JWT_SECRET configured)")
	} else {
		log.Println("Warning: JWT_SECRET not set. Persistent user JWT tokens will not be generated.")
	}
	handler := api.New(svc)

	hub := sse.NewHub(repo)
	go hub.Run()
	defer hub.Stop()

	r := mux.NewRouter()

	apiRouter := r.PathPrefix("/api").Subrouter()
	handler.RegisterRoutes(apiRouter)

	// Serve static frontend files (with SPA fallback)
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}
	r.PathPrefix("/").Handler(spaFileServer(staticDir))

	// Global middleware stack (applied in order, outermost first)
	corsOrigin := strings.TrimSpace(os.Getenv("CORS_ORIGIN"))
	if corsOrigin == "" {
		corsOrigin = "*"
	}
	r.Use(corsMiddleware(corsOrigin))
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.MaxBytesReader)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s (CORS origin: %s)", port, corsOrigin)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server error:", err)
	}
}

// corsMiddleware returns a CORS middleware with configurable origin
func corsMiddleware(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// spaFileServer returns an http.Handler that serves static files with SPA fallback.
// If the requested file doesn't exist, it serves index.html instead.
func spaFileServer(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't intercept API calls
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the requested file
		path := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// File not found, serve index.html for SPA routing
			r.URL.Path = "/"
		}

		fs.ServeHTTP(w, r)
	})
}

func runMigrations(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"sqlite3",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	log.Println("Database migrations applied successfully")
	return nil
}
