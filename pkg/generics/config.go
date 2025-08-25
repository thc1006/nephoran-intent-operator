//go:build go1.24

package generics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Configurable defines an interface for types that can be configured.
type Configurable[T any] interface {
	Configure(config T) Result[T, error]
	Validate() Result[bool, error]
	Default() T
}

// ConfigSource defines where configuration can be loaded from.
type ConfigSource int

const (
	ConfigSourceEnv ConfigSource = iota
	ConfigSourceFile
	ConfigSourceKubernetes
	ConfigSourceVault
	ConfigSourceConsul
)

// ConfigProvider defines an interface for configuration providers.
type ConfigProvider[T any] interface {
	Load(ctx context.Context, key string) Result[T, error]
	Store(ctx context.Context, key string, value T) Result[bool, error]
	Watch(ctx context.Context, key string) <-chan Result[T, error]
	Close() error
}

// ConfigManager manages configuration with type safety and validation.
type ConfigManager[T any] struct {
	providers  map[ConfigSource]ConfigProvider[T]
	validators []ConfigValidator[T]
	transforms []ConfigTransform[T]
	cache      *Map[string, ConfigEntry[T]]
	defaults   T
}

// ConfigEntry represents a cached configuration entry.
type ConfigEntry[T any] struct {
	Value     T
	Source    ConfigSource
	Timestamp time.Time
	Version   string
}

// ConfigValidator validates configuration values.
type ConfigValidator[T any] func(T) Result[bool, error]

// ConfigTransform transforms configuration values.
type ConfigTransform[T any] func(T) Result[T, error]

// NewConfigManager creates a new configuration manager.
func NewConfigManager[T any](defaults T) *ConfigManager[T] {
	return &ConfigManager[T]{
		providers: make(map[ConfigSource]ConfigProvider[T]),
		cache:     NewMap[string, ConfigEntry[T]](),
		defaults:  defaults,
	}
}

// RegisterProvider registers a configuration provider.
func (cm *ConfigManager[T]) RegisterProvider(source ConfigSource, provider ConfigProvider[T]) {
	cm.providers[source] = provider
}

// AddValidator adds a configuration validator.
func (cm *ConfigManager[T]) AddValidator(validator ConfigValidator[T]) {
	cm.validators = append(cm.validators, validator)
}

// AddTransform adds a configuration transformer.
func (cm *ConfigManager[T]) AddTransform(transform ConfigTransform[T]) {
	cm.transforms = append(cm.transforms, transform)
}

// Load loads configuration with cascading from multiple sources.
func (cm *ConfigManager[T]) Load(ctx context.Context, key string, sources ...ConfigSource) Result[T, error] {
	if len(sources) == 0 {
		sources = []ConfigSource{ConfigSourceEnv, ConfigSourceFile, ConfigSourceKubernetes}
	}

	// Check cache first
	if entry := cm.cache.Get(key); entry.IsSome() {
		cacheEntry := entry.Value()
		// Consider cache valid for 5 minutes
		if time.Since(cacheEntry.Timestamp) < 5*time.Minute {
			return Ok[T, error](cacheEntry.Value)
		}
	}

	// Try each source in priority order
	var lastErr error
	for _, source := range sources {
		if provider, exists := cm.providers[source]; exists {
			result := provider.Load(ctx, key)
			if result.IsOk() {
				config := result.Value()

				// Apply transforms
				for _, transform := range cm.transforms {
					transformResult := transform(config)
					if transformResult.IsErr() {
						lastErr = transformResult.Error()
						continue
					}
					config = transformResult.Value()
				}

				// Validate configuration
				for _, validator := range cm.validators {
					validationResult := validator(config)
					if validationResult.IsErr() {
						lastErr = validationResult.Error()
						continue
					}
					if !validationResult.Value() {
						lastErr = fmt.Errorf("configuration validation failed for key: %s", key)
						continue
					}
				}

				// Cache the result
				entry := ConfigEntry[T]{
					Value:     config,
					Source:    source,
					Timestamp: time.Now(),
					Version:   fmt.Sprintf("%d", time.Now().Unix()),
				}
				cm.cache.Set(key, entry)

				return Ok[T, error](config)
			}
			lastErr = result.Error()
		}
	}

	// Return defaults if all sources fail
	if !isZero(cm.defaults) {
		return Ok[T, error](cm.defaults)
	}

	return Err[T, error](fmt.Errorf("failed to load config for key %s: %w", key, lastErr))
}

