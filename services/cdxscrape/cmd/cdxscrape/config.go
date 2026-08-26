package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/cdxindex"
)

const (
	EnvScrapeRequestNATSURL = "SCRAPE_REQUEST_NATS_URL"

	FlagCDXURL     = "cdx-url"
	FlagReplayURL  = "replay-url"
	FlagCollection = "collection"
	FlagURL        = "url"
	FlagMatchType  = "match-type"
	FlagFrom       = "from"
	FlagTo         = "to"
	FlagLimit      = "limit"
	FlagDryRun     = "dry-run"

	DefaultMatchType = "domain"

	ScrapedMediaType  = "text/html"
	ScrapedStatusCode = http.StatusOK
)

type CommandConfig struct {
	CDXURL               *url.URL
	ReplayURL            *url.URL
	Collection           string
	Query                cdxindex.Query
	ScrapeRequestNATSURL string
	DryRun               bool
}

func LoadCommandConfig(
	arguments []string,
	getenv func(string) string,
) (CommandConfig, error) {
	flags := flag.NewFlagSet("cdxscrape", flag.ContinueOnError)
	cdxURL := flags.String(FlagCDXURL, "", "address this command asks for the index")
	replayURL := flags.String(
		FlagReplayURL,
		"",
		"address the scraper reads replays at, when it differs from -"+FlagCDXURL,
	)
	collection := flags.String(FlagCollection, "", "collection inside the archive")
	queried := flags.String(FlagURL, "", "url the archive is asked about")
	matchType := flags.String(FlagMatchType, DefaultMatchType, "exact, prefix, host, or domain")
	from := flags.String(FlagFrom, "", "earliest capture timestamp")
	to := flags.String(FlagTo, "", "latest capture timestamp")
	limit := flags.Int(FlagLimit, 0, "most captures to read")
	dryRun := flags.Bool(FlagDryRun, false, "write the scrape requests instead of publishing")
	if err := flags.Parse(arguments); err != nil {
		return CommandConfig{}, fmt.Errorf("read arguments: %w", err)
	}

	parsedCDXURL, err := parsedFlagURL(FlagCDXURL, *cdxURL)
	if err != nil {
		return CommandConfig{}, err
	}
	parsedReplayURL := parsedCDXURL
	if strings.TrimSpace(*replayURL) != "" {
		parsedReplayURL, err = parsedFlagURL(FlagReplayURL, *replayURL)
		if err != nil {
			return CommandConfig{}, err
		}
	}
	if strings.TrimSpace(*collection) == "" {
		return CommandConfig{}, fmt.Errorf("-%s is required", FlagCollection)
	}
	if strings.TrimSpace(*queried) == "" {
		return CommandConfig{}, fmt.Errorf("-%s is required", FlagURL)
	}
	natsURL := getenv(EnvScrapeRequestNATSURL)
	if !*dryRun && strings.TrimSpace(natsURL) == "" {
		return CommandConfig{}, fmt.Errorf(
			"%s is required unless -%s is given", EnvScrapeRequestNATSURL, FlagDryRun,
		)
	}

	return CommandConfig{
		CDXURL:     parsedCDXURL,
		ReplayURL:  parsedReplayURL,
		Collection: *collection,
		Query: cdxindex.Query{
			URL:        *queried,
			MatchType:  *matchType,
			MediaType:  ScrapedMediaType,
			StatusCode: ScrapedStatusCode,
			From:       *from,
			To:         *to,
			Limit:      *limit,
		},
		ScrapeRequestNATSURL: natsURL,
		DryRun:               *dryRun,
	}, nil
}

func parsedFlagURL(name string, raw string) (*url.URL, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("-%s is required", name)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("read -%s %q: %w", name, raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("-%s %q must be an http or https address", name, raw)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("-%s %q names no host", name, raw)
	}
	return parsed, nil
}
