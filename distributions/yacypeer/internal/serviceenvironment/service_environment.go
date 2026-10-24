// Package serviceenvironment gives each service of the peer the settings it reads: the value
// the peer wires, else the value the operator sets for that service, else the value the
// operator sets for every service, else the value the peer defaults to. It refuses a setting
// the operator must not set.
package serviceenvironment

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

const serviceSeparator = "__"

type Wiring map[string]string

type Defaults map[string]string

type Environment struct {
	wiring        Wiring
	defaults      Defaults
	servicePrefix string
	process       func(string) string
}

func For(
	service string,
	wiring Wiring,
	defaults Defaults,
	unreadSettings []string,
	process func(string) string,
) (Environment, error) {
	environment := Environment{
		wiring:        wiring,
		defaults:      defaults,
		servicePrefix: strings.ToUpper(service) + serviceSeparator,
		process:       process,
	}
	if err := errors.Join(
		environment.refused(slices.Sorted(maps.Keys(wiring)), "wired by yacypeer"),
		environment.refused(unreadSettings, "not read by yacypeer"),
	); err != nil {
		return Environment{}, err
	}
	return environment, nil
}

func (e Environment) Lookup(key string) string {
	if value, wired := e.wiring[key]; wired {
		return value
	}
	if e.isSet(e.servicePrefix + key) {
		return e.process(e.servicePrefix + key)
	}
	if e.isSet(key) {
		return e.process(key)
	}
	return e.defaults[key]
}

func (e Environment) refused(keys []string, reason string) error {
	var refusals []error
	for _, key := range keys {
		for _, name := range []string{key, e.servicePrefix + key} {
			if e.isSet(name) {
				refusals = append(refusals, fmt.Errorf("%s: %s, must not be set", name, reason))
			}
		}
	}
	return errors.Join(refusals...)
}

func (e Environment) isSet(name string) bool {
	return strings.TrimSpace(e.process(name)) != ""
}
