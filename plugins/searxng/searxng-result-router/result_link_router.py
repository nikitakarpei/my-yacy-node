# SPDX-License-Identifier: AGPL-3.0-or-later

import hashlib
import hmac
import os
import time
import typing as t
from functools import partial
from urllib.parse import quote, urlparse

from searx.plugins import Plugin, PluginInfo

if t.TYPE_CHECKING:
    from searx.extended_types import SXNG_Request
    from searx.plugins import PluginCfg
    from searx.result_types import LegacyResult, Result
    from searx.search import SearchWithPlugins


LINK_LIFETIME_DEFAULT = 86400


class SXNGPlugin(Plugin):
    id = "result_link_router"

    def __init__(self, plg_cfg: "PluginCfg") -> None:
        super().__init__(plg_cfg)
        self.info = PluginInfo(
            id=self.id,
            name="Result link router",
            description="Route result links through visitcrawl before their destination",
            preference_section="privacy",
        )
        link_secret = os.environ.get("VISITCRAWL_LINK_SECRET")
        if not link_secret:
            raise ValueError("VISITCRAWL_LINK_SECRET must be set")
        self.link_secret = link_secret
        self.link_lifetime = link_lifetime_from(
            os.environ.get("RESULT_LINK_ROUTER_LINK_LIFETIME")
        )

    def on_result(
        self, request: "SXNG_Request", search: "SearchWithPlugins", result: "Result"
    ) -> bool:
        visited_page_url = getattr(result, "url", None)
        result.filter_urls(
            partial(self.route_through_visitcrawl, results_origin_of(request))
        )
        if visited_page_url and visited_page_url.startswith(("http://", "https://")):
            result.parsed_url = urlparse(visited_page_url)
        return True

    def route_through_visitcrawl(
        self,
        results_origin: str,
        result: "Result | LegacyResult",
        field_name: str,
        url_src: str,
    ) -> bool | str:
        if field_name != "url" or not url_src.startswith(("http://", "https://")):
            return True
        return self.visit_link_for(results_origin, url_src)

    def visit_link_for(self, results_origin: str, visited_page: str) -> str:
        expires = str(int(time.time()) + self.link_lifetime)
        return (
            f"{results_origin}/visit"
            f"?url={quote(visited_page, safe='')}"
            f"&expires={expires}"
            f"&signature={self.signature_of(expires, visited_page)}"
        )

    def signature_of(self, expires: str, visited_page: str) -> str:
        return hmac.new(
            self.link_secret.encode(),
            f"{expires}\n{visited_page}".encode(),
            hashlib.sha256,
        ).hexdigest()


def results_origin_of(request: "SXNG_Request") -> str:
    return request.host_url.rstrip("/")


def link_lifetime_from(configured: str | None) -> int:
    if configured is None:
        return LINK_LIFETIME_DEFAULT
    if not configured.isdigit() or int(configured) <= 0:
        raise ValueError(
            "RESULT_LINK_ROUTER_LINK_LIFETIME must be a positive whole number of seconds"
        )
    return int(configured)
