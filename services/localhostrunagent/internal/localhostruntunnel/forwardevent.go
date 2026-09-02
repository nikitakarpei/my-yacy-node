package localhostruntunnel

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	forwardEventName    = "tcpip-forward"
	maximumHostnameByte = 253
	maximumLabelByte    = 63
)

var (
	ErrBadEvent         = errors.New("bad tunnel event")
	errNotAForwardEvent = errors.New("not a forward event")
)

type forwardEvent struct {
	Event   string `json:"event"`
	Address string `json:"address"`
}

func assignedHostnameFrom(line []byte) (string, error) {
	var reported forwardEvent

	if err := json.Unmarshal(line, &reported); err != nil {
		return "", errNotAForwardEvent
	}

	if reported.Event != forwardEventName {
		return "", errNotAForwardEvent
	}

	if !isPublicHostname(reported.Address) {
		return "", fmt.Errorf("%w: assigned address %q is not a public hostname",
			ErrBadEvent, reported.Address)
	}

	return reported.Address, nil
}

func isPublicHostname(address string) bool {
	if !strings.Contains(address, ".") {
		return false
	}

	if len(address) > maximumHostnameByte {
		return false
	}

	for label := range strings.SplitSeq(address, ".") {
		if !isHostnameLabel(label) {
			return false
		}
	}

	return true
}

func isHostnameLabel(label string) bool {
	if label == "" || len(label) > maximumLabelByte {
		return false
	}

	if label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}

	for index := range len(label) {
		character := label[index]
		lettered := (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z')
		numbered := character >= '0' && character <= '9'

		if !lettered && !numbered && character != '-' {
			return false
		}
	}

	return true
}
