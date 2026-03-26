package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"git-guider/internal/server"
	"git-guider/internal/tasks"
	"git-guider/internal/training"
)

func main() {
	port := flag.Int("port", 3000, "server port")
	dev := flag.Bool("dev", false, "development mode (no embedded static, CORS enabled)")
	taskDir := flag.String("tasks", "", "path to tasks directory (overrides embedded)")
	flag.Parse()

	var bank *training.TaskBank
	var err error

	if *taskDir != "" {
		bank, err = training.LoadTaskBank(*taskDir)
	} else {
		bank, err = training.LoadTaskBankFS(tasks.FS)
	}
	if err != nil {
		log.Fatalf("load tasks: %v", err)
	}
	log.Printf("loaded %d topics", len(bank.Topics))

	home, _ := os.UserHomeDir()
	dbDir := filepath.Join(home, ".git-guider")
	os.MkdirAll(dbDir, 0o755)
	dbPath := filepath.Join(dbDir, "progress.db")
	sandboxBase := filepath.Join(dbDir, "sandboxes")
	os.MkdirAll(sandboxBase, 0o755)

	store, err := training.NewStore(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	svc := training.NewService(store, bank, sandboxBase)

	var handler http.Handler
	if *dev {
		handler = server.NewDevRouter(svc)
		handler = corsMiddleware(handler)
		log.Printf("dev mode: API on :%d, serve frontend via `bun run dev` on :5173", *port)
	} else {
		handler = server.NewRouter(svc)
		log.Printf("serving on http://localhost:%d", *port)
	}

	addr := fmt.Sprintf(":%d", *port)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
