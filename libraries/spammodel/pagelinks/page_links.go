// Package pagelinks resolves the links a page writes into canonical web
// addresses.
package pagelinks

import (
	"encoding/base64"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const encodedWebSchemePrefix = "aHR0c"

var writtenLinkPrefixes = [...]string{"http://", "https://", "/"}

type DataLink struct {
	Address canonicalurl.CanonicalURL
	Encoded bool
}

func BaseFrom(address canonicalurl.CanonicalURL, baseHRef string) canonicalurl.CanonicalURL {
	base, err := address.CanonicalURLOfLink(baseHRef)
	if err != nil {
		return address
	}
	return base
}

func WebLinksFrom(hrefs []string, base canonicalurl.CanonicalURL) []canonicalurl.CanonicalURL {
	seenHRefs := make(map[string]bool, len(hrefs))
	seenLinks := make(map[canonicalurl.CanonicalURL]bool, len(hrefs))
	links := make([]canonicalurl.CanonicalURL, 0, len(hrefs))
	for _, href := range hrefs {
		if seenHRefs[href] {
			continue
		}
		seenHRefs[href] = true
		link, err := base.CanonicalURLOfLink(href)
		if err != nil || seenLinks[link] {
			continue
		}
		seenLinks[link] = true
		links = append(links, link)
	}
	return links
}

func DataLinksFrom(dataAttributeValues []string, base canonicalurl.CanonicalURL) []DataLink {
	links := make([]DataLink, 0, len(dataAttributeValues))
	for _, value := range dataAttributeValues {
		if link, found := dataLinkOf(strings.TrimSpace(value), base); found {
			links = append(links, link)
		}
	}
	return links
}

func dataLinkOf(value string, base canonicalurl.CanonicalURL) (DataLink, bool) {
	if strings.HasPrefix(value, encodedWebSchemePrefix) {
		address, err := canonicalurl.CanonicalURLOf(decodedOf(value))
		return DataLink{Address: address, Encoded: true}, err == nil
	}
	if !isWrittenLink(value) {
		return DataLink{}, false
	}
	address, err := base.CanonicalURLOfLink(value)
	return DataLink{Address: address}, err == nil
}

func decodedOf(value string) string {
	decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(value, "="))
	if err != nil {
		return ""
	}
	return string(decoded)
}

func isWrittenLink(value string) bool {
	for _, prefix := range writtenLinkPrefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
