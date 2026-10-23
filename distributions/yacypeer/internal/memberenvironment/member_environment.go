// Package memberenvironment gives each member of the peer the settings it reads: the value
// the peer wires, else the value the operator sets for that member, else the value the
// operator sets for every member. It refuses a setting the operator must not set.
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

func For(
	member string,
	wiring Wiring,
	unreadSettings []string,
	process func(string) string,
) (Environment, error) {
	environment := Environment{
		wiring:       wiring,
		memberPrefix: strings.ToUpper(member) + memberSeparator,
		process:      process,
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
	if e.isSet(e.memberPrefix + key) {
		return e.process(e.memberPrefix + key)
	}
	return e.process(key)
}

func (e Environment) refused(keys []string, reason string) error {
	var refusals []error
	for _, key := range keys {
		for _, name := range []string{key, e.memberPrefix + key} {
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
