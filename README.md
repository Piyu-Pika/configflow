# ConfigFlow

[![Go Reference](https://pkg.go.dev/badge/github.com/Piyu-Pika/configflow.svg)](https://pkg.go.dev/github.com/Piyu-Pika/configflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/Piyu-Pika/configflow)](https://goreportcard.com/report/github.com/Piyu-Pika/configflow)

ConfigFlow is a flexible configuration management library for Go that supports multiple sources, validation, and type conversion. Inspired by popular JavaScript configuration libraries like `convict`, `joi`, and `config`, but designed specifically for Go's type system.

## Features

- 🔄 **Multiple Sources**: Load from JSON/YAML files, environment variables, and maps
- ✅ **Built-in Validation**: Required, URL, email, range, min/max validators
- 🎯 **Custom Validators**: Add your own validation logic
- 🌍 **Environment Override**: Environment variables take precedence
- 📁 **Nested Config**: Support for nested configuration structures
- 🔧 **Default Values**: Fallback to default values when not provided
- 🏷️ **Type Conversion**: Automatic type conversion for strings, ints, bools, floats

## Installation

```bash
go get github.com/Piyu-Pika/configflow
```

## Quick Start

Define your configuration structure:

```go
type Config struct {
    Port     int    `cfg:"port" env:"PORT" validate:"range:1000,9999" default:"8080"`
    Database string `cfg:"database.url" env:"DATABASE_URL" validate:"required,url"`
    Debug    bool   `cfg:"debug" env:"DEBUG" default:"false"`
    Email    string `cfg:"admin.email" env:"ADMIN_EMAIL" validate:"required,email"`
}
```

Load configuration:

```go
config := &Config{}
loader := configflow.New().
    AddFile("config.yaml").     // Load from YAML file
    AddEnv().                   // Override with env vars
    EnableValidation()          // Enable validation

err := loader.Load(config)
if err != nil {
    log.Fatal(err)
}
```

## Configuration Sources

### File Sources

Supports JSON and YAML files:

```yaml
# config.yaml
port: 3000
database:
  url: "postgres://localhost/mydb"
admin:
  email: "admin@example.com"
debug: true
```

```json
{
  "port": 3000,
  "database": {
    "url": "postgres://localhost/mydb"
  },
  "admin": {
    "email": "admin@example.com"
  },
  "debug": true
}
```

### Environment Variables

Environment variables take precedence over file values:

```bash
export PORT=8080
export DATABASE_URL=postgres://prod/mydb
export DEBUG=false
```

### Map Sources (Defaults)

Perfect for setting application defaults:

```go
defaults := map[string]interface{}{
    "port": 8080,
    "debug": false,
    "timeout": 30,
}

loader := configflow.New().
    AddMap(defaults).           // Lowest priority
    AddFile("config.yaml").     // Medium priority
    AddEnv()                    // Highest priority
```

## Validation

### Built-in Validators

- `required` - Field must not be empty
- `url` - Must be a valid URL
- `email` - Must be a valid email address
- `range:min,max` - Integer must be within range
- `min:value` - Integer must be at least value
- `max:value` - Integer must be at most value

### Custom Validators

Add your own validation logic:

```go
loader.AddValidator("positive", func(value interface{}, param string) error {
    if val, err := strconv.Atoi(fmt.Sprintf("%v", value)); err == nil {
        if val <= 0 {
            return fmt.Errorf("value must be positive")
        }
    }
    return nil
})

// Use in struct tags
type Config struct {
    Count int `validate:"positive"`
}
```

## Examples

### Web Server Configuration

```go
type ServerConfig struct {
    Host         string `cfg:"server.host" env:"HOST" default:"localhost"`
    Port         int    `cfg:"server.port" env:"PORT" validate:"range:1000,65535" default:"8080"`
    ReadTimeout  int    `cfg:"server.read_timeout" env:"READ_TIMEOUT" default:"30"`
    WriteTimeout int    `cfg:"server.write_timeout" env:"WRITE_TIMEOUT" default:"30"`
    DatabaseURL  string `cfg:"database.url" env:"DATABASE_URL" validate:"required,url"`
    RedisURL     string `cfg:"redis.url" env:"REDIS_URL" validate:"required,url"`
    LogLevel     string `cfg:"log.level" env:"LOG_LEVEL" default:"info"`
    Debug        bool   `cfg:"debug" env:"DEBUG" default:"false"`
}

func main() {
    config := &ServerConfig{}
    
    loader := configflow.New().
        AddFile("config.yaml").
        AddEnv().
        EnableValidation()
    
    if err := loader.Load(config); err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    fmt.Printf("Server will start on %s:%d\n", config.Host, config.Port)
}
```

### Database Configuration with Custom Validation

```go
type DBConfig struct {
    Driver   string `cfg:"db.driver" env:"DB_DRIVER" validate:"required,db_driver"`
    Host     string `cfg:"db.host" env:"DB_HOST" validate:"required"`
    Port     int    `cfg:"db.port" env:"DB_PORT" validate:"range:1,65535"`
    Database string `cfg:"db.name" env:"DB_NAME" validate:"required"`
    Username string `cfg:"db.user" env:"DB_USER" validate:"required"`
    Password string `cfg:"db.password" env:"DB_PASSWORD" validate:"required"`
    MaxConns int    `cfg:"db.max_connections" env:"DB_MAX_CONNS" validate:"min:1" default:"10"`
}

func main() {
    loader := configflow.New().
        AddValidator("db_driver", func(value interface{}, param string) error {
            driver := fmt.Sprintf("%v", value)
            allowed := []string{"postgres", "mysql", "sqlite"}
            for _, d := range allowed {
                if driver == d {
                    return nil
                }
            }
            return fmt.Errorf("driver must be one of: %v", allowed)
        }).
        AddFile("database.yaml").
        AddEnv()

    config := &DBConfig{}
    if err := loader.Load(config); err != nil {
        log.Fatal(err)
    }
}
```

## Error Handling

ConfigFlow provides detailed error information:

```go
err := loader.Load(config)
if err != nil {
    if validationErr, ok := err.(*configflow.ValidationError); ok {
        fmt.Printf("Validation failed for field '%s': %s\n", 
            validationErr.Field, validationErr.Message)
    } else {
        fmt.Printf("Config error: %v\n", err)
    }
}
```

## Best Practices

1. **Use struct tags** to clearly define field mapping and validation
2. **Set reasonable defaults** for optional configuration
3. **Validate critical fields** like URLs, ports, and required strings
4. **Use environment variables** for deployment-specific overrides
5. **Keep configuration files** in version control (excluding secrets)
6. **Use custom validators** for domain-specific validation logic

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request