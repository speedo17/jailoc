package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/seznam/jailoc/internal/workspace"
)

func TestIsDuplicate(t *testing.T) {
	tests := []struct {
		name   string
		paths  []string
		target string
		want   bool
	}{
		{
			name:   "empty slice",
			paths:  []string{},
			target: "/path/to/something",
			want:   false,
		},
		{
			name:   "single element exact match",
			paths:  []string{"/home/user/projects"},
			target: "/home/user/projects",
			want:   true,
		},
		{
			name:   "single element no match",
			paths:  []string{"/home/user/projects"},
			target: "/home/user/other",
			want:   false,
		},
		{
			name:   "multiple elements match in middle",
			paths:  []string{"/path/a", "/path/b", "/path/c"},
			target: "/path/b",
			want:   true,
		},
		{
			name:   "multiple elements no match",
			paths:  []string{"/path/a", "/path/b", "/path/c"},
			target: "/path/d",
			want:   false,
		},
		{
			name:   "case sensitivity different",
			paths:  []string{"Foo"},
			target: "foo",
			want:   false,
		},
		{
			name:   "case sensitivity same",
			paths:  []string{"foo"},
			target: "foo",
			want:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isDuplicate(tc.paths, tc.target)
			if got != tc.want {
				t.Errorf("isDuplicate(%#v, %q) = %v, want %v", tc.paths, tc.target, got, tc.want)
			}
		})
	}
}

func TestAddComposePath(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
	}{
		{
			name:      "default workspace",
			workspace: "default",
		},
		{
			name:      "api workspace",
			workspace: "api",
		},
		{
			name:      "custom workspace name",
			workspace: "my-production-env",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			got := filepath.Join(ComposeCacheDir(tc.workspace), "docker-compose.yml")
			expected := filepath.Join(home, ".cache", "jailoc", tc.workspace, "docker-compose.yml")

			if got != expected {
				t.Errorf("ComposeCacheDir(%q) compose path = %q, want %q", tc.workspace, got, expected)
			}

			if !filepath.IsAbs(got) {
				t.Errorf("expected absolute path, got %q", got)
			}
		})
	}

	t.Run("fallback to TempDir when UserHomeDir fails", func(t *testing.T) {
		t.Setenv("HOME", "")

		got := filepath.Join(ComposeCacheDir("test-workspace"), "docker-compose.yml")

		if !filepath.IsAbs(got) {
			t.Fatalf("expected absolute path, got %q", got)
		}

		expected := filepath.Join(os.TempDir(), "jailoc", "test-workspace", "docker-compose.yml")
		if got != expected {
			t.Errorf("fallback compose path = %q, want %q", got, expected)
		}
	})
}

// TestMaybeRestartWorkspace exercises maybeRestartWorkspace for cases that do
// not require Docker. Uses t.Setenv, so no t.Parallel().
func TestMaybeRestartWorkspace(t *testing.T) {
	t.Run("returns nil without password when workspace is stopped", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("OPENCODE_SERVER_PASSWORD", "")

		// Create the compose file so the function proceeds past the stat check.
		// IsRunning will fail (no Docker), returning nil before the password guard.
		ws := &workspace.Resolved{Name: "test"}
		compDir := ComposeCacheDir(ws.Name)
		if err := os.MkdirAll(compDir, 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(compDir, "docker-compose.yml"), []byte("services: {}"), 0o600); err != nil {
			t.Fatalf("write compose file: %v", err)
		}

		// No Docker available — IsRunning returns an error, function returns nil.
		err := maybeRestartWorkspace(context.Background(), ws)
		if err != nil {
			t.Fatalf("expected nil for stopped workspace, got: %v", err)
		}
	})
}
