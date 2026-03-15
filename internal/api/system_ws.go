package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

type wsSnapshot struct {
	Type      string      `json:"type"`
	At        time.Time   `json:"at"`
	Status    interface{} `json:"status"`
	Traffic   interface{} `json:"traffic,omitempty"`
	MihomoErr string      `json:"mihomo_error,omitempty"`
}

// SystemWS pushes basic status snapshots over websocket.
func (h *Handler) SystemWS(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if token == "" {
		h.respondError(c, http.StatusUnauthorized, "token required")
		return
	}
	if _, err := h.authManager.ValidateToken(token); err != nil {
		h.respondError(c, http.StatusUnauthorized, "invalid token")
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, readErr := conn.ReadMessage(); readErr != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	pingTicker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer pingTicker.Stop()

	h.writeSnapshot(conn)

	for {
		select {
		case <-readDone:
			return
		case <-ticker.C:
			h.writeSnapshot(conn)
		case <-pingTicker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(3*time.Second)); err != nil {
				return
			}
		}
	}
}

func (h *Handler) writeSnapshot(conn *websocket.Conn) {
	snapshot := h.buildWSSnapshot()
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_ = conn.WriteMessage(websocket.TextMessage, payload)
}

func (h *Handler) buildWSSnapshot() wsSnapshot {
	status := map[string]interface{}{"mihomo_status": "unknown"}
	if h.mihomoManager.IsRunning() {
		status["mihomo_status"] = "running"
		status["mihomo_pid"] = h.mihomoManager.GetPID()
	} else {
		status["mihomo_status"] = "stopped"
		status["mihomo_pid"] = 0
	}

	if action, details, ts, ok := lastAutoUpdateSummary(h.auditStore); ok {
		status["last_auto_update"] = map[string]interface{}{
			"action":  action,
			"details": details,
			"at":      ts,
		}
	}

	if action, details, ts, ok := lastAlertSummary(h.auditStore); ok {
		status["last_alert"] = map[string]interface{}{
			"action":  action,
			"details": details,
			"at":      ts,
		}
	}

	snapshot := wsSnapshot{
		Type:   "snapshot",
		At:     time.Now(),
		Status: status,
	}

	if traffic, err := h.mihomoManager.GetTraffic(); err == nil {
		snapshot.Traffic = traffic
	} else {
		snapshot.MihomoErr = err.Error()
	}

	return snapshot
}

// MihomoLogStream proxies the mihomo /logs WebSocket to the browser client.
// Query param: level=debug|info|warning|error (default: info)
func (h *Handler) MihomoLogStream(c *gin.Context) {
	level := c.DefaultQuery("level", "info")

	// Upgrade the browser connection.
	clientConn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()

	if !h.mihomoManager.IsRunning() {
		msg, _ := json.Marshal(map[string]string{"type": "error", "payload": "mihomo is not running"})
		_ = clientConn.WriteMessage(websocket.TextMessage, msg)
		return
	}

	// Connect to mihomo log WebSocket.
	mihomoURL := fmt.Sprintf("ws://127.0.0.1:%d/logs?level=%s", h.mihomoAPIPort, level)
	mihomoConn, _, err := websocket.DefaultDialer.Dial(mihomoURL, nil)
	if err != nil {
		msg, _ := json.Marshal(map[string]string{"type": "error", "payload": "failed to connect to mihomo: " + err.Error()})
		_ = clientConn.WriteMessage(websocket.TextMessage, msg)
		return
	}
	defer mihomoConn.Close()

	// Forward messages from mihomo → browser.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			mt, msg, err := mihomoConn.ReadMessage()
			if err != nil {
				return
			}
			if writeErr := clientConn.WriteMessage(mt, msg); writeErr != nil {
				return
			}
		}
	}()

	// Keep alive until browser disconnects.
	for {
		select {
		case <-done:
			return
		default:
			if _, _, err := clientConn.ReadMessage(); err != nil {
				return
			}
		}
	}
}
