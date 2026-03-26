package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"git-guider/internal/cmdexec"
	"git-guider/internal/training"

	"golang.org/x/net/websocket"
)

type wsMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func HandleWS(svc *training.Service) http.Handler {
	return websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()

		sessionID := ""
		if c, err := conn.Request().Cookie("session_id"); err == nil {
			sessionID = c.Value
		}
		if sessionID == "" {
			sendWS(conn, "error", "no session")
			return
		}

		sess, err := svc.GetSession(sessionID)
		if err != nil {
			sendWS(conn, "error", "session not found")
			return
		}

		exec := cmdexec.NewExecutor(sess.SandboxRoot, sess.CWD)
		var mu sync.Mutex

		// Send initial prompt
		sendWS(conn, "prompt", buildPrompt(exec))

		for {
			var msg wsMessage
			if err := websocket.JSON.Receive(conn, &msg); err != nil {
				return
			}

			if msg.Type != "cmd" || strings.TrimSpace(msg.Data) == "" {
				continue
			}

			mu.Lock()
			result, execErr := exec.Run(msg.Data)
			if result != nil {
				// Update session CWD
				sess.CWD = result.CWD
				exec.CWD = result.CWD
				svc.UpdateSessionCWD(sessionID, result.CWD)
			}
			mu.Unlock()

			if execErr != nil {
				if result != nil && result.Stderr != "" {
					sendWS(conn, "output", result.Stderr)
				} else {
					sendWS(conn, "error", execErr.Error())
				}
			} else if result != nil {
				if result.Stdout != "" {
					sendWS(conn, "output", result.Stdout)
				}
				if result.Stderr != "" {
					sendWS(conn, "stderr", result.Stderr)
				}
			}

			sendWS(conn, "prompt", buildPrompt(exec))
		}
	})
}

func buildPrompt(exec *cmdexec.Executor) string {
	branch := currentBranch(exec)
	rel := relativeDir(exec)

	var parts []string
	if branch != "" {
		parts = append(parts, "("+branch+")")
	}
	if rel != "" && rel != "." {
		parts = append(parts, rel)
	} else {
		parts = append(parts, "~")
	}
	parts = append(parts, "$")
	return strings.Join(parts, " ") + " "
}

func currentBranch(exec *cmdexec.Executor) string {
	result, err := exec.RunInternal("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(result)
}

func relativeDir(exec *cmdexec.Executor) string {
	if exec.CWD == exec.SandboxRoot {
		return "~"
	}
	if strings.HasPrefix(exec.CWD, exec.SandboxRoot+"/") {
		return strings.TrimPrefix(exec.CWD, exec.SandboxRoot+"/")
	}
	return exec.CWD
}

func sendWS(conn *websocket.Conn, msgType, data string) {
	msg := wsMessage{Type: msgType, Data: data}
	if err := websocket.JSON.Send(conn, msg); err != nil {
		log.Printf("ws send error: %v", err)
	}
}

// For debug logging
func init() {
	_ = json.Marshal
}
