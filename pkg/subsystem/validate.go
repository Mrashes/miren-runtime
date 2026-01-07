package subsystem

import (
	"errors"
	"fmt"
)

// ValidationError represents a missing or invalid configuration field.
type ValidationError struct {
	Subsystem string
	Field     string
	Message   string
}

func (e *ValidationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s: %s", e.Subsystem, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s is required", e.Subsystem, e.Field)
}

// Validator collects validation errors for a subsystem configuration.
type Validator struct {
	subsystem string
	errs      []error
}

// NewValidator creates a validator for the named subsystem.
func NewValidator(subsystem string) *Validator {
	return &Validator{subsystem: subsystem}
}

// Required checks that a value is non-nil/non-zero and records an error if not.
func (v *Validator) Required(field string, value any) {
	if isZero(value) {
		v.errs = append(v.errs, &ValidationError{
			Subsystem: v.subsystem,
			Field:     field,
		})
	}
}

// RequiredString checks that a string is non-empty.
func (v *Validator) RequiredString(field string, value string) {
	if value == "" {
		v.errs = append(v.errs, &ValidationError{
			Subsystem: v.subsystem,
			Field:     field,
		})
	}
}

// Error returns a combined error if any validations failed, nil otherwise.
func (v *Validator) Error() error {
	if len(v.errs) == 0 {
		return nil
	}
	return errors.Join(v.errs...)
}

// isZero checks if a value is the zero value for its type.
func isZero(value any) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return v == ""
	case int, int8, int16, int32, int64:
		return v == 0
	case uint, uint8, uint16, uint32, uint64:
		return v == 0
	case float32:
		return v == 0
	case float64:
		return v == 0
	case bool:
		return !v
	default:
		// For pointers, interfaces, slices, maps, channels - check against nil
		// This is a simplified check; reflect could be more thorough but adds overhead
		return value == nil
	}
}
