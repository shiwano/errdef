package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/shiwano/errdef/examples/http_api/internal/handler"
	"github.com/shiwano/errdef/examples/http_api/internal/middleware"
	"github.com/shiwano/errdef/examples/http_api/internal/repository"
	"github.com/shiwano/errdef/examples/http_api/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	repo := repository.NewInMemory()
	svc := service.New(repo)
	h := handler.New(svc)

	mux := http.NewServeMux()

	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetUser(w, r)
		case http.MethodPut:
			h.UpdateUser(w, r)
		case http.MethodDelete:
			h.DeleteUser(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.CreateUser(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	handler := middleware.Recovery(
		middleware.Logging(
			middleware.Tracing(mux),
		),
	)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Print demo instructions
	printDemoInstructions()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	slog.Info("server stopped")
}

func printDemoInstructions() {
	instructions := `
╔════════════════════════════════════════════════════════════════════════════╗
║                         HTTP API Example Running                          ║
╚════════════════════════════════════════════════════════════════════════════╝

Server is listening on http://localhost:8080

Try these example requests:

1. Get user (success):
   curl http://localhost:8080/users/1

2. Get non-existent user (not found):
   curl http://localhost:8080/users/999

3. Create user (success):
   curl -X POST http://localhost:8080/users \
     -H "Content-Type: application/json" \
     -d '{"name":"Charlie","email":"charlie@example.com"}'

4. Create user with invalid email (validation error):
   curl -X POST http://localhost:8080/users \
     -H "Content-Type: application/json" \
     -d '{"name":"David","email":"invalid-email"}'

5. Create user with duplicate email (conflict):
   curl -X POST http://localhost:8080/users \
     -H "Content-Type: application/json" \
     -d '{"name":"Alice2","email":"alice@example.com"}'

6. Update user (requires authentication):
   curl -X PUT http://localhost:8080/users/1 \
     -H "Content-Type: application/json" \
     -H "X-User-ID: 1" \
     -d '{"name":"Alice Updated","email":"alice.new@example.com"}'

7. Update another user's data (forbidden):
   curl -X PUT http://localhost:8080/users/1 \
     -H "Content-Type: application/json" \
     -H "X-User-ID: 2" \
     -d '{"name":"Alice Hacked","email":"hacked@example.com"}'

8. Delete user (requires authentication):
   curl -X DELETE http://localhost:8080/users/2 \
     -H "X-User-ID: 2"

9. Delete another user's data (forbidden):
   curl -X DELETE http://localhost:8080/users/1 \
     -H "X-User-ID: 2"

Press Ctrl+C to stop the server.
`
	for line := range strings.SplitSeq(strings.TrimSpace(instructions), "\n") {
		slog.Info(line)
	}
}
