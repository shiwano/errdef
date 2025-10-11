package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/shiwano/errdef/examples/http_api/internal/handler"
	"github.com/shiwano/errdef/examples/http_api/internal/middleware"
	"github.com/shiwano/errdef/examples/http_api/internal/repository"
	"github.com/shiwano/errdef/examples/http_api/internal/service"
)

func TestHTTPAPI(t *testing.T) {
	srv := setupTestServer()
	defer srv.Close()

	tests := []struct {
		name         string
		method       string
		path         string
		body         string
		headers      map[string]string
		wantStatus   int
		wantResponse map[string]any
	}{
		{
			name:       "get user success",
			method:     http.MethodGet,
			path:       "/users/1",
			wantStatus: http.StatusOK,
			wantResponse: map[string]any{
				"ID":    "1",
				"Name":  "Alice",
				"Email": "alice@example.com",
			},
		},
		{
			name:       "get non-existent user",
			method:     http.MethodGet,
			path:       "/users/999",
			wantStatus: http.StatusNotFound,
			wantResponse: map[string]any{
				"kind":     "not_found",
				"error":    "user not found",
				"trace_id": nil,
			},
		},
		{
			name:       "create user with invalid email",
			method:     http.MethodPost,
			path:       "/users",
			body:       `{"name":"David","email":"invalid-email"}`,
			wantStatus: http.StatusBadRequest,
			wantResponse: map[string]any{
				"kind":     "validation",
				"error":    "validation failed",
				"trace_id": nil,
				"validation_errors": map[string]any{
					"email": "email is invalid",
				},
			},
		},
		{
			name:       "create user with duplicate email",
			method:     http.MethodPost,
			path:       "/users",
			body:       `{"name":"Alice2","email":"alice@example.com"}`,
			wantStatus: http.StatusConflict,
			wantResponse: map[string]any{
				"kind":     "conflict",
				"error":    "email already exists",
				"trace_id": nil,
			},
		},
		{
			name:       "update another user's data",
			method:     http.MethodPut,
			path:       "/users/1",
			body:       `{"name":"Alice Hacked","email":"hacked@example.com"}`,
			headers:    map[string]string{"X-User-ID": "2"},
			wantStatus: http.StatusForbidden,
			wantResponse: map[string]any{
				"kind":     "forbidden",
				"error":    "an internal error occurred",
				"trace_id": nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody io.Reader
			if tt.body != "" {
				reqBody = bytes.NewBufferString(tt.body)
			}

			req, err := http.NewRequest(tt.method, srv.URL+tt.path, reqBody)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed to send request: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			want := tt.wantResponse
			if _, ok := want["trace_id"]; ok && want["trace_id"] == nil {
				want["trace_id"] = got["trace_id"]
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("response mismatch:\ngot:  %+v\nwant: %+v", got, want)
			}
		})
	}
}

func setupTestServer() *httptest.Server {
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

	handlerWithMiddleware := middleware.Recovery(
		middleware.Logging(
			middleware.Tracing(mux),
		),
	)

	return httptest.NewServer(handlerWithMiddleware)
}
