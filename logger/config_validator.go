package logger

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ConfigValidator provides validation for logging configuration
type ConfigValidator struct {
	errors   []string
	warnings []string
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		errors:   make([]string, 0),
		warnings: make([]string, 0),
	}
}

// ValidateEnvironmentConfig validates the current environment configuration
func (cv *ConfigValidator) ValidateEnvironmentConfig() error {
	cv.validateEnvironmentVariable()
	cv.validateLogLevelVariable()
	cv.validateFilePermissions()
	cv.validateDiskSpace()

	if len(cv.errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(cv.errors, "; "))
	}

	// Log warnings if any
	if len(cv.warnings) > 0 {
		for _, warning := range cv.warnings {
			if globalLogger != nil {
				context := NewSystemContext("config_validation", "logger")
				context["warning"] = warning
				globalLogger.LogWithContext(WARN, context, "Configuration warning: %s", warning)
			}
		}
	}

	return nil
}

// validateEnvironmentVariable checks ENV variable validity
func (cv *ConfigValidator) validateEnvironmentVariable() {
	env := os.Getenv("ENV")
	if env == "" {
		cv.warnings = append(cv.warnings, "ENV variable not set, defaulting to production mode")
		return
	}

	validEnvs := []string{"production", "prod", "development", "dev", "test"}
	for _, validEnv := range validEnvs {
		if strings.ToLower(env) == validEnv {
			return
		}
	}

	cv.warnings = append(cv.warnings, fmt.Sprintf("Invalid ENV value '%s', falling back to production mode", env))
}

// validateLogLevelVariable checks LOG_LEVEL variable validity
func (cv *ConfigValidator) validateLogLevelVariable() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		return // Optional variable
	}

	validLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	for _, validLevel := range validLevels {
		if strings.ToUpper(logLevel) == validLevel {
			return
		}
	}

	cv.warnings = append(cv.warnings, fmt.Sprintf("Invalid LOG_LEVEL value '%s', using environment default", logLevel))
}

// validateFilePermissions checks if log directory can be created and written to
func (cv *ConfigValidator) validateFilePermissions() {
	env := os.Getenv("ENV")
	if env == "test" {
		return // Test mode doesn't use files
	}

	// Try to create logs directory
	if err := os.MkdirAll("logs", 0o750); err != nil {
		cv.errors = append(cv.errors, fmt.Sprintf("Cannot create logs directory: %v", err))
		return
	}

	// Try to create a test file
	testFile := "logs/.write_test"
	if file, err := os.Create(testFile); err != nil {
		cv.errors = append(cv.errors, fmt.Sprintf("Cannot write to logs directory: %v", err))
	} else {
		if err := file.Close(); err != nil {
			cv.errors = append(cv.errors, fmt.Sprintf("Cannot close test log file: %v", err))
		}
		if err := os.Remove(testFile); err != nil {
			cv.errors = append(cv.errors, fmt.Sprintf("Cannot remove test log file: %v", err))
		}
	}
}

// validateDiskSpace checks available disk space for logging
func (cv *ConfigValidator) validateDiskSpace() {
	// This is a simplified check - in production you might want more sophisticated disk space monitoring
	if maxFileSizeStr := os.Getenv("MAX_LOG_FILE_SIZE"); maxFileSizeStr != "" {
		if maxSize, err := strconv.ParseInt(maxFileSizeStr, 10, 64); err != nil {
			cv.warnings = append(cv.warnings, fmt.Sprintf("Invalid MAX_LOG_FILE_SIZE value '%s'", maxFileSizeStr))
		} else if maxSize < 1024*1024 { // Less than 1MB
			cv.warnings = append(cv.warnings, "MAX_LOG_FILE_SIZE is very small, may cause frequent rotations")
		}
	}
}

// GetValidationSummary returns a summary of validation results
func (cv *ConfigValidator) GetValidationSummary() map[string]interface{} {
	return map[string]interface{}{
		"errorCount":   len(cv.errors),
		"warningCount": len(cv.warnings),
		"errors":       cv.errors,
		"warnings":     cv.warnings,
		"isValid":      len(cv.errors) == 0,
	}
}

// ValidateGlobalConfig validates the global logging configuration
func ValidateGlobalConfig() error {
	validator := NewConfigValidator()
	return validator.ValidateEnvironmentConfig()
}
