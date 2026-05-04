package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"

	"tomatogether/backend/internal/api"
	"tomatogether/backend/internal/repository"
	"tomatogether/backend/internal/service"
)

func main() {
	// 加载环境变量
	godotenv.Load()

	// 初始化数据库
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/tomatogether.db"
	}

	// 确保数据目录存在
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// 初始化数据库表
	if err := initDB(db); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 初始化服务
	repo := repository.New(db)
	svc := service.New(repo)
	handler := api.New(svc)

	// 创建路由器
	r := mux.NewRouter()

	// 注册 API 路由
	apiRouter := r.PathPrefix("/api").Subrouter()
	handler.RegisterRoutes(apiRouter)

	// CORS 中间件
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server error:", err)
	}
}

func initDB(db *sql.DB) error {
	// 创建房间表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS rooms (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL DEFAULT '',
			is_readonly INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// 创建房间成员表（核心用户表）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS room_members (
			id TEXT PRIMARY KEY,
			room_id TEXT NOT NULL,
			username TEXT NOT NULL,
			password_hash TEXT NOT NULL DEFAULT '',
			is_owner INTEGER NOT NULL DEFAULT 0,
			joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
			UNIQUE (room_id, username)
		)
	`)
	if err != nil {
		return err
	}

	// 创建 RoomToken 表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS room_tokens (
			id TEXT PRIMARY KEY,
			member_id TEXT NOT NULL,
			room_id TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			last_heartbeat DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
			UNIQUE (member_id, room_id)
		)
	`)
	if err != nil {
		return err
	}

	// 创建项目表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			member_id TEXT NOT NULL,
			room_id TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
			UNIQUE (member_id, room_id, name)
		)
	`)
	if err != nil {
		return err
	}

	// 创建 WIP 待办表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			client_id TEXT,
			member_id TEXT NOT NULL,
			room_id TEXT NOT NULL,
			project_id TEXT,
			title TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'TODO' CHECK(status IN ('TODO', 'WIP', 'DONE')),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
		)
	`)
	if err != nil {
		return err
	}

	// 创建番茄记录表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pomodoro_sessions (
			id TEXT PRIMARY KEY,
			member_id TEXT NOT NULL,
			room_id TEXT NOT NULL,
			project_id TEXT,
			task_id TEXT,
			duration INTEGER NOT NULL DEFAULT 0,
			planned_duration INTEGER NOT NULL DEFAULT 1500,
			is_followed INTEGER NOT NULL DEFAULT 0,
			leader_id TEXT,
			started_at DATETIME NOT NULL,
			ended_at DATETIME,
			FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET NULL,
			FOREIGN KEY (leader_id) REFERENCES room_members(id) ON DELETE SET NULL
		)
	`)
	if err != nil {
		return err
	}

	// 创建用户状态表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_statuses (
			id TEXT PRIMARY KEY,
			member_id TEXT NOT NULL,
			room_id TEXT NOT NULL,
			emoji TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
			UNIQUE (member_id, room_id)
		)
	`)
	if err != nil {
		return err
	}

	// 创建公告表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS announcements (
			id TEXT PRIMARY KEY,
			room_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
			FOREIGN KEY (sender_id) REFERENCES room_members(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	return nil
}
