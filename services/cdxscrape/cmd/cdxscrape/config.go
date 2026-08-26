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

	FlagArchiveURL        = "archive-url"
	FlagArchiveCollection = "archive-collection"
	FlagURL               = "url"
	FlagMatchType         = "match-type"
	FlagFrom              = "from"
	FlagTo                = "to"
	FlagLimit             = "limit"
	FlagDryRun            = "dry-run"

	DefaultMatchType = "domain"

	ScrapedMediaType  = "text/html"
	ScrapedStatusCode = http.StatusOK
)

type CommandConfig struct {
	ArchiveURL           *url.URL
	ArchiveCollection    string
	Query                cdxindex.Query
	ScrapeRequestNATSURL string
	DryRun               bool
}

func LoadCommandConfig(
	arguments []string,
	getenv func(string) string,
) (CommandConfig, error) {
	flags := flag.NewFlagSet("cdxscrape", flag.ContinueOnError)
	archiveURL := flags.String(FlagArchiveURL, "", "base address of the web archive")
	archiveCollection := flags.String(FlagArchiveCollection, "", "collection inside the archive")
	queried := flags.String(FlagURL, "", "url the archive is asked about")
	matchType := flags.String(FlagMatchType, DefaultMatchType, "exact, prefix, host, or domain")
	from := flags.String(FlagFrom, "", "earliest capture timestamp")
	to := flags.String(FlagTo, "", "latest capture timestamp")
	limit := flags.Int(FlagLimit, 0, "most captures to read")
	dryRun := flags.Bool(FlagDryRun, false, "write the scrape requests instead of publishing")
	if err := flags.Parse(arguments); err != nil {
		return CommandConfig{}, fmt.Errorf("read arguments: %w", err)
	}

	parsedArchiveURL, err := parsedFlagURL(FlagArchiveURL, *archiveURL)
	if err != nil {
		return CommandConfig{}, err
	}
	if strings.TrimSpace(*archiveCollection) == "" {
		return CommandConfig{}, fmt.Errorf("-%s is required", FlagArchiveCollection)
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
		ArchiveURL:        parsedArchiveURL,
		ArchiveCollection: *archiveCollection,
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
