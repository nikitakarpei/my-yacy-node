// Package processenvironmentlease carries one process environment snapshot from
// the process that owns a short-lived capability to the process that runs with
// it. The package owns the grant value, the protocol version, the frame limit,
// the validation rules, and the rule that overlays a grant on a parent process
// environment. It opens no socket and starts no process.
package processenvironmentlease

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

const (
	ProtocolVersion  = 1
	MaximumFrameByte = 64 * 1024
)

var ErrBadGrant = errors.New("bad process environment grant")

type Grant struct {
	ProcessEnvironment map[string]string
}

type grantFrame struct {
	ProtocolVersion    *int               `json:"protocol_version"`
	ProcessEnvironment *map[string]string `json:"process_environment"`
}

func GrantFrom(frame []byte) (Grant, error) {
	if len(frame) > MaximumFrameByte {
		return Grant{}, fmt.Errorf(
			"%w: frame of %d bytes is longer than %d",
			ErrBadGrant, len(frame), MaximumFrameByte,
		)
	}

	var decoded grantFrame
	if err := json.Unmarshal(frame, &decoded, json.RejectUnknownMembers(true)); err != nil {
		return Grant{}, fmt.Errorf("%w: %w", ErrBadGrant, err)
	}

	return grantOf(decoded)
}

func grantOf(frame grantFrame) (Grant, error) {
	if frame.ProtocolVersion == nil {
		return Grant{}, fmt.Errorf("%w: protocol_version is absent", ErrBadGrant)
	}
	if *frame.ProtocolVersion != ProtocolVersion {
		return Grant{}, fmt.Errorf(
			"%w: protocol_version %d is not %d",
			ErrBadGrant, *frame.ProtocolVersion, ProtocolVersion,
		)
	}
	if frame.ProcessEnvironment == nil {
		return Grant{}, fmt.Errorf("%w: process_environment is absent", ErrBadGrant)
	}
	if err := processEnvironmentFault(*frame.ProcessEnvironment); err != nil {
		return Grant{}, err
	}

	return Grant{ProcessEnvironment: *frame.ProcessEnvironment}, nil
}

func processEnvironmentFault(processEnvironment map[string]string) error {
	for _, name := range slices.Sorted(maps.Keys(processEnvironment)) {
		if !isEnvironmentName(name) {
			return fmt.Errorf("%w: %q is not an environment name", ErrBadGrant, name)
		}
		if !isEnvironmentValue(processEnvironment[name]) {
			return fmt.Errorf("%w: the value of %s is not an environment value", ErrBadGrant, name)
		}
	}

	return nil
}

func isEnvironmentName(candidate string) bool {
	return candidate != "" &&
		!strings.Contains(candidate, "=") &&
		!strings.ContainsRune(candidate, 0)
}

func isEnvironmentValue(candidate string) bool {
	return !strings.ContainsRune(candidate, 0)
}

func FrameOf(grant Grant) ([]byte, error) {
	if err := processEnvironmentFault(grant.ProcessEnvironment); err != nil {
		return nil, err
	}

	processEnvironment := grant.ProcessEnvironment
	if processEnvironment == nil {
		processEnvironment = map[string]string{}
	}
	version := ProtocolVersion

	frame, err := json.Marshal(grantFrame{
		ProtocolVersion:    &version,
		ProcessEnvironment: &processEnvironment,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadGrant, err)
	}
	frame = append(frame, '\n')

	if len(frame) > MaximumFrameByte {
		return nil, fmt.Errorf(
			"%w: frame of %d bytes is longer than %d",
			ErrBadGrant, len(frame), MaximumFrameByte,
		)
	}

	return frame, nil
}

func (grant Grant) LeasedProcessEnvironmentFrom(parentEnvironment []string) []string {
	leasedValues := make(
		map[string]string,
		len(parentEnvironment)+len(grant.ProcessEnvironment),
	)
	for _, entry := range parentEnvironment {
		if name, value, found := strings.Cut(entry, "="); found {
			leasedValues[name] = value
		}
	}
	maps.Copy(leasedValues, grant.ProcessEnvironment)

	leased := make([]string, 0, len(leasedValues))
	for _, name := range slices.Sorted(maps.Keys(leasedValues)) {
		leased = append(leased, name+"="+leasedValues[name])
	}

	return leased
}