// Store stores configuration to the specified source.
func (cm *ConfigManager[T]) Store(ctx context.Context, key string, value T, source ConfigSource) Result[bool, error] {
	provider, exists := cm.providers[source]
	if !exists {
		return Err[bool, error](fmt.Errorf("no provider registered for source: %d", source))
	}

	// Validate before storing
	for _, validator := range cm.validators {
		result := validator(value)
		if result.IsErr() {
			return Err[bool, error](result.Error())
		}
		if !result.Value() {
			return Err[bool, error](fmt.Errorf("validation failed for key: %s", key))
		}
	}

	// Store the configuration
	result := provider.Store(ctx, key, value)
	if result.IsOk() {
		// Update cache
		entry := ConfigEntry[T]{
			Value:     value,
			Source:    source,
			Timestamp: time.Now(),
			Version:   fmt.Sprintf("%d", time.Now().Unix()),
		}
		cm.cache.Set(key, entry)
	}

	return result
}

// Watch watches for configuration changes.
func (cm *ConfigManager[T]) Watch(ctx context.Context, key string, source ConfigSource) <-chan Result[T, error] {
	provider, exists := cm.providers[source]
	if !exists {
		resultChan := make(chan Result[T, error], 1)
		resultChan <- Err[T, error](fmt.Errorf("no provider registered for source: %d", source))
		close(resultChan)
		return resultChan
	}

	return provider.Watch(ctx, key)
}

// Reload reloads all cached configurations.
func (cm *ConfigManager[T]) Reload(ctx context.Context) Result[int, error] {
	reloaded := 0
	keys := cm.cache.Keys()

	for _, key := range keys {
		entry := cm.cache.Get(key)
		if entry.IsNone() {
			continue
		}

		cacheEntry := entry.Value()
		result := cm.Load(ctx, key, cacheEntry.Source)
		if result.IsOk() {
			reloaded++
		}
	}

	return Ok[int, error](reloaded)
}

// Close closes all providers.
func (cm *ConfigManager[T]) Close() error {
	var errs []error

	for _, provider := range cm.providers {
		if err := provider.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing providers: %v", errs)
	}

	return nil
}

// EnvironmentProvider loads configuration from environment variables.
type EnvironmentProvider[T any] struct {
	prefix string
	parser EnvParser[T]
}

// EnvParser parses environment variables into type T.
type EnvParser[T any] interface {
	Parse(env map[string]string) Result[T, error]
}

// NewEnvironmentProvider creates a new environment provider.
func NewEnvironmentProvider[T any](prefix string, parser EnvParser[T]) *EnvironmentProvider[T] {
	return &EnvironmentProvider[T]{
		prefix: prefix,
		parser: parser,
	}
}

// Load loads configuration from environment variables.
func (ep *EnvironmentProvider[T]) Load(ctx context.Context, key string) Result[T, error] {
	envVars := make(map[string]string)

	// Collect all environment variables with the prefix
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envKey := parts[0]
		envValue := parts[1]

		if strings.HasPrefix(envKey, ep.prefix) {
			envVars[envKey] = envValue
		}
	}

	return ep.parser.Parse(envVars)
}

// Store is not supported for environment provider.
func (ep *EnvironmentProvider[T]) Store(ctx context.Context, key string, value T) Result[bool, error] {
	return Err[bool, error](fmt.Errorf("store not supported for environment provider"))
}

// Watch is not supported for environment provider.
func (ep *EnvironmentProvider[T]) Watch(ctx context.Context, key string) <-chan Result[T, error] {
	resultChan := make(chan Result[T, error], 1)
	resultChan <- Err[T, error](fmt.Errorf("watch not supported for environment provider"))
	close(resultChan)
	return resultChan
}

// Close closes the environment provider.
func (ep *EnvironmentProvider[T]) Close() error {
	return nil
}

// ReflectiveEnvParser uses reflection to parse environment variables.
type ReflectiveEnvParser[T any] struct{}

