// Package environmenttext holds the text in which an app keeps the environment variables of
// a service: one NAME=value line each, with blank lines and lines that start with # left out.
// It reads the variables from the text, and writes the variables that a service reads, each
// with its default.
package environmenttext

import (
	"fmt"
	"strings"
)

type Variables struct {
	valueByName map[string]string
}

func VariablesFrom(text string) (Variables, error) {
	valueByName := map[string]string{}
	for number, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, assigned := strings.Cut(line, "=")
		name = strings.TrimSpace(name)
		if !assigned || name == "" {
			return Variables{}, fmt.Errorf("line %d: want NAME=value, got %q", number+1, line)
		}
		if _, repeated := valueByName[name]; repeated {
			return Variables{}, fmt.Errorf("line %d: %s is set twice", number+1, name)
		}
		valueByName[name] = strings.TrimSpace(value)
	}
	return Variables{valueByName: valueByName}, nil
}

func (v Variables) Get(name string) string {
	return v.valueByName[name]
}
