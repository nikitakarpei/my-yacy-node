// Package memberenvironment gives each member of the peer the settings it reads: the value
// the peer wires, else the value the operator sets for that member, else the value the
// operator sets for every member.
package memberenvironment

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

const memberSeparator = "__"

type Wiring map[string]string

type Environment struct {
	wiring       Wiring
	memberPrefix string
	process      func(string) string
}

func For(member string, wiring Wiring, process func(string) string) (Environment, error) {
	environment := Environment{
		wiring:       wiring,
		memberPrefix: strings.ToUpper(member) + memberSeparator,
		process:      process,
	}
	if err := environment.overriddenWiring(); err != nil {
		return Environment{}, err
	}
	return environment, nil
}

func (e Environment) Lookup(key string) string {
	if value, wired := e.wiring[key]; wired {
		return value
	}
	if value := e.process(e.memberPrefix + key); strings.TrimSpace(value) != "" {
		return value
	}
	return e.process(key)
}

func (e Environment) overriddenWiring() error {
	var overrides []error
	for _, key := range slices.Sorted(maps.Keys(e.wiring)) {
		for _, name := range []string{key, e.memberPrefix + key} {
			if strings.TrimSpace(e.process(name)) != "" {
				overrides = append(overrides, wiredSettingOverridden(name))
			}
		}
	}
	return errors.Join(overrides...)
}

func wiredSettingOverridden(name string) error {
	return fmt.Errorf("%s: wired by yacypeer, must not be set", name)
}