// Parse parses environment variables using reflection.
func (rep ReflectiveEnvParser[T]) Parse(env map[string]string) Result[T, error] {
	var config T
	configValue := reflect.ValueOf(&config).Elem()
	configType := reflect.TypeOf(config)

	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)
		fieldValue := configValue.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		// Get environment variable name from tag or field name
		envName := field.Tag.Get("env")
		if envName == "" {
			envName = strings.ToUpper(field.Name)
		}

		envValue, exists := env[envName]
		if !exists {
			// Check for default value
			if defaultValue := field.Tag.Get("default"); defaultValue != "" {
				envValue = defaultValue
			} else {
				continue
			}
		}

		// Parse the value based on field type
		if err := setFieldValue(fieldValue, envValue); err != nil {
			return Err[T, error](fmt.Errorf("failed to set field %s: %w", field.Name, err))
		}
	}

	return Ok[T, error](config)
}

// setFieldValue sets a reflect.Value from a string.
func setFieldValue(fieldValue reflect.Value, value string) error {
	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if fieldValue.Type().String() == "time.Duration" {
			duration, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			fieldValue.SetInt(int64(duration))
		} else {
			intValue, err := strconv.ParseInt(value, 10, fieldValue.Type().Bits())
			if err != nil {
				return err
			}
			fieldValue.SetInt(intValue)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, fieldValue.Type().Bits())
		if err != nil {
			return err
		}
		fieldValue.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, fieldValue.Type().Bits())
		if err != nil {
			return err
		}
		fieldValue.SetFloat(floatValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		fieldValue.SetBool(boolValue)
	case reflect.Slice:
		// Handle string slices
		if fieldValue.Type().Elem().Kind() == reflect.String {
			values := strings.Split(value, ",")
			slice := reflect.MakeSlice(fieldValue.Type(), len(values), len(values))
			for i, v := range values {
				slice.Index(i).SetString(strings.TrimSpace(v))
			}
			fieldValue.Set(slice)
		}
	default:
		return fmt.Errorf("unsupported field type: %s", fieldValue.Kind())
	}

	return nil
}

// FileProvider loads configuration from files.
type FileProvider[T any] struct {
	basePath string
	format   FileFormat
}

// FileFormat represents configuration file formats.
type FileFormat int

const (
	FormatJSON FileFormat = iota
	FormatYAML
	FormatTOML
	FormatProperties
)

// NewFileProvider creates a new file provider.
func NewFileProvider[T any](basePath string, format FileFormat) *FileProvider[T] {
	return &FileProvider[T]{
		basePath: basePath,
		format:   format,
	}
}

// Load loads configuration from a file.
func (fp *FileProvider[T]) Load(ctx context.Context, key string) Result[T, error] {
	filename := fp.getFilename(key)

	data, err := os.ReadFile(filename)
	if err != nil {
		return Err[T, error](fmt.Errorf("failed to read config file %s: %w", filename, err))
	}

	var config T

	switch fp.format {
	case FormatJSON:
		if err := json.Unmarshal(data, &config); err != nil {
			return Err[T, error](fmt.Errorf("failed to unmarshal JSON config: %w", err))
		}
	default:
		return Err[T, error](fmt.Errorf("unsupported file format: %d", fp.format))
	}

	return Ok[T, error](config)
}

// Store stores configuration to a file.
func (fp *FileProvider[T]) Store(ctx context.Context, key string, value T) Result[bool, error] {
	filename := fp.getFilename(key)

	var data []byte
	var err error

	switch fp.format {
	case FormatJSON:
		data, err = json.MarshalIndent(value, "", "  ")
		if err != nil {
			return Err[bool, error](fmt.Errorf("failed to marshal JSON config: %w", err))
		}
	default:
		return Err[bool, error](fmt.Errorf("unsupported file format: %d", fp.format))
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return Err[bool, error](fmt.Errorf("failed to write config file %s: %w", filename, err))
	}

	return Ok[bool, error](true)
}

