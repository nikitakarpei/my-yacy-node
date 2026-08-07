# SPDX-License-Identifier: AGPL-3.0-or-later

import importlib
import json

import pytest

crawled_text_search = importlib.import_module("crawled_text_search")


@pytest.fixture(autouse=True)
def configured(monkeypatch):
    monkeypatch.setattr(crawled_text_search, "search_index_engine", "elasticsearch")
    monkeypatch.setattr(
        crawled_text_search, "elasticsearch_url", "http://elasticsearch:9200"
    )
    monkeypatch.setattr(crawled_text_search, "elasticsearch_index", "yacy_text_v1")


def build_params(pageno=1, language=None):
    params = {"pageno": pageno, "headers": {}, "url": "", "method": "GET", "data": ""}
    if language is not None:
        params["language"] = language
    return params


@pytest.fixture
def manticore(monkeypatch):
    monkeypatch.setattr(crawled_text_search, "search_index_engine", "manticore")
    monkeypatch.setattr(crawled_text_search, "manticore_url", "http://manticore:9308")
    monkeypatch.setattr(crawled_text_search, "manticore_table", "yacy_text_v1")


def test_request_fans_out_over_every_language_index():
    params = crawled_text_search.request("wildflower", build_params())
    assert params["url"] == "http://elasticsearch:9200/yacy_text_v1_*/_search"
    assert params["method"] == "POST"


def test_request_body_carries_combined_fields_query():
    params = crawled_text_search.request("wildflower", build_params())
    body = json.loads(params["data"])
    assert body["query"]["combined_fields"]["query"] == "wildflower"
    assert body["query"]["combined_fields"]["fields"] == ["title^3", "content"]
    assert body["query"]["combined_fields"]["operator"] == "and"


def test_request_paginates_from_pageno(monkeypatch):
    monkeypatch.setattr(crawled_text_search, "results_per_page", 7)
    params = crawled_text_search.request("wildflower", build_params(pageno=3))
    body = json.loads(params["data"])
    assert body["from"] == 14
    assert body["size"] == 7


def test_manticore_request_targets_configured_table(manticore):
    params = crawled_text_search.request("wildflower", build_params())
    assert params["url"] == "http://manticore:9308/search"
    assert params["method"] == "POST"
    body = json.loads(params["data"])
    assert body["table"] == "yacy_text_v1"


def test_manticore_request_matches_both_fields_with_title_weight(manticore):
    params = crawled_text_search.request("wildflower", build_params())
    body = json.loads(params["data"])
    assert body["query"]["match"]["title,content"] == {
        "query": "wildflower",
        "operator": "and",
    }
    assert body["options"]["field_weights"]["title"] == 3


def test_manticore_request_paginates_from_pageno(manticore, monkeypatch):
    monkeypatch.setattr(crawled_text_search, "results_per_page", 7)
    params = crawled_text_search.request("wildflower", build_params(pageno=3))
    body = json.loads(params["data"])
    assert body["offset"] == 14
    assert body["limit"] == 7


@pytest.mark.parametrize("language", ["en", "en-US", "EN"])
def test_request_filters_elasticsearch_by_the_search_language(language):
    params = crawled_text_search.request("wildflower", build_params(language=language))
    body = json.loads(params["data"])
    assert body["query"]["bool"]["filter"] == [{"term": {"language": "en"}}]
    assert "combined_fields" in body["query"]["bool"]["must"][0]


@pytest.mark.parametrize("language", [None, "", "all"])
def test_request_without_a_search_language_filters_nothing(language):
    params = crawled_text_search.request("wildflower", build_params(language=language))
    body = json.loads(params["data"])
    assert "combined_fields" in body["query"]
    assert "bool" not in body["query"]


@pytest.mark.parametrize("language", ["ru", "ru-RU"])
def test_manticore_request_filters_by_the_search_language(manticore, language):
    params = crawled_text_search.request("wildflower", build_params(language=language))
    body = json.loads(params["data"])
    assert {"equals": {"language": "ru"}} in body["query"]["bool"]["must"]


def test_manticore_request_without_a_search_language_filters_nothing(manticore):
    params = crawled_text_search.request("wildflower", build_params(language="all"))
    body = json.loads(params["data"])
    assert "match" in body["query"]
    assert "bool" not in body["query"]


@pytest.mark.parametrize("engine", ["", "sphinx"])
def test_request_rejects_unset_or_unknown_engine(monkeypatch, engine):
    monkeypatch.setattr(crawled_text_search, "search_index_engine", engine)
    with pytest.raises(ValueError):
        crawled_text_search.request("wildflower", build_params())


class FakeResponse:
    def __init__(self, payload):
        self._payload = payload

    def json(self):
        return self._payload


def test_response_maps_hit_to_result_with_highlight():
    resp = FakeResponse(
        {
            "hits": {
                "hits": [
                    {
                        "_source": {
                            "title": "Riverside Wildflower Guide",
                            "url": "https://example.invalid/wildflower-guide",
                            "content": "A field guide to wildflowers.",
                        },
                        "highlight": {
                            "content": ["A field guide to <em>wildflowers</em>."]
                        },
                    }
                ]
            }
        }
    )
    results = crawled_text_search.response(resp)
    assert results == [
        {
            "title": "Riverside Wildflower Guide",
            "url": "https://example.invalid/wildflower-guide",
            "content": "A field guide to <em>wildflowers</em>.",
        }
    ]


def test_response_falls_back_to_truncated_content_without_highlight():
    content = "wildflower " * 100
    resp = FakeResponse(
        {
            "hits": {
                "hits": [
                    {
                        "_source": {
                            "title": "Riverside Wildflower Guide",
                            "url": "https://example.invalid/wildflower-guide",
                            "content": content,
                        }
                    }
                ]
            }
        }
    )
    results = crawled_text_search.response(resp)
    assert results[0]["content"] == content[:300]


def test_response_maps_a_manticore_hit_with_its_highlight():
    resp = FakeResponse(
        {
            "took": 0,
            "timed_out": False,
            "hits": {
                "total": 1,
                "hits": [
                    {
                        "_id": 4,
                        "_score": 1704,
                        "_source": {
                            "title": "Riverside Wildflower Guide",
                            "url": "https://example.invalid/wildflower-guide",
                            "content": "A field guide to wildflowers.",
                        },
                        "highlight": {
                            "title": ["Riverside Wildflower Guide"],
                            "content": ["A field guide to wildflowers."],
                        },
                    }
                ],
            },
        }
    )
    assert crawled_text_search.response(resp) == [
        {
            "title": "Riverside Wildflower Guide",
            "url": "https://example.invalid/wildflower-guide",
            "content": "A field guide to wildflowers.",
        }
    ]


def test_response_skips_hit_missing_title_or_url():
    resp = FakeResponse(
        {"hits": {"hits": [{"_source": {"content": "no title or url"}}]}}
    )
    assert crawled_text_search.response(resp) == []


def test_response_returns_empty_list_on_malformed_body():
    assert crawled_text_search.response(FakeResponse({"unexpected": "shape"})) == []


def test_response_returns_empty_list_when_json_raises():
    class RaisingResponse:
        def json(self):
            raise ValueError("not json")

    assert crawled_text_search.response(RaisingResponse()) == []
