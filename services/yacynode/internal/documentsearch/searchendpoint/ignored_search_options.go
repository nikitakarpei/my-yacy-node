package searchendpoint

import "github.com/nikitakarpei/yacy-rwi-node/yacyproto"

type ignorableOption struct {
	name  string
	isSet func(yacyproto.SearchRequest) bool
}

var ignorableOptions = []ignorableOption{
	{"prefer", func(r yacyproto.SearchRequest) bool { return r.Prefer != "" }},
	{"filter", func(r yacyproto.SearchRequest) bool { return r.Filter != "" && r.Filter != ".*" }},
	{"profile", func(r yacyproto.SearchRequest) bool { return r.Profile != "" }},
	{"author", func(r yacyproto.SearchRequest) bool { return r.Author != "" }},
	{"collection", func(r yacyproto.SearchRequest) bool { return r.Collection != "" }},
	{"filetype", func(r yacyproto.SearchRequest) bool { return r.FileType != "" }},
	{"protocol", func(r yacyproto.SearchRequest) bool { return r.Protocol != "" }},
	{"timezoneOffset", func(r yacyproto.SearchRequest) bool { return r.TimezoneOffset != 0 }},
}

func ignoredOptionNames(req yacyproto.SearchRequest) []string {
	var names []string
	for _, option := range ignorableOptions {
		if option.isSet(req) {
			names = append(names, option.name)
		}
	}

	return names
}
