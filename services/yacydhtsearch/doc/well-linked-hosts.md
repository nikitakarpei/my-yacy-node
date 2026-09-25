# Well-linked hosts

A list of well-linked hosts names the hosts that many other hosts link to.

## File format

The file is plain UTF-8 text with one host name on each line, such as `www.example.org`. The
service ignores blank lines, spaces at the start and end of a line, letter case, and a trailing
dot. A line gives no rank or score, and the order of the lines has no effect.

A host on the list also covers its name with or without `www.` in front. It does not cover other
subdomains: `example.org` does not cover `blog.example.org`. Write names of international
domains in punycode, such as `xn--bcher-kva.example`.

## Make the list from Common Crawl

Common Crawl publishes host ranks for each release of its web graph. This command keeps the ten
million hosts with the highest harmonic centrality:

```sh
curl -sS https://data.commoncrawl.org/projects/hyperlinkgraph/cc-main-2026-jun-jul-aug/host/cc-main-2026-jun-jul-aug-host-ranks.txt.gz \
| zcat \
| awk -F'\t' 'NR > 1 && $1 <= 10000000 {
    n = split($5, part, "."); host = part[n]
    for (i = n - 1; i >= 1; i--) host = host "." part[i]
    print host }' \
> well-linked-hosts.txt
```

Use the newest release.

## Check a list against the judged queries

`TestResultsOfHostsOffTheWellLinkedListRankLowerWithFewerSpamAndNoLessGain` orders the judged
queries with the hosts in `test/judgedqueries/testdata/well-linked-hosts.txt`. That file holds the
hosts of the Common Crawl list above that have a result in the judged queries. The test fails
when the mean gain falls by more than the tolerance of the gate, or when the spam documents in
the first ten do not become fewer. [judged-queries.md](judged-queries.md) tells how it measures.

Before you use another source, another measure or another number of hosts, write the file again
from your list and run the test from `services/yacydhtsearch`:

```sh
zcat test/judgedqueries/testdata/answers/*.json.gz \
| grep -oE '"Address": ?"[a-z]+://[^/:"]+' | cut -d/ -f3 \
| awk 'NR == FNR { site = tolower($0); sub(/\.$/, "", site); sub(/^www\./, "", site); sites[site]; next }
  { site = tolower($0); sub(/\.$/, "", site); sub(/^www\./, "", site); if (site in sites) print }' \
  - well-linked-hosts.txt | sort -u > test/judgedqueries/testdata/well-linked-hosts.txt
go test -v -run TestResultsOfHostsOffTheWellLinkedList ./test/judgedqueries/
```
