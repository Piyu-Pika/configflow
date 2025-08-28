package configflow

import (
	"os"
	"testing"
)

func TestBasicLoading(t *testing.T) {
	type Config struct {
		Port    int    `cfg:"port" default:"8080"`
		AppName string `cfg:"app.name" default:"TestApp"`
		Debug   bool   `cfg:"debug" default:"false"`
	}

	config := &Config{}
	loader := New().AddMap(map[string]interface{}{
		"port":     3000,
		"app.name": "MyApp",
		"debug":    true,
	})

	err := loader.Load(config)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", config.Port)
	}
	if config.AppName != "MyApp" {
		t.Errorf("Expected app name 'MyApp', got %s", config.AppName)
	}
	if !config.Debug {
		t.Errorf("Expected debug true, got %t", config.Debug)
	}
}

func TestEnvironmentOverride(t *testing.T) {
	type Config struct {
		Port int `cfg:"port" env:"TEST_PORT" default:"8080"`
	}

	// Set environment variable
	os.Setenv("TEST_PORT", "9090")
	defer os.Unsetenv("TEST_PORT")

	config := &Config{}
	loader := New().
		AddMap(map[string]interface{}{"port": 3000}).
		AddEnv()

	err := loader.Load(config)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Port != 9090 {
		t.Errorf("Expected port 9090 from env, got %d", config.Port)
	}
}

func TestValidation(t *testing.T) {
	type Config struct {
		Port  int    `cfg:"port" validate:"range:1000,9999"`
		Email string `cfg:"email" validate:"required,email"`
	}

	tests := []struct {
		name      string
		data      map[string]interface{}
		expectErr bool
	}{
		{
			name: "valid config",
			data: map[string]interface{}{
				"port":  8080,
				"email": "test@example.com",
			},
			expectErr: false,
		},
		{
			name: "invalid port range",
			data: map[string]interface{}{
				"port":  500,
				"email": "test@example.com",
			},
			expectErr: true,
		},
		{
			name: "invalid email",
			data: map[string]interface{}{
				"port":  8080,
				"email": "invalid-email",
			},
			expectErr: true,
		},
		{
			name: "missing required field",
			data: map[string]interface{}{
				"port": 8080,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			loader := New().AddMap(tt.data).EnableValidation()
			err := loader.Load(config)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestCustomValidator(t *testing.T) {
	type Config struct {
		Status string `cfg:"status" validate:"custom_status"`
	}

	loader := New().AddValidator("custom_status", func(value interface{}, param string) error {
		status := value.(string)
		if status != "active" && status != "inactive" {
			return ValidationError{
				Field:   "status",
				Value:   value,
				Rule:    "custom_status",
				Message: "status must be 'active' or 'inactive'",
			}
		}
		return nil
	})

	// Test valid status
	config := &Config{}
	err := loader.AddMap(map[string]interface{}{"status": "active"}).Load(config)
	if err != nil {
		t.Errorf("Expected no error for valid status, got: %v", err)
	}

	// Test invalid status
	config = &Config{}
	err = loader.AddMap(map[string]interface{}{"status": "unknown"}).Load(config)
	if err == nil {
		t.Error("Expected error for invalid status")
	}
}

func TestDefaultValues(t *testing.T) {
	type Config struct {
		Port    int    `cfg:"port" default:"8080"`
		AppName string `cfg:"app.name" default:"DefaultApp"`
		Debug   bool   `cfg:"debug" default:"true"`
	}

	config := &Config{}
	loader := New() // No sources, should use defaults

	err := loader.Load(config)
	if err != nil {
		t.Fatalf("Failed to load config with defaults: %v", err)
	}

	if config.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.Port)
	}
	if config.AppName != "DefaultApp" {
		t.Errorf("Expected default app name 'DefaultApp', got %s", config.AppName)
	}
	if !config.Debug {
		t.Errorf("Expected default debug true, got %t", config.Debug)
	}
}

func TestNestedConfig(t *testing.T) {
	type DatabaseConfig struct {
		URL      string `cfg:"database.url"`
		MaxConns int    `cfg:"database.max_connections" default:"10"`
	}

	config := &DatabaseConfig{}
	loader := New().AddMap(map[string]interface{}{
		"database.url":             "postgres://localhost/test",
		"database.max_connections": 20,
	})

	err := loader.Load(config)
	if err != nil {
		t.Fatalf("Failed to load nested config: %v", err)
	}

	if config.URL != "postgres://localhost/test" {
		t.Errorf("Expected database URL, got %s", config.URL)
	}
	if config.MaxConns != 20 {
		t.Errorf("Expected max connections 20, got %d", config.MaxConns)
	}
}

func TestTypeConversion(t *testing.T) {
	type Config struct {
		Port    int     `cfg:"port"`
		Rate    float64 `cfg:"rate"`
		Enabled bool    `cfg:"enabled"`
		Name    string  `cfg:"name"`
	}

	config := &Config{}
	loader := New().AddMap(map[string]interface{}{
		"port":    "8080",    // string to int
		"rate":    "3.14",    // string to float
		"enabled": "true",    // string to bool
		"name":    123,       // int to string
	})

	err := loader.Load(config)
	if err != nil {
		t.Fatalf("Failed to load config with type conversion: %v", err)
	}

	if config.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Port)
	}
	if config.Rate != 3.14 {
		t.Errorf("Expected rate 3.14, got %f", config.Rate)
	}
	if !config.Enabled {
		t.Errorf("Expected enabled true, got %t", config.Enabled)
	}
	if config.Name != "123" {
		t.Errorf("Expected name '123', got %s", config.Name)
	}
}