# SPDX-License-Identifier: AGPL-3.0-or-later

from typing import Any, Protocol
from urllib.parse import urlencode

about: dict[str, Any] = {}
categories = ["general"]
paging = True

results_per_page = 10
corpustext_url = ""

_search_path = "/search"


class SearchResponse(Protocol):
    def json(self) -> Any: ...


def request(query: str, params: dict[str, Any]) -> dict[str, Any]:
    if not corpustext_url:
        raise ValueError("corpustext_url: must be set")
    search = {
        "query": query,
        "offset": _result_offset(params),
        "limit": results_per_page,
    }
    language = _search_language(params)
    if language:
        search["language"] = language
    params["method"] = "GET"
    params["url"] = "{}{}?{}".format(
        corpustext_url.rstrip("/"), _search_path, urlencode(search)
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
            "content": match.get("snippet", ""),
        }
        for match in matches
        if match.get("title") and match.get("url")
    ]


def _search_language(params: dict[str, Any]) -> str:
    language = str(params.get("language") or "").strip()
    if language.lower() in ("", "all"):
        return ""
    return language


def _result_offset(params: dict[str, Any]) -> int:
    return (params["pageno"] - 1) * results_per_page
