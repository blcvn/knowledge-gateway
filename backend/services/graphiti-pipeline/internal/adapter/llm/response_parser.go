package llm

import (
	"encoding/json"
	"errors"
	"strings"
)

func ParseJSONResponse(response string, v interface{}) error {
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 || start > end {
		// try array format []
		start = strings.Index(response, "[")
		end = strings.LastIndex(response, "]")
		if start == -1 || end == -1 || start > end {
			return errors.New("invalid JSON response format")
		}
	}

	jsonStr := response[start : end+1]
	return json.Unmarshal([]byte(jsonStr), v)
}
