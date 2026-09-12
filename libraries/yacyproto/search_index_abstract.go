package yacyproto

import (
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const pathPrefixLength = yacymodel.HashLength - yacymodel.HostHashLength

func encodeSearchIndexAbstract(urlHashes []yacymodel.URLHash) string {
	if len(urlHashes) == 0 {
		return "{}"
	}

	domains := make(map[string][]string)
	for _, hash := range urlHashes {
		if hash.IsZero() {
			continue
		}
		raw := hash.String()
		hostHash := raw[len(raw)-yacymodel.HostHashLength:]
		domains[hostHash] = append(domains[hostHash], raw[:pathPrefixLength])
	}

	if len(domains) == 0 {
		return "{}"
	}

	keys := make([]string, 0, len(domains))
	for domain := range domains {
		keys = append(keys, domain)
	}
	slices.SortFunc(keys, compareBase64Strings)
	for _, domain := range keys {
		slices.SortFunc(domains[domain], compareBase64Strings)
	}

	var b strings.Builder
	b.WriteByte('{')
	for i, domain := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(domain)
		b.WriteByte(':')
		for _, path := range domains[domain] {
			b.WriteString(path)
		}
	}
	b.WriteByte('}')
	return b.String()
}

func compareBase64Strings(a, b string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		av := base64Order(a[i])
		bv := base64Order(b[i])
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

func base64Order(c byte) int {
	for i := range len(yacymodel.Alphabet) {
		if yacymodel.Alphabet[i] == c {
			return i
		}
	}
	return len(yacymodel.Alphabet) + int(c)
}

func decodeSearchIndexAbstract(abstract string) []yacymodel.URLHash {
	packed, ok := strings.CutPrefix(abstract, "{")
	if !ok {
		return nil
	}
	packed, ok = strings.CutSuffix(packed, "}")
	if !ok {
		return nil
	}

	var urlHashes []yacymodel.URLHash
	for _, group := range strings.Split(packed, ",") {
		urlHashes = append(urlHashes, urlHashesOfDomainGroup(group)...)
	}

	return urlHashes
}

func urlHashesOfDomainGroup(group string) []yacymodel.URLHash {
	domain, paths, ok := strings.Cut(group, ":")
	if !ok || len(domain) != yacymodel.HostHashLength {
		return nil
	}

	urlHashes := make([]yacymodel.URLHash, 0, len(paths)/pathPrefixLength)
	for start := 0; start+pathPrefixLength <= len(paths); start += pathPrefixLength {
		urlHash, err := yacymodel.ParseURLHash(paths[start:start+pathPrefixLength] + domain)
		if err != nil {
			continue
		}
		urlHashes = append(urlHashes, urlHash)
	}

	return urlHashes
}
