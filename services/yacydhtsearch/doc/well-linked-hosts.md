# Well-linked hosts

Many other hosts link to a well-linked host. When you give the service a list of well-linked
hosts, a result of a host that is not on the list ranks lower. The service reads the list when it
starts. To use a new list, restart the service. Set the list with
`YACYDHTSEARCH_WELL_LINKED_HOSTS_FILE`, as [configuration.md](configuration.md) tells.

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
