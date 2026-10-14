package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// Guard against CVE-2026-55677, including the bypasses left open by v4.15.3.
func TestEchoStaticRouteProtection(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "admin"), 0755); err != nil {
		t.Fatal(err)
	}

	const publicContent = "public test file"
	const privateContent = "protected test file"
	for _, file := range []struct {
		name    string
		content string
	}{
		{"index.txt", publicContent},
		{filepath.Join("admin", "secret.txt"), privateContent},
	} {
		if err := os.WriteFile(filepath.Join(root, file.name), []byte(file.content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"public file", "/index.txt", http.StatusOK, publicContent},
		{"guarded file", "/admin/secret.txt", http.StatusForbidden, ""},
		{"encoded slash lowercase", "/admin%2fsecret.txt", http.StatusNotFound, ""},
		{"encoded slash uppercase", "/admin%2Fsecret.txt", http.StatusNotFound, ""},
		{"encoded backslash", "/admin%5csecret.txt", http.StatusNotFound, ""},
		{"encoded guarded prefix", "/%61dmin/secret.txt", http.StatusNotFound, ""},
		{"encoded traversal uppercase", "/public/%2E%2E/admin/secret.txt", http.StatusNotFound, ""},
		{"encoded traversal lowercase", "/public/%2e%2e/admin/secret.txt", http.StatusNotFound, ""},
	}

	for _, method := range []string{"Static", "StaticFS"} {
		t.Run(method, func(t *testing.T) {
			e := echo.New()
			if method == "Static" {
				e.Static("/", root)
			} else {
				e.StaticFS("/", os.DirFS(root))
			}
			e.GET("/admin/*", func(c echo.Context) error {
				return echo.ErrForbidden
			})

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					request := httptest.NewRequest(http.MethodGet, tt.path, nil)
					response := httptest.NewRecorder()
					e.ServeHTTP(response, request)

					if response.Code != tt.wantStatus {
						t.Fatalf("GET %q: status %d, want %d", tt.path, response.Code, tt.wantStatus)
					}
					body := response.Body.String()
					if tt.wantBody != "" && body != tt.wantBody {
						t.Errorf("GET %q: body %q, want %q", tt.path, body, tt.wantBody)
					}
					if strings.Contains(body, privateContent) {
						t.Errorf("GET %q disclosed the protected file", tt.path)
					}
				})
			}
		})
	}
}
