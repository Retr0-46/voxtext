package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromDotEnv(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDir) })

	for _, name := range []string{"DATABASE_URL", "JWT_SECRET", "HTTP_ADDR"} {
		value, exists := os.LookupEnv(name)
		if exists {
			if err := os.Unsetenv(name); err != nil {
				t.Fatal(err)
			}
		}
		t.Cleanup(func() {
			if exists {
				_ = os.Setenv(name, value)
			} else {
				_ = os.Unsetenv(name)
			}
		})
	}

	contents := []byte("DATABASE_URL=postgres://localhost/voxtext\nJWT_SECRET=test-secret\nHTTP_ADDR=:9090\n")
	if err := os.WriteFile(filepath.Join(".", ".env"), contents, 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseAddress != "postgres://localhost/voxtext" || cfg.JWTSecret != "test-secret" || cfg.ServerAddress != ":9090" {
		t.Fatalf("unexpected configuration: %+v", cfg)
	}
}

func TestLoadWithoutDotEnvRequiresSecrets(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDir) })

	t.Setenv("DATABASE_URL", "postgres://localhost/voxtext")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("HTTP_ADDR", "")
	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("Load() error = %v, want missing JWT_SECRET", err)
	}
}
