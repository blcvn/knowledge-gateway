package biz

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// ValidateJSONSchema parses the provided JSON Schema Definition and validates it
// to ensure it is a valid draft schema.
func ValidateJSONSchema(schemaStr string) error {
	sl := gojsonschema.NewStringLoader(schemaStr)
	_, err := gojsonschema.NewSchema(sl)
	if err != nil {
		return fmt.Errorf("invalid json schema definition: %w", err)
	}
	return nil
}

// ValidatePayloadAgainstSchema validates a JSON payload against a defined JSON Schema
func ValidatePayloadAgainstSchema(schemaStr string, payloadStr string) error {
	schemaLoader := gojsonschema.NewStringLoader(schemaStr)
	documentLoader := gojsonschema.NewStringLoader(payloadStr)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("validation execution error: %w", err)
	}

	if result.Valid() {
		return nil
	}

	var errMsgs []string
	for _, desc := range result.Errors() {
		errMsgs = append(errMsgs, fmt.Sprintf("- %s", desc))
	}

	return errors.New("payload validation failed:\n" + strings.Join(errMsgs, "\n"))
}
