package configflow_test

import (
	"fmt"
	"log"
	"os"

	"github.com/Piyu-Pika/configflow"
)

// ExampleBasicUsage demonstrates basic configuration loading
func ExampleBasicUsage() {
	type Config struct {
		Port    int    `cfg:"port" env:"PORT" default:"8080"`
		Debug   bool   `cfg:"debug" env:"DEBUG" default:"false"`
		AppName string `cfg:"app.name" default:"MyApp"`
	}

	config := &Config{}
	loader := configflow.New().
		AddMap(map[string]interface{}{
			"port":     3000,
			"app.name": "TestApp",
		}).
		EnableValidation()

	err := loader.Load(config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Port: %d, Debug: %t, App: %s\n", config.Port, config.Debug, config.AppName)
	// Output: Port: 3000, Debug: false, App: TestApp
}

// ExampleValidation demonstrates configuration validation
func ExampleValidation() {
	type Config struct {
		Port  int    `cfg:"port" validate:"range:1000,9999"`
		Email string `cfg:"email" validate:"required,email"`
		URL   string `cfg:"url" validate:"url"`
	}

	config := &Config{}
	loader := configflow.New().
		AddMap(map[string]interface{}{
			"port":  8080,
			"email": "admin@example.com",
			"url":   "https://example.com",
		}).
		EnableValidation()

	err := loader.Load(config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Valid config loaded: Port=%d\n", config.Port)
	// Output: Valid config loaded: Port=8080
}

// ExampleCustomValidator shows how to add custom validators
func ExampleCustomValidator() {
	type Config struct {
		Environment string `cfg:"env" validate:"environment"`
	}

	loader := configflow.New().
		AddValidator("environment", func(value interface{}, param string) error {
			env := fmt.Sprintf("%v", value)
			allowed := []string{"development", "staging", "production"}
			for _, e := range allowed {
				if env == e {
					return nil
				}
			}
			return fmt.Errorf("environment must be one of: %v", allowed)
		}).
		AddMap(map[string]interface{}{
			"env": "development",
		})

	config := &Config{}
	err := loader.Load(config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Environment: %s\n", config.Environment)
	// Output: Environment: development
}

// ExampleEnvironmentOverride demonstrates environment variable precedence
func ExampleEnvironmentOverride() {
	type Config struct {
		Port int `cfg:"port" env:"APP_PORT" default:"3000"`
	}

	// Set environment variable
	os.Setenv("APP_PORT", "8080")
	defer os.Unsetenv("APP_PORT")

	config := &Config{}
	loader := configflow.New().
		AddMap(map[string]interface{}{"port": 3000}). // Default from map
		AddEnv() // Environment override

	err := loader.Load(config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Port from env: %d\n", config.Port)
	// Output: Port from env: 8080
}