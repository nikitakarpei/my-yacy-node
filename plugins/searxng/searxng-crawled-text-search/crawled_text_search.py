# SPDX-License-Identifier: AGPL-3.0-or-later

from typing import Any, Protocol
from urllib.parse import urlencode

about: dict[str, Any] = {}
categories = ["general"]
paging = True

results_per_page = 10
corpustext_search_url = ""

_search_path = "/search"


class SearchResponse(Protocol):
    def json(self) -> Any: ...


def request(query: str, params: dict[str, Any]) -> dict[str, Any]:
    if not corpustext_search_url:
        raise ValueError("corpustext_search_url: must be set")
    search = {
        "query": query,
        "offset": _result_offset(params),
        "limit": results_per_page,
    }
    if params.get("language"):
        search["language"] = params["language"]
    params["method"] = "GET"
    params["url"] = "{}{}?{}".format(
        corpustext_search_url.rstrip("/"), _search_path, urlencode(search)
    )
    return params


def response(resp: SearchResponse) -> list[dict[str, str]]:
    try:
        matches = resp.json()["matches"]
    except (ValueError, KeyError, TypeError):
        return []
    return [
        {
            "title": match["title"],
            "url": match["url"],
            "content": match["snippet"],
        }
        for match in matches
    ]


def _result_offset(params: dict[str, Any]) -> int:
    return (params["pageno"] - 1) * results_per_page
