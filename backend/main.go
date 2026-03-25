package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"

	httpserver "redblue-server/internal/http"
	"redblue-server/internal/db"
	"redblue-server/internal/match"
	"redblue-server/internal/ws"
)

// === 1. 数据结构定义 ===

type AttackEvent struct {
	Event string    `json:"event"`
	Data  EventData `json:"data"`
}

type EventData struct {
	SourceIP    string `json:"source_ip,omitempty"`
	SourceCity  string `json:"source_city,omitempty"` // 新增：指定攻击源城市
	TargetCity  string `json:"target_city,omitempty"`
	TeamID      int    `json:"team_id,omitempty"`
	AttackType  string `json:"attack_type,omitempty"`
	ScoreChange int    `json:"score_change,omitempty"`
	Message     string `json:"message,omitempty"`
	Status      string `json:"status,omitempty"`   // 新增：attempt(尝试), lateral(横向), success(成功)
	MapType     string `json:"map_type,omitempty"` // 新增：china 或 taizhou
	PanelID     string `json:"panel_id,omitempty"` // 新增：用于控制大屏面板隐藏
	Visible     bool   `json:"visible,omitempty"`  // 新增：面板是否可见
	Teams       any    `json:"teams,omitempty"`    // 新增：用于同步队伍数据
}

var (
	upgrader     = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	clients      = make(map[*websocket.Conn]bool)
	clientsMutex sync.Mutex
	broadcast    = make(chan AttackEvent)
)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 升级失败: %v", err)
		return
	}
	defer ws.Close()

	clientsMutex.Lock()
	clients[ws] = true
	clientsMutex.Unlock()
	log.Printf("新大屏节点已连接，当前连接数: %d", len(clients))

	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			clientsMutex.Lock()
			delete(clients, ws)
			clientsMutex.Unlock()
			log.Printf("大屏节点断开连接，当前连接数: %d", len(clients))
			break
		}
	}
}

func handleMessages() {
	for {
		event := <-broadcast
		clientsMutex.Lock()
		for client := range clients {
			if err := client.WriteJSON(event); err != nil {
				log.Printf("推送数据失败: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
		clientsMutex.Unlock()
	}
}

func triggerAttack(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理预检请求 (OPTIONS)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "仅支持 POST", http.StatusMethodNotAllowed)
		return
	}

	var event AttackEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "JSON 解析失败", http.StatusBadRequest)
		return
	}

	broadcast <- event
	log.Printf("分发指令: [%s]", event.Event)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "msg": "指令已推送到大屏"})
}

func main() {
	// 自动读取环境变量（优先 backend/.env，其次项目根 .env）
	// 用于免去手动 export JWT_SECRET/ADMIN_USERNAME/ADMIN_PASSWORD。
	_, thisFile, _, ok := runtime.Caller(0)
	if ok {
		backendEnv := filepath.Join(filepath.Dir(thisFile), ".env")
		rootEnv := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), ".env")
		frontendEnv := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), "frontend", ".env")

		// 只要存在就加载；失败不阻断启动。
		if _, err := os.Stat(backendEnv); err == nil {
			_ = godotenv.Load(backendEnv)
		} else if _, err := os.Stat(rootEnv); err == nil {
			_ = godotenv.Load(rootEnv)
		} else if _, err := os.Stat(frontendEnv); err == nil {
			_ = godotenv.Load(frontendEnv)
		}
	} else {
		// fallback
		_ = godotenv.Load()
	}

	// 数据库文件放在后端目录，便于本机部署与排查。
	dbPath := filepath.Join(".", "redblue.db")
	if env := os.Getenv("RED_BLUE_DB_PATH"); env != "" {
		dbPath = env
	}

	store, err := db.NewStore(dbPath)
	if err != nil {
		log.Fatalf("init sqlite failed: %v", err)
	}

	hub := ws.NewHub()
	matcher := match.NewService(store)

	uploadDir := filepath.Join(filepath.Dir(dbPath), "uploads")
	if v := os.Getenv("RED_BLUE_UPLOAD_DIR"); strings.TrimSpace(v) != "" {
		uploadDir = strings.TrimSpace(v)
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		log.Fatalf("create uploads dir: %v", err)
	}
	log.Printf("📁 得分总榜等上传目录: %s", uploadDir)

	srv := httpserver.NewServer(store, matcher, hub, uploadDir)

	port := os.Getenv("RED_BLUE_PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("🚀 红蓝对抗态势感知后端已启动: http://0.0.0.0%s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
