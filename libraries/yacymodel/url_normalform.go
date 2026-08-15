package yacymodel

import (
	"crypto/md5"
	"net/url"
	"strconv"
	"strings"
)

const (
	httpProtocol  = "http"
	httpsProtocol = "https"
	ftpProtocol   = "ftp"
	smbProtocol   = "smb"
	fileProtocol  = "file"
)

var defaultProtocolPort = map[string]int{
	httpProtocol:  80,
	httpsProtocol: 443,
	ftpProtocol:   21,
	smbProtocol:   445,
}

var sessionIDNames = []string{"phpsessionid", "phpsessid", "jsessionid", "sid"}

const anonymousFTPUser = "anonymous"

// URLNormalform is an address cut down to what decides which document it names.
// It drops the letter case, the default port, the dot segments and the session
// id, so two addresses that name one document share one normalform. It is not
// the address a caller fetched or shows.
type URLNormalform struct {
	address url.URL
}

func URLNormalformOf(address *url.URL) URLNormalform {
	return URLNormalform{address: *address}
}

func (n URLNormalform) Hash() URLHash {
	dom, subdom := n.domSubdom()
	port := strconv.Itoa(n.port())
	flag := domainID(n.host())<<2 | int(domLengthKey(len(dom)))
	if n.protocol() != httpProtocol {
		flag |= 32
	}

	var symbols strings.Builder
	symbols.WriteString(md5Base64(n.String())[:5])
	symbols.WriteByte(md5Base64(subdom + ":" + port + ":" + n.rootpath())[0])
	symbols.WriteString(md5Base64(n.protocol() + ":" + bracketedHost(n.host()) + ":" + port)[:5])
	symbols.WriteByte(Alphabet[flag&0x3f])

	return URLHash{hash: Hash{value: symbols.String()}}
}

func (n URLNormalform) HostHash() HostHash {
	return n.Hash().HostHash()
}

func (n URLNormalform) domSubdom() (dom, subdom string) {
	host := n.host()
	lastDot := -1
	if host != "" && !strings.Contains(host, ":") {
		lastDot = strings.LastIndex(host, ".")
	}
	if lastDot > 0 {
		dom = host[:lastDot]
	}
	lastDot = strings.LastIndex(dom, ".")
	if lastDot <= 0 {
		return dom, ""
	}

	return dom[lastDot+1:], dom[:lastDot]
}

func (n URLNormalform) host() string {
	return strings.ToLower(toPunycode(n.address.Hostname()))
}

func (n URLNormalform) port() int {
	if written := n.address.Port(); written != "" {
		if port, err := strconv.Atoi(written); err == nil {
			return port
		}
	}
	if port, known := defaultProtocolPort[n.protocol()]; known {
		return port
	}

	return -1
}

func (n URLNormalform) protocol() string {
	if n.protocolIsAbsent() {
		return httpProtocol
	}

	return strings.ToLower(n.address.Scheme)
}

func (n URLNormalform) protocolIsAbsent() bool {
	return n.address.Scheme == ""
}

func domLengthKey(length int) byte {
	switch {
	case length <= 8:
		return 0
	case length <= 12:
		return 1
	case length <= 16:
		return 2
	default:
		return 3
	}
}

func md5Base64(s string) string {
	sum := md5.Sum([]byte(s))

	return Encode(sum[:])
}

func (n URLNormalform) String() string {
	var form strings.Builder
	form.WriteString(n.protocol())
	form.WriteString("://")
	if host := n.host(); host != "" {
		if userInfo := n.userInfo(); userInfo != "" {
			form.WriteString(userInfo)
			form.WriteString("@")
		}
		form.WriteString(host)
	}
	if !n.portIsImplied() {
		form.WriteString(":")
		form.WriteString(strconv.Itoa(n.port()))
	}
	form.WriteString(n.file())

	return form.String()
}

func (n URLNormalform) userInfo() string {
	if n.address.User == nil {
		return ""
	}
	userInfo := n.address.User.String()
	if n.protocol() == ftpProtocol && strings.HasPrefix(userInfo, anonymousFTPUser) {
		return ""
	}

	return userInfo
}

func (n URLNormalform) portIsImplied() bool {
	if n.protocol() == fileProtocol {
		return true
	}
	defaultPort, known := defaultProtocolPort[n.protocol()]
	if !known {
		return false
	}

	return n.port() < 0 || n.port() == defaultPort
}

func (n URLNormalform) file() string {
	if n.protocolIsAbsent() || (n.address.RawQuery == "" && !n.address.ForceQuery) {
		return n.path()
	}

	query := n.address.RawQuery
	for _, sessionID := range sessionIDNames {
		lowered := strings.ToLower(query)
		if strings.HasPrefix(lowered, sessionID+"=") {
			next := strings.IndexByte(query, '&')
			if next < 0 {
				return n.path()
			}
			query = query[next+1:]
			continue
		}
		if start := strings.Index(lowered, "&"+sessionID+"="); start >= 0 {
			if next := strings.IndexByte(query[start+1:], '&'); next < 0 {
				query = query[:start]
			} else {
				query = query[:start] + query[start+1+next:]
			}
		}
	}

	return n.path() + "?" + query
}

func (n URLNormalform) path() string {
	if n.protocolIsAbsent() {
		return n.address.String()
	}
	path := n.address.EscapedPath()
	if path == "" {
		path = "/"
	}
	switch n.protocol() {
	case httpProtocol, httpsProtocol, ftpProtocol:
		return resolveBackpath(path)
	}

	return path
}

func (n URLNormalform) rootpath() string {
	path := n.path()
	if n.protocol() == fileProtocol && strings.Contains(path, "\\") {
		path = strings.ReplaceAll(path, "\\", "/")
	}
	start := 0
	end := len(path) - 1
	if len(path) > 0 && path[0] == '/' {
		start = 1
	}
	if strings.HasSuffix(path, "/") {
		end = len(path) - 2
	}
	separator := strings.IndexByte(path[start:], '/')
	if separator >= 0 {
		separator += start
	}
	if separator > 0 && separator < end {
		return path[start:separator]
	}

	return ""
}

func bracketedHost(host string) string {
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}

	return host
}
