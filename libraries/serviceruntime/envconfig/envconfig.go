// Package envconfig reads typed configuration values from environment
// variables, falling back to a default when a variable is unset or blank.
package envconfig

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func String(getenv func(string) string, key, fallback string) string {
	if value := strings.TrimSpace(getenv(key)); value != "" {
		return value
	}
	return fallback
}

func Required(getenv func(string) string, key string) (string, error) {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return "", fmt.Errorf("%s: must be set", key)
	}
	return value, nil
}

func RequiredHTTPURL(getenv func(string) string, key string) (*url.URL, error) {
	raw, err := Required(getenv, key)
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%s: scheme must be http or https", key)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("%s: must include a host", key)
	}
	return parsed, nil
}

func List(getenv func(string) string, key string) []string {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return nil
	}
	var values []string
	for item := range strings.SplitSeq(raw, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func Bool(getenv func(string) string, key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return value, nil
}

func Int(getenv func(string) string, key string, fallback int) (int, error) {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return value, nil
}

func NonNegativeInt(getenv func(string) string, key string, fallback int) (int, error) {
	value, err := Int(getenv, key, fallback)
	if err != nil {
		return 0, err
	}
	if value < 0 {
		return 0, fmt.Errorf("%s: must not be negative", key)
	}
	return value, nil
}

func PositiveInt(getenv func(string) string, key string, fallback int) (int, error) {
	value, err := Int(getenv, key, fallback)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s: must be positive", key)
	}
	return value, nil
}

func PositiveInt64(getenv func(string) string, key string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s: must be positive", key)
	}
	return value, nil
}

func Duration(
	getenv func(string) string,
	key string,
	fallback time.Duration,
) (time.Duration, error) {
	value, err := parseDuration(getenv, key, fallback)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s: must be positive", key)
	}
	return value, nil
}

func NonNegativeDuration(
	getenv func(string) string,
	key string,
	fallback time.Duration,
) (time.Duration, error) {
	value, err := parseDuration(getenv, key, fallback)
	if err != nil {
		return 0, err
	}
	if value < 0 {
		return 0, fmt.Errorf("%s: must not be negative", key)
	}
	return value, nil
}

func parseDuration(
	getenv func(string) string,
	key string,
	fallback time.Duration,
) (time.Duration, error) {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return value, nil
}

func ByteSize(getenv func(string) string, key, fallback string) (int64, error) {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		raw = fallback
	}
	value, err := parseByteSize(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return value, nil
}

var byteSizeUnits = []struct {
	suffix string
	factor int64
}{
	{"TB", 1 << 40},
	{"GB", 1 << 30},
	{"MB", 1 << 20},
	{"KB", 1 << 10},
	{"B", 1},
}

func parseByteSize(raw string) (int64, error) {
	text := strings.ToUpper(strings.TrimSpace(raw))
	for _, unit := range byteSizeUnits {
		if !strings.HasSuffix(text, unit.suffix) {
			continue
		}
		digits := strings.TrimSpace(strings.TrimSuffix(text, unit.suffix))
		value, err := strconv.ParseInt(digits, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size %q: %w", raw, err)
		}
		if value < 0 {
			return 0, fmt.Errorf("invalid size %q: must not be negative", raw)
		}

		return value * unit.factor, nil
	}

	return 0, fmt.Errorf("invalid size %q: missing unit suffix", raw)
}
