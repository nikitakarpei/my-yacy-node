package yacyproto

import (
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const pathPrefixLength = yacymodel.HashLength - yacymodel.HostHashLength

func encodeSearchIndexAbstract(urlHashes yacymodel.URLHashes) string {
	if urlHashes.Amount() == 0 {
		return "{}"
	}

	domains := make(map[string][]string)
	for hash := range urlHashes.All() {
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
	slices.SortFunc(keys, yacymodel.CompareInAlphabetOrder)
	for _, domain := range keys {
		slices.SortFunc(domains[domain], yacymodel.CompareInAlphabetOrder)
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

func decodeSearchIndexAbstract(abstract string) yacymodel.URLHashes {
	packed, ok := strings.CutPrefix(abstract, "{")
	if !ok {
		return yacymodel.URLHashes{}
	}
	packed, ok = strings.CutSuffix(packed, "}")
	if !ok {
		return yacymodel.URLHashes{}
	}

	var urlHashes yacymodel.URLHashesBuilder
	urlHashes.Grow(amountOfURLHashesIn(packed))
	for group := range strings.SplitSeq(packed, ",") {
		addURLHashesOfDomainGroup(&urlHashes, group)
	}

	return urlHashes.URLHashes()
}

func amountOfURLHashesIn(packed string) int {
	amountOfDomainGroups := strings.Count(packed, ",") + 1
	separatorsAndDomains := amountOfDomainGroups - 1 + amountOfDomainGroups*(yacymodel.HostHashLength+1)

	return max(0, len(packed)-separatorsAndDomains) / pathPrefixLength
}

func addURLHashesOfDomainGroup(urlHashes *yacymodel.URLHashesBuilder, group string) {
	domain, paths, ok := strings.Cut(group, ":")
	if !ok || len(domain) != yacymodel.HostHashLength {
		return
	}

	for start := 0; start+pathPrefixLength <= len(paths); start += pathPrefixLength {
		urlHash, err := yacymodel.ParseURLHash(paths[start:start+pathPrefixLength] + domain)
		if err != nil {
			continue
		}
		urlHashes.Add(urlHash)
	}
}
