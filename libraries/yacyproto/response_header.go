package yacyproto

type ResponseHeader struct {
	Version string
	Uptime  int
}

func InjectResponseHeader(dst ResponseFields, version string, uptime int) {
	setString(dst, FieldVersion, version)
	setInt(dst, FieldUptime, uptime)
}

func parseResponseHeader(m ResponseFields) (ResponseHeader, error) {
	uptime, err := optionalInt(FieldUptime, m[FieldUptime])
	if err != nil {
		return ResponseHeader{}, err
	}

	return ResponseHeader{Version: m[FieldVersion], Uptime: uptime}, nil
}
