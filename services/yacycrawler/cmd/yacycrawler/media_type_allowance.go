package main

import "fmt"

type mediaTypeAllowance map[string]bool

func mediaTypeAllowanceFrom(contentTypes []string) mediaTypeAllowance {
	if len(contentTypes) == 0 {
		return nil
	}
	allowance := make(mediaTypeAllowance, len(contentTypes))
	for _, mediaType := range contentTypes {
		allowance[mediaType] = true
	}
	return allowance
}

func (a mediaTypeAllowance) admits(mediaType string) bool {
	return a == nil || a[mediaType]
}

func ensureRegisteredMediaTypes(contentTypes []string, registered map[string]bool) error {
	for _, mediaType := range contentTypes {
		if !registered[mediaType] {
			return fmt.Errorf("%s: no extractor reads %q", EnvContentTypes, mediaType)
		}
	}
	return nil
}
