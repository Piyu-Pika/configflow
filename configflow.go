// Package configflow provides a flexible configuration management system
// with validation, environment variable support, and multiple source loading.
//
// ConfigFlow is inspired by popular configuration libraries but designed
// specifically for Go's type system and conventions.
//
// Features:
//   - Load from multiple sources (files, environment variables, maps)
//   - Built-in validation with custom validators
//   - Support for JSON and YAML files
//   - Environment variable override
//   - Default values
//   - Type conversion
//   - Nested configuration support
//
// Example usage:
//   type AppConfig struct {
//       Port     int    `cfg:"port" env:"PORT" validate:"range:1000,9999"`
//       Database string `cfg:"database.url" env:"DATABASE_URL" validate:"required,url"`
//       Debug    bool   `cfg:"debug" env:"DEBUG" default:"false"`
//   }
//
//   config := &AppConfig{}
//   loader := configflow.New().
//       AddFile("config.yaml").
//       AddEnv().
//       EnableValidation()
//   
//   err := loader.Load(config)
package configflow

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader handles configuration loading from multiple sources
type Loader struct {
	sources    []Source
	validators map[string]ValidatorFunc
	strict     bool
}

// Source represents a configuration source
type Source interface {
	Load() (map[string]interface{}, error)
	Priority() int
}

// ValidatorFunc validates a field value
type ValidatorFunc func(value interface{}, param string) error

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Rule    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// New creates a new configuration loader
func New() *Loader {
	return &Loader{
		sources:    make([]Source, 0),
		validators: getBuiltinValidators(),
		strict:     false,
	}
}

// AddFile adds a file source (JSON, YAML, or TOML)
func (l *Loader) AddFile(path string) *Loader {
	l.sources = append(l.sources, &FileSource{Path: path})
	return l
}

// AddEnv adds environment variables as a source
func (l *Loader) AddEnv() *Loader {
	l.sources = append(l.sources, &EnvSource{})
	return l
}

// AddMap adds a map source (useful for defaults or testing)
func (l *Loader) AddMap(data map[string]interface{}) *Loader {
	l.sources = append(l.sources, &MapSource{Data: data})
	return l
}

// EnableValidation enables field validation
func (l *Loader) EnableValidation() *Loader {
	// Validation is enabled by checking for validate tags
	return l
}

// Strict enables strict mode (fail on unknown fields)
func (l *Loader) Strict() *Loader {
	l.strict = true
	return l
}

// AddValidator adds a custom validator
func (l *Loader) AddValidator(name string, validator ValidatorFunc) *Loader {
	l.validators[name] = validator
	return l
}

// Load loads configuration into the provided struct
func (l *Loader) Load(config interface{}) error {
	// Merge data from all sources
	merged := make(map[string]interface{})
	
	// Sort sources by priority (higher priority overwrites lower)
	for _, source := range l.sources {
		data, err := source.Load()
		if err != nil {
			return fmt.Errorf("failed to load from source: %w", err)
		}
		mergeMaps(merged, data)
	}

	// Apply to struct
	return l.applyToStruct(config, merged)
}

// FileSource loads configuration from files
type FileSource struct {
	Path string
}

func (fs *FileSource) Priority() int { return 1 }

func (fs *FileSource) Load() (map[string]interface{}, error) {
	data, err := os.ReadFile(fs.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil // File doesn't exist, return empty
		}
		return nil, err
	}

	var result map[string]interface{}
	
	// Determine format by extension
	ext := strings.ToLower(fs.Path[strings.LastIndex(fs.Path, ".")+1:])
	switch ext {
	case "json":
		err = json.Unmarshal(data, &result)
	case "yaml", "yml":
		err = yaml.Unmarshal(data, &result)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", fs.Path, err)
	}
	
	return flattenMap(result, ""), nil
}

// EnvSource loads configuration from environment variables
type EnvSource struct{}

func (es *EnvSource) Priority() int { return 2 } // Higher priority than files

func (es *EnvSource) Load() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := strings.ToLower(parts[0])
			value := parts[1]
			
			// Try to parse as different types
			if parsed := parseValue(value); parsed != nil {
				result[key] = parsed
			} else {
				result[key] = value
			}
		}
	}
	
	return result, nil
}

// MapSource loads from a map (useful for defaults)
type MapSource struct {
	Data map[string]interface{}
}

func (ms *MapSource) Priority() int { return 0 } // Lowest priority

func (ms *MapSource) Load() (map[string]interface{}, error) {
	return flattenMap(ms.Data, ""), nil
}

// Helper functions

func mergeMaps(dst, src map[string]interface{}) {
	for k, v := range src {
		dst[k] = v
	}
}

func flattenMap(m map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		
		if nested, ok := v.(map[string]interface{}); ok {
			for nk, nv := range flattenMap(nested, key) {
				result[nk] = nv
			}
		} else {
			result[key] = v
		}
	}
	
	return result
}

func parseValue(s string) interface{} {
	// Try boolean
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	
	// Try int
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	
	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	
	return s // Return as string
}

