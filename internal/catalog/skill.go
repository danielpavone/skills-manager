package catalog

import "fmt"

type Skill struct {
	Name       string `json:"name"`
	SourcePath string `json:"source_path"`
}

type ErrorCode string

const ErrorCatalogInvalid ErrorCode = "catalog_invalid"

type DomainError struct {
	Code     ErrorCode
	Value    string
	Expected string
	Cause    error
}

func (e *DomainError) Error() string {
	message := fmt.Sprintf("%s %q: esperado %s", e.Code, e.Value, e.Expected)
	if e.Cause == nil {
		return message
	}
	return fmt.Sprintf("%s: %v", message, e.Cause)
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}
