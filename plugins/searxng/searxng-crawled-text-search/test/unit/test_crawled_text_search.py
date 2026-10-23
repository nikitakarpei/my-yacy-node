# SPDX-License-Identifier: AGPL-3.0-or-later

import importlib
from typing import Any
from urllib.parse import parse_qs, urlsplit

import pytest

crawled_text_search = importlib.import_module("crawled_text_search")


@pytest.fixture(autouse=True)
def configured(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(
        crawled_text_search, "corpustext_search_url", "http://corpustext:8096/"
    )


def build_params(pageno: int = 1, language: str | None = None) -> dict[str, Any]:
    params: dict[str, Any] = {
        "pageno": pageno,
        "headers": {},
        "url": "",
        "method": "POST",
        "data": "",
    }
    if language is not None:
        params["language"] = language
    return params


def search_of(params: dict[str, Any]) -> dict[str, list[str]]:
    return parse_qs(urlsplit(params["url"]).query)


def test_request_asks_the_corpustext_search() -> None:
    params = crawled_text_search.request("wildflower", build_params())
    target = urlsplit(params["url"])
    assert params["method"] == "GET"
    assert (target.scheme, target.netloc, target.path) == (
        "http",
        "corpustext:8096",
        "/search",
    )
    assert search_of(params)["query"] == ["wildflower"]


def test_request_paginates_from_pageno(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(crawled_text_search, "results_per_page", 7)
    params = crawled_text_search.request("wildflower", build_params(pageno=3))
    search = search_of(params)
    assert search["offset"] == ["14"]
    assert search["limit"] == ["7"]


@pytest.mark.parametrize("language", ["en", "en-US", "all"])
def test_request_passes_the_search_language(language: str) -> None:
    params = crawled_text_search.request("wildflower", build_params(language=language))
    assert search_of(params)["language"] == [language]


@pytest.mark.parametrize("language", [None, ""])
def test_request_without_a_search_language_names_none(language: str | None) -> None:
    params = crawled_text_search.request("wildflower", build_params(language=language))
    assert "language" not in search_of(params)


def test_request_rejects_an_unset_corpustext_search_url(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setattr(crawled_text_search, "corpustext_search_url", "")
    with pytest.raises(ValueError):
        crawled_text_search.request("wildflower", build_params())


class FakeResponse:
    def __init__(self, payload: Any) -> None:
        self._payload = payload

    def json(self) -> Any:
        return self._payload


def test_response_maps_a_match_to_a_result() -> None:
    resp = FakeResponse(
        {
            "matches": [
                {
                    "title": "Riverside Wildflower Guide",
                    "url": "https://example.invalid/wildflower-guide",
                    "snippet": "A field guide to wildflowers.",
                }
            ]
        }
    )
    assert crawled_text_search.response(resp) == [
        {
            "title": "Riverside Wildflower Guide",
            "url": "https://example.invalid/wildflower-guide",
            "content": "A field guide to wildflowers.",
        }
    ]


def test_response_returns_empty_list_on_malformed_body() -> None:
    assert crawled_text_search.response(FakeResponse({"unexpected": "shape"})) == []


def test_response_returns_empty_list_when_json_raises() -> None:
    class RaisingResponse:
        def json(self) -> Any:
            raise ValueError("not json")

    assert crawled_text_search.response(RaisingResponse()) == []