// Watch watches for file changes (simplified implementation).
func (fp *FileProvider[T]) Watch(ctx context.Context, key string) <-chan Result[T, error] {
	resultChan := make(chan Result[T, error])

	go func() {
		defer close(resultChan)

		filename := fp.getFilename(key)
		lastModTime := time.Time{}

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stat, err := os.Stat(filename)
				if err != nil {
					continue
				}

				if stat.ModTime().After(lastModTime) {
					lastModTime = stat.ModTime()
					result := fp.Load(ctx, key)
					resultChan <- result
				}
			}
		}
	}()

	return resultChan
}

// Close closes the file provider.
func (fp *FileProvider[T]) Close() error {
	return nil
}

// getFilename constructs the filename for a configuration key.
func (fp *FileProvider[T]) getFilename(key string) string {
	extension := ".json"
	switch fp.format {
	case FormatYAML:
		extension = ".yaml"
	case FormatTOML:
		extension = ".toml"
	case FormatProperties:
		extension = ".properties"
	}

	return fmt.Sprintf("%s/%s%s", fp.basePath, key, extension)
}

// ConfigMerger merges multiple configurations.
type ConfigMerger[T any] struct {
	mergeFunc func(T, T) Result[T, error]
}

// NewConfigMerger creates a new configuration merger.
func NewConfigMerger[T any](mergeFunc func(T, T) Result[T, error]) *ConfigMerger[T] {
	return &ConfigMerger[T]{
		mergeFunc: mergeFunc,
	}
}

// Merge merges multiple configurations.
func (cm *ConfigMerger[T]) Merge(configs ...T) Result[T, error] {
	if len(configs) == 0 {
		return Err[T, error](fmt.Errorf("no configurations to merge"))
	}

	result := configs[0]

	for i := 1; i < len(configs); i++ {
		mergeResult := cm.mergeFunc(result, configs[i])
		if mergeResult.IsErr() {
			return mergeResult
		}
		result = mergeResult.Value()
	}

	return Ok[T, error](result)
}

// Utility functions

// isZero checks if a value is the zero value of its type.
func isZero[T any](v T) bool {
	return reflect.ValueOf(v).IsZero()
}

// DeepCopy performs a deep copy of a configuration value.
func DeepCopy[T any](original T) Result[T, error] {
	data, err := json.Marshal(original)
	if err != nil {
		return Err[T, error](fmt.Errorf("failed to marshal for deep copy: %w", err))
	}

	var copy T
	if err := json.Unmarshal(data, &copy); err != nil {
		return Err[T, error](fmt.Errorf("failed to unmarshal for deep copy: %w", err))
	}

	return Ok[T, error](copy)
}

// CommonValidators provides common configuration validators.

// NonEmptyStringValidator validates that string fields are not empty.
func NonEmptyStringValidator[T any]() ConfigValidator[T] {
	return func(config T) Result[bool, error] {
		value := reflect.ValueOf(config)
		return validateNonEmptyStrings(value)
	}
}

// validateNonEmptyStrings recursively validates string fields.
func validateNonEmptyStrings(value reflect.Value) Result[bool, error] {
	switch value.Kind() {
	case reflect.String:
		if value.String() == "" {
			return Ok[bool, error](false)
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if field.CanInterface() {
				result := validateNonEmptyStrings(field)
				if result.IsErr() || !result.Value() {
					return result
				}
			}
		}
	case reflect.Ptr:
		if !value.IsNil() {
			return validateNonEmptyStrings(value.Elem())
		}
	}

	return Ok[bool, error](true)
}

// RangeValidator validates that numeric fields are within specified ranges.
func RangeValidator[T any](min, max int64) ConfigValidator[T] {
	return func(config T) Result[bool, error] {
		value := reflect.ValueOf(config)
		return validateRanges(value, min, max)
	}
}

// validateRanges recursively validates numeric ranges.
func validateRanges(value reflect.Value, min, max int64) Result[bool, error] {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue := value.Int()
		if intValue < min || intValue > max {
			return Ok[bool, error](false)
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if field.CanInterface() {
				result := validateRanges(field, min, max)
				if result.IsErr() || !result.Value() {
					return result
				}
			}
		}
	case reflect.Ptr:
		if !value.IsNil() {
			return validateRanges(value.Elem(), min, max)
		}
	}

	return Ok[bool, error](true)
}
