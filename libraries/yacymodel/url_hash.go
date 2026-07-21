package yacymodel

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	hostHashLength = 6
	ftpHostPrefix  = "ftp."
	httpScheme     = "http://"
	ftpScheme      = "ftp://"
)

type URLHash struct{ hash Hash }

func ParseURLHash(raw string) (URLHash, error) {
	hash, err := ParseHash(raw)
	if err != nil {
		return URLHash{}, fmt.Errorf("parse url hash: %w", err)
	}

	return URLHash{hash: hash}, nil
}

func HashURL(rawURL string) (URLHash, error) {
	a := parseURLAddress(rawURL)

	dom, subdom := a.domSubdom()
	rootpath := a.rootpath()

	flag := DomainID(a.host)<<2 | int(domLengthKey(len(dom)))
	if a.protocol != "http" {
		flag |= 32
	}

	port := strconv.Itoa(a.port)

	var h strings.Builder
	h.WriteString(md5Base64(a.normalform())[:5])
	h.WriteByte(md5Base64(subdom + ":" + port + ":" + rootpath)[0])
	h.WriteString(md5Base64(a.protocol + ":" + hostForHash(a.host) + ":" + port)[:5])
	h.WriteByte(Alphabet[flag&0x3f])
	return ParseURLHash(h.String())
}

func HashURLHost(host string) (URLHash, error) {
	host = strings.Trim(strings.ToLower(host), ".")
	if host == "" {
		return URLHash{}, fmt.Errorf("parse url host: empty host")
	}

	scheme := httpScheme
	if strings.HasPrefix(host, ftpHostPrefix) {
		scheme = ftpScheme
	}
	parsed, err := url.Parse(scheme + host)
	if err != nil {
		return URLHash{}, fmt.Errorf("parse url host: %w", err)
	}
	if parsed.Hostname() == "" {
		return URLHash{}, fmt.Errorf("parse url host: invalid host %q", host)
	}

	return HashURL(scheme + host)
}

func (h URLHash) MarshalText() ([]byte, error) {
	return h.hash.MarshalText()
}

func (h *URLHash) UnmarshalText(text []byte) error {
	parsed, err := ParseURLHash(string(text))
	if err != nil {
		return err
	}
	*h = parsed

	return nil
}

func (h URLHash) Hash() Hash {
	return h.hash
}

func (h URLHash) IsZero() bool { return h.hash.IsZero() }

func (h URLHash) String() string {
	return h.hash.value
}

func (h URLHash) HostHash() string {
	return h.hash.value[HashLength-hostHashLength:]
}

func md5Base64(s string) string {
	sum := md5.Sum([]byte(s))
	return Encode(sum[:])
}

func hostForHash(host string) string {
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}
	return host
}

func domLengthKey(l int) byte {
	switch {
	case l <= 8:
		return 0
	case l <= 12:
		return 1
	case l <= 16:
		return 2
	default:
		return 3
	}
}
