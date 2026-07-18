package yacyproto

type ResponseHeader struct {
	Version string
	Uptime  int
}

func InjectResponseHeader(dst Message, version string, uptime int) {
	setString(dst, FieldVersion, version)
	setInt(dst, FieldUptime, uptime)
}

func parseResponseHeader(m Message) (ResponseHeader, error) {
	uptime, err := optionalInt(FieldUptime, m[FieldUptime])
	if err != nil {
		return ResponseHeader{}, err
	}

	return ResponseHeader{Version: m[FieldVersion], Uptime: uptime}, nil
}
