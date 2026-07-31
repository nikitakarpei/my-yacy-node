# SPDX-License-Identifier: AGPL-3.0-or-later

import os
import typing as t
from urllib.parse import quote, urlparse

from searx.plugins import Plugin, PluginInfo

if t.TYPE_CHECKING:
    from searx.extended_types import SXNG_Request
    from searx.plugins import PluginCfg
    from searx.result_types import LegacyResult, Result
    from searx.search import SearchWithPlugins


DISABLE_HEADER_DEFAULT = "X-Result-Link-Router-Disable"


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
        base_url = os.environ.get("VISITCRAWL_BASE_URL")
        if not base_url:
            raise ValueError("VISITCRAWL_BASE_URL must be set")
        self.visitcrawl_base_url = base_url.rstrip("/")
        self.disable_header = os.environ.get(
            "RESULT_LINK_ROUTER_DISABLE_HEADER", DISABLE_HEADER_DEFAULT
        )

    def on_result(
        self, request: "SXNG_Request", search: "SearchWithPlugins", result: "Result"
    ) -> bool:
        if request is not None and request.headers.get(self.disable_header) is not None:
            return True
        visited_page_url = getattr(result, "url", None)
        result.filter_urls(self.route_through_visitcrawl)
        if visited_page_url and visited_page_url.startswith(("http://", "https://")):
            result.parsed_url = urlparse(visited_page_url)
        return True

    def route_through_visitcrawl(
        self, result: "Result | LegacyResult", field_name: str, url_src: str
    ) -> bool | str:
        if field_name != "url" or not url_src.startswith(("http://", "https://")):
            return True
        return f"{self.visitcrawl_base_url}/visit?url={quote(url_src, safe='')}"
