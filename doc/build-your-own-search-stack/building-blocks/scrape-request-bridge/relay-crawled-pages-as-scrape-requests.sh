#!/bin/sh
set -u

until nats --server "$CRAWL_NATS_URL" consumer info YACY_CRAWL_PAGES scrape_request_bridge >/dev/null 2>&1; do
  nats --server "$CRAWL_NATS_URL" consumer add YACY_CRAWL_PAGES scrape_request_bridge \
    --pull --filter crawl.page.indexable --deliver all --ack explicit --defaults >/dev/null 2>&1
  sleep 2
done

while true; do
  page=$(nats --server "$CRAWL_NATS_URL" consumer next YACY_CRAWL_PAGES scrape_request_bridge \
    --raw --timeout 30s 2>/dev/null) || continue
  [ -n "$page" ] || continue
  printf '%s' "$page" | nats --server "$CRAWL_NATS_URL" pub scrape.request --force-stdin -J >/dev/null
done