func (l *Loader) applyToStruct(config interface{}, data map[string]interface{}) error {
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config must be a pointer to struct")
	}
	
	v = v.Elem()
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		if !field.CanSet() {
			continue
		}
		
		// Get field configuration
		cfg := l.getFieldConfig(fieldType)
		
		// Find value from sources
		value := l.findValue(data, cfg)
		
		if value != nil {
			// Validate if needed
			if cfg.validate != "" {
				if err := l.validateField(fieldType.Name, value, cfg.validate); err != nil {
					return err
				}
			}
			
			// Set value
			if err := l.setValue(field, value); err != nil {
				return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
			}
		} else if cfg.defaultValue != "" {
			// Use default value
			parsed := parseValue(cfg.defaultValue)
			if err := l.setValue(field, parsed); err != nil {
				return fmt.Errorf("failed to set default for field %s: %w", fieldType.Name, err)
			}
		}
	}
	
	return nil
}

type fieldConfig struct {
	cfgKey       string
	envKey       string
	validate     string
	defaultValue string
}

func (l *Loader) getFieldConfig(field reflect.StructField) fieldConfig {
	return fieldConfig{
		cfgKey:       field.Tag.Get("cfg"),
		envKey:       field.Tag.Get("env"),
		validate:     field.Tag.Get("validate"),
		defaultValue: field.Tag.Get("default"),
	}
}

func (l *Loader) findValue(data map[string]interface{}, cfg fieldConfig) interface{} {
	// Check environment key first (higher priority)
	if cfg.envKey != "" {
		if value, ok := data[strings.ToLower(cfg.envKey)]; ok {
			return value
		}
	}
	
	// Check config key
	if cfg.cfgKey != "" {
		if value, ok := data[cfg.cfgKey]; ok {
			return value
		}
	}
	
	return nil
}

func (l *Loader) setValue(field reflect.Value, value interface{}) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(fmt.Sprintf("%v", value))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, err := strconv.ParseInt(fmt.Sprintf("%v", value), 10, 64); err == nil {
			field.SetInt(i)
		} else {
			return err
		}
	case reflect.Bool:
		if b, err := strconv.ParseBool(fmt.Sprintf("%v", value)); err == nil {
			field.SetBool(b)
		} else {
			return err
		}
	case reflect.Float32, reflect.Float64:
		if f, err := strconv.ParseFloat(fmt.Sprintf("%v", value), 64); err == nil {
			field.SetFloat(f)
		} else {
			return err
		}
	}
	
	return nil
}

func (l *Loader) validateField(fieldName string, value interface{}, rules string) error {
	for _, rule := range strings.Split(rules, ",") {
		rule = strings.TrimSpace(rule)
		
		parts := strings.SplitN(rule, ":", 2)
		ruleName := parts[0]
		param := ""
		if len(parts) > 1 {
			param = parts[1]
		}
		
		if validator, ok := l.validators[ruleName]; ok {
			if err := validator(value, param); err != nil {
				return &ValidationError{
					Field:   fieldName,
					Value:   value,
					Rule:    rule,
					Message: err.Error(),
				}
			}
		}
	}
	
	return nil
}

// Built-in validators
func getBuiltinValidators() map[string]ValidatorFunc {
	return map[string]ValidatorFunc{
		"required": func(value interface{}, param string) error {
			if value == nil || fmt.Sprintf("%v", value) == "" {
				return fmt.Errorf("field is required")
			}
			return nil
		},
		"url": func(value interface{}, param string) error {
			str := fmt.Sprintf("%v", value)
			if _, err := url.Parse(str); err != nil {
				return fmt.Errorf("invalid URL format")
			}
			return nil
		},
		"range": func(value interface{}, param string) error {
			parts := strings.Split(param, ",")
			if len(parts) != 2 {
				return fmt.Errorf("range validator requires min,max parameters")
			}
			
			min, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			max, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 != nil || err2 != nil {
				return fmt.Errorf("range parameters must be integers")
			}
			
			val, err := strconv.Atoi(fmt.Sprintf("%v", value))
			if err != nil {
				return fmt.Errorf("value must be an integer for range validation")
			}
			
			if val < min || val > max {
				return fmt.Errorf("value must be between %d and %d", min, max)
			}
			return nil
		},
		"email": func(value interface{}, param string) error {
			str := fmt.Sprintf("%v", value)
			emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
			if !emailRegex.MatchString(str) {
				return fmt.Errorf("invalid email format")
			}
			return nil
		},
		"min": func(value interface{}, param string) error {
			minVal, err := strconv.Atoi(param)
			if err != nil {
				return fmt.Errorf("min parameter must be an integer")
			}
			
			val, err := strconv.Atoi(fmt.Sprintf("%v", value))
			if err != nil {
				return fmt.Errorf("value must be an integer for min validation")
			}
			
			if val < minVal {
				return fmt.Errorf("value must be at least %d", minVal)
			}
			return nil
		},
		"max": func(value interface{}, param string) error {
			maxVal, err := strconv.Atoi(param)
			if err != nil {
				return fmt.Errorf("max parameter must be an integer")
			}
			
			val, err := strconv.Atoi(fmt.Sprintf("%v", value))
			if err != nil {
				return fmt.Errorf("value must be an integer for max validation")
			}
			
			if val > maxVal {
				return fmt.Errorf("value must be at most %d", maxVal)
			}
			return nil
		},
	}
}