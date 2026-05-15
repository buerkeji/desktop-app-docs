package model

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" && trimmed != "<nil>" {
			return trimmed
		}
	}
	return ""
}

func MustJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func CompactJSONString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}

	data, err := json.Marshal(value)
	if err != nil {
		return raw
	}

	return string(data)
}

func ToStringAnyMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return map[string]any{}
		}
		result := make(map[string]any)
		if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
			return map[string]any{}
		}
		return result
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return map[string]any{}
		}
		result := make(map[string]any)
		if err := json.Unmarshal(data, &result); err != nil {
			return map[string]any{}
		}
		return result
	}
}

func Int64FromAny(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case json.Number:
		result, _ := typed.Int64()
		return result
	case string:
		typed = strings.TrimSpace(typed)
		if typed == "" {
			return 0
		}
		result, err := strconv.ParseInt(typed, 10, 64)
		if err == nil {
			return result
		}
		floatResult, floatErr := strconv.ParseFloat(typed, 64)
		if floatErr == nil {
			return int64(floatResult)
		}
	}
	return 0
}

func ResponseRequestID(response *RemoteDesktopAPIResponse) string {
	if response == nil {
		return ""
	}
	return strings.TrimSpace(response.RequestID)
}

func RequestIDFromHeaders(headers map[string]string) string {
	for key, value := range headers {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "x-request-id", "x-idempotency-key":
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func NormaliseSubmitStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success", "failed", "partial", "pending":
		return strings.ToLower(strings.TrimSpace(status))
	default:
		return "success"
	}
}

func NormaliseLogLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "info", "warn", "error":
		return strings.ToLower(strings.TrimSpace(level))
	default:
		return "info"
	}
}

func NormaliseTaskStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success", "failed", "pending":
		return strings.ToLower(strings.TrimSpace(status))
	default:
		return "success"
	}
}

func NormaliseUploadScene(scene string) string {
	return strings.TrimSpace(scene)
}

func FirstNonEmptyTimestamp(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func NormaliseDateBoundary(value string, endOfDay bool) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.Format(time.RFC3339)
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		if endOfDay {
			parsed = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		return parsed.Format(time.RFC3339)
	}

	return value
}

func NullableInt64(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}
