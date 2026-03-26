package server

import (
	"embed"
	"io/fs"
	"net/http"

	"git-guider/internal/training"
)

//go:embed static
var staticFS embed.FS

func NewRouter(svc *training.Service) http.Handler {
	mux := http.NewServeMux()

	api := NewAPI(svc)

	mux.HandleFunc("GET /api/session", api.HandleSession)
	mux.HandleFunc("POST /api/task/next", api.HandleTaskNext)
	mux.HandleFunc("POST /api/task/verify", api.HandleTaskVerify)
	mux.HandleFunc("GET /api/progress", api.HandleProgress)
	mux.HandleFunc("GET /api/levels", api.HandleLevels)

	mux.Handle("/ws", HandleWS(svc))

	// Serve embedded static files (Svelte build output)
	sub, err := fs.Sub(staticFS, "static")
	if err == nil {
		fileServer := http.FileServer(http.FS(sub))
		mux.Handle("/", fileServer)
	}

	return mux
}

// NewDevRouter creates a router for development (no embedded static files).
func NewDevRouter(svc *training.Service) http.Handler {
	mux := http.NewServeMux()

	api := NewAPI(svc)

	mux.HandleFunc("GET /api/session", api.HandleSession)
	mux.HandleFunc("POST /api/task/next", api.HandleTaskNext)
	mux.HandleFunc("POST /api/task/verify", api.HandleTaskVerify)
	mux.HandleFunc("GET /api/progress", api.HandleProgress)
	mux.HandleFunc("GET /api/levels", api.HandleLevels)

	mux.Handle("/ws", HandleWS(svc))

	return mux
}
