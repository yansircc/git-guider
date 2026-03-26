package server

import (
	"encoding/json"
	"net/http"

	"git-guider/internal/training"
)

type API struct {
	svc *training.Service
}

func NewAPI(svc *training.Service) *API {
	return &API{svc: svc}
}

func (a *API) HandleSession(w http.ResponseWriter, r *http.Request) {
	// GET: get or create session (stored in cookie)
	sessionID := ""
	if c, err := r.Cookie("session_id"); err == nil {
		sessionID = c.Value
	}

	var sess interface{}
	var err error

	if sessionID != "" {
		sess, err = a.svc.GetSession(sessionID)
		if err != nil {
			sess = nil
		}
	}

	if sess == nil {
		s, err := a.svc.CreateSession()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:  "session_id",
			Value: s.ID,
			Path:  "/",
		})
		sess = s
	}

	writeJSON(w, sess)
}

func (a *API) HandleTaskNext(w http.ResponseWriter, r *http.Request) {
	sess, err := a.sessionFromCookie(r)
	if err != nil {
		http.Error(w, "no session", 400)
		return
	}

	task, err := a.svc.SelectNextTask(sess.ID)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	if err := a.svc.StartTask(sess, task.ID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Reload session to get updated task info
	sess, _ = a.svc.GetSession(sess.ID)
	writeJSON(w, map[string]any{
		"task":    task,
		"session": sess,
	})
}

func (a *API) HandleTaskVerify(w http.ResponseWriter, r *http.Request) {
	sess, err := a.sessionFromCookie(r)
	if err != nil {
		http.Error(w, "no session", 400)
		return
	}

	result, err := a.svc.VerifyTask(sess)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	writeJSON(w, result)
}

func (a *API) HandleProgress(w http.ResponseWriter, r *http.Request) {
	progress, err := a.svc.GetProgress()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, progress)
}

func (a *API) HandleLevels(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.svc.GetLevels())
}

func (a *API) sessionFromCookie(r *http.Request) (*training.SessionInfo, error) {
	c, err := r.Cookie("session_id")
	if err != nil {
		return nil, err
	}
	sess, err := a.svc.GetSession(c.Value)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
