package mediatypeallowance

type MediaTypeAllowance map[string]bool

func MediaTypeAllowanceFrom(contentTypes []string) MediaTypeAllowance {
	if len(contentTypes) == 0 {
		return nil
	}
	allowance := make(MediaTypeAllowance, len(contentTypes))
	for _, mediaType := range contentTypes {
		allowance[mediaType] = true
	}

	return allowance
}

func (a MediaTypeAllowance) Admits(mediaType string) bool {
	return a == nil || a[mediaType]
}
