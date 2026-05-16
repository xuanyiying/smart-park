package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %s, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Database.Driver != "postgres" {
		t.Errorf("Database.Driver = %s, want postgres", cfg.Database.Driver)
	}
	if cfg.Database.Source == "" {
		t.Error("Database.Source should not be empty")
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("Redis.Addr = %s, want localhost:6379", cfg.Redis.Addr)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %s, want info", cfg.Log.Level)
	}
	if cfg.Telemetry.Enabled != false {
		t.Error("Telemetry.Enabled should be false by default")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid port zero",
			config: &Config{
				Server:   ServerConfig{Port: 0},
				Database: DatabaseConfig{Source: "source"},
				Redis:    RedisConfig{Addr: "localhost:6379"},
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "invalid port negative",
			config: &Config{
				Server:   ServerConfig{Port: -1},
				Database: DatabaseConfig{Source: "source"},
				Redis:    RedisConfig{Addr: "localhost:6379"},
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "invalid port too large",
			config: &Config{
				Server:   ServerConfig{Port: 70000},
				Database: DatabaseConfig{Source: "source"},
				Redis:    RedisConfig{Addr: "localhost:6379"},
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "empty database source",
			config: &Config{
				Server:   ServerConfig{Port: 8080},
				Database: DatabaseConfig{Source: ""},
				Redis:    RedisConfig{Addr: "localhost:6379"},
			},
			wantErr: true,
			errMsg:  "database source is required",
		},
		{
			name: "empty redis addr",
			config: &Config{
				Server:   ServerConfig{Port: 8080},
				Database: DatabaseConfig{Source: "source"},
				Redis:    RedisConfig{Addr: ""},
			},
			wantErr: true,
			errMsg:  "redis addr is required",
		},
		{
			name: "valid boundary port 1",
			config: &Config{
				Server:   ServerConfig{Port: 1},
				Database: DatabaseConfig{Source: "source"},
				Redis:    RedisConfig{Addr: "localhost:6379"},
			},
			wantErr: false,
		},
		{
			name: "valid boundary port 65535",
			config: &Config{
				Server:   ServerConfig{Port: 65535},
				Database: DatabaseConfig{Source: "source"},
				Redis:    RedisConfig{Addr: "localhost:6379"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadFromFile() expected error for non-existent file")
	}
}

func TestLoadFromFile_ValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`
server:
  port: 9000
  host: "127.0.0.1"
  timeout: 30
database:
  driver: postgres
  source: "host=localhost user=test password=test dbname=test port=5432 sslmode=disable"
  maxOpenConns: 50
  maxIdleConns: 5
redis:
  addr: "redis:6379"
  password: ""
  db: 0
log:
  level: debug
  format: json
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("Server.Port = %d, want 9000", cfg.Server.Port)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %s, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Redis.Addr != "redis:6379" {
		t.Errorf("Redis.Addr = %s, want redis:6379", cfg.Redis.Addr)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %s, want debug", cfg.Log.Level)
	}
}

func TestNewLoader_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`
server:
  port: not_a_number
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := NewLoader(cfgPath)
	if err == nil {
		t.Error("NewLoader() expected error for invalid YAML")
	}
}

func TestConfigLoader_Get(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`
server:
  port: 8080
database:
  driver: postgres
  source: "host=localhost"
redis:
  addr: "localhost:6379"
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loader, err := NewLoader(cfgPath)
	if err != nil {
		t.Fatalf("NewLoader() error: %v", err)
	}

	cfg := loader.Get()
	if cfg == nil {
		t.Fatal("Get() returned nil")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
