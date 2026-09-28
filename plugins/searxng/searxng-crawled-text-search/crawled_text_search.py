# SPDX-License-Identifier: AGPL-3.0-or-later

import json
from typing import Any, Protocol

about: dict[str, Any] = {}
categories = ["general"]
paging = True

results_per_page = 10
search_index_engine = ""
elasticsearch_url = ""
elasticsearch_index = "yacy_text_v1"
manticore_url = ""
manticore_table = "yacy_text_v1"

_content_fragment_length = 300
_title_weight = 3


class SearchResponse(Protocol):
    def json(self) -> Any: ...


def request(query: str, params: dict[str, Any]) -> dict[str, Any]:
    params["method"] = "POST"
    params["headers"]["Content-Type"] = "application/json"
    if search_index_engine == "manticore":
        _manticore_request(query, params)
    elif search_index_engine == "elasticsearch":
        _elasticsearch_request(query, params)
    else:
        raise ValueError(f"unknown search_index_engine: {search_index_engine}")
    return params


def response(resp: SearchResponse) -> list[dict[str, str]]:
    try:
        hits = resp.json()["hits"]["hits"]
    except (ValueError, KeyError, TypeError):
        return []

    results = []
    for hit in hits:
        source = hit.get("_source", {})
        title = source.get("title")
        url = source.get("url")
        if not title or not url:
            continue
        results.append(
            {
                "title": title,
                "url": url,
                "content": _matched_content(hit, source),
            }
        )
    return results


def _elasticsearch_request(query: str, params: dict[str, Any]) -> None:
    params["url"] = "{}/{}_*/_search".format(
        elasticsearch_url.rstrip("/"), elasticsearch_index
    )
    params["data"] = json.dumps(
        {
            "from": _result_offset(params),
            "size": results_per_page,
            "query": _elasticsearch_query(query, _search_language(params)),
            "highlight": {
                "fields": {"content": {}},
                "pre_tags": [""],
                "post_tags": [""],
            },
        }
    )


def _elasticsearch_query(query: str, language: str) -> dict[str, Any]:
    match = {
        "combined_fields": {
            "query": query,
            "fields": [f"title^{_title_weight}", "content"],
            "operator": "and",
        }
    }
    if not language:
        return match
    return {"bool": {"must": [match], "filter": [{"term": {"language": language}}]}}


def _manticore_request(query: str, params: dict[str, Any]) -> None:
    params["url"] = "{}/search".format(manticore_url.rstrip("/"))
    params["data"] = json.dumps(
        {
            "table": manticore_table,
            "offset": _result_offset(params),
            "limit": results_per_page,
            "query": _manticore_query(query, _search_language(params)),
            "options": {
                "field_weights": {"title": _title_weight, "content": 1},
            },
            "highlight": {
                "fields": ["content"],
                "before_match": "",
                "after_match": "",
            },
        }
    )


def _manticore_query(query: str, language: str) -> dict[str, Any]:
    match = {"match": {"title,content": {"query": query, "operator": "and"}}}
    if not language:
        return match
    return {"bool": {"must": [match, {"equals": {"language": language}}]}}


def _search_language(params: dict[str, Any]) -> str:
    language = str(params.get("language") or "").strip().lower()
    if language in ("", "all"):
        return ""
    return language.split("-")[0]


def _result_offset(params: dict[str, Any]) -> int:
    return (params["pageno"] - 1) * results_per_page


def _matched_content(hit: dict[str, Any], source: dict[str, Any]) -> str:
    fragments = hit.get("highlight", {}).get("content")
    if fragments:
        return " … ".join(fragments)
    return source.get("content", "")[:_content_fragment_length]
