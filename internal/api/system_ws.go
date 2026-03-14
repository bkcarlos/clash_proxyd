package api

import (
	"encoding/json"
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

	ticker := time.NewTicker(3 * time.Second)
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
	if runtimeState, err := h.runtimeStore.Get(); err == nil {
		status["mihomo_status"] = runtimeState.Status
		status["mihomo_pid"] = runtimeState.PID
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
