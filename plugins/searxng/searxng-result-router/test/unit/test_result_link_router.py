# SPDX-License-Identifier: AGPL-3.0-or-later

import importlib

import pytest

from searx.plugins import PluginCfg

result_link_router = importlib.import_module("result_link_router")


class FakeResult:
    def __init__(self, **fields: str) -> None:
        self._fields = dict(fields)
        self.filter_urls_calls = 0
        self.url = fields.get("url")
        self.parsed_url = None

    def filter_urls(self, filter_func) -> None:
        self.filter_urls_calls += 1
        for field_name, url_src in list(self._fields.items()):
            new_url = filter_func(self, field_name, url_src)
            if isinstance(new_url, str):
                self._fields[field_name] = new_url
                if field_name == "url":
                    self.url = new_url

    def __getitem__(self, field_name: str) -> str:
        return self._fields[field_name]


@pytest.fixture
def plugin(monkeypatch):
    monkeypatch.setenv("VISITCRAWL_BASE_URL", "http://visitcrawl:8091")
    return result_link_router.SXNGPlugin(PluginCfg(active=True))


def test_rewrites_http_url(plugin):
    rewritten = plugin.route_through_visitcrawl(None, "url", "http://example.com/a")
    assert rewritten == "http://visitcrawl:8091/visit?url=http%3A%2F%2Fexample.com%2Fa"


def test_rewrites_https_url(plugin):
    rewritten = plugin.route_through_visitcrawl(
        None, "url", "https://example.com/a?b=c"
    )
    assert (
        rewritten
        == "http://visitcrawl:8091/visit?url=https%3A%2F%2Fexample.com%2Fa%3Fb%3Dc"
    )


def test_leaves_non_url_field_unchanged(plugin):
    assert (
        plugin.route_through_visitcrawl(None, "img_src", "http://example.com/a.png")
        is True
    )


def test_leaves_non_http_scheme_unchanged(plugin):
    assert plugin.route_through_visitcrawl(None, "url", "ftp://example.com/a") is True


def test_respects_configured_base_url(monkeypatch):
    monkeypatch.setenv("VISITCRAWL_BASE_URL", "https://visitcrawl.internal:9443/")
    configured = result_link_router.SXNGPlugin(PluginCfg(active=True))
    rewritten = configured.route_through_visitcrawl(
        None, "url", "https://example.com/a"
    )
    assert (
        rewritten
        == "https://visitcrawl.internal:9443/visit?url=https%3A%2F%2Fexample.com%2Fa"
    )


def test_requires_base_url_configured(monkeypatch):
    monkeypatch.delenv("VISITCRAWL_BASE_URL", raising=False)
    with pytest.raises(ValueError):
        result_link_router.SXNGPlugin(PluginCfg(active=True))


def test_on_result_rewrites_url_and_keeps_result(plugin):
    result = FakeResult(
        url="https://example.com/a", img_src="https://example.com/a.png"
    )

    kept = plugin.on_result(request=None, search=None, result=result)

    assert kept is True
    assert result.filter_urls_calls == 1
    assert (
        result["url"]
        == "http://visitcrawl:8091/visit?url=https%3A%2F%2Fexample.com%2Fa"
    )
    assert result["img_src"] == "https://example.com/a.png"


def test_on_result_shows_visited_page_as_pretty_url(plugin):
    result = FakeResult(url="https://example.com/a")

    plugin.on_result(request=None, search=None, result=result)

    assert result.parsed_url.geturl() == "https://example.com/a"


class FakeRequest:
    def __init__(self, headers: dict[str, str]) -> None:
        self.headers = headers


def test_on_result_skips_rewrite_when_disable_header_present(plugin):
    result = FakeResult(url="https://example.com/a")
    request = FakeRequest({"X-Result-Link-Router-Disable": "1"})

    kept = plugin.on_result(request=request, search=None, result=result)

    assert kept is True
    assert result.filter_urls_calls == 0
    assert result["url"] == "https://example.com/a"


def test_on_result_rewrites_when_disable_header_absent(plugin):
    result = FakeResult(url="https://example.com/a")
    request = FakeRequest({})

    plugin.on_result(request=request, search=None, result=result)

    assert result.filter_urls_calls == 1
    assert result["url"].startswith("http://visitcrawl:8091/visit?url=")


def test_disable_header_name_is_configurable(monkeypatch):
    monkeypatch.setenv("VISITCRAWL_BASE_URL", "http://visitcrawl:8091")
    monkeypatch.setenv("RESULT_LINK_ROUTER_DISABLE_HEADER", "X-Custom-Disable")
    configured = result_link_router.SXNGPlugin(PluginCfg(active=True))
    result = FakeResult(url="https://example.com/a")
    request = FakeRequest({"X-Custom-Disable": "1"})

    configured.on_result(request=request, search=None, result=result)

    assert result.filter_urls_calls == 0
    assert result["url"] == "https://example.com/a"
