"""Tests for the TradingAgents research adapter — fully mocked, no network/LLM."""

from __future__ import annotations

from datetime import date

import pytest

from patresearch.ai_research.tradingagents_adapter import (
    HAS_TRADINGAGENTS,
    ResearchReport,
    TradingAgentsAdapter,
    TradingAgentsAdapterConfig,
)

RUN_DATE = date(2026, 8, 29)
D = RUN_DATE.isoformat()


@pytest.fixture()
def cfg() -> TradingAgentsAdapterConfig:
    return TradingAgentsAdapterConfig(symbol="XAUUSD", provider_symbol="GLD")


class TestResearchReport:
    def test_normalization_bounds_bias(self):
        r = ResearchReport(run_date=D, symbol="XAUUSD", bias="moon", confidence=5.0)
        n = r.normalized()
        assert n.bias == "NEUTRAL"
        assert n.confidence == 1.0

    def test_negative_confidence_clamped(self):
        r = ResearchReport(run_date=D, symbol="XAUUSD", bias="BULLISH", confidence=-2)
        assert r.normalized().confidence == 0.0

    def test_impact_bullish(self):
        r = ResearchReport(run_date=D, symbol="XAUUSD", bias="BULLISH", confidence=0.8)
        assert r.impact() == pytest.approx(48.0)

    def test_impact_bearish_neutral(self):
        assert (
            ResearchReport(run_date=D, symbol="XAUUSD", bias="BEARISH", confidence=0.5).impact()
            == pytest.approx(-30.0)
        )
        assert ResearchReport(run_date=D, symbol="XAUUSD").impact() == 0.0

    def test_rejected_report_has_zero_impact(self):
        r = ResearchReport(run_date=D, symbol="XAUUSD", bias="BULLISH", confidence=0.9, quality="REJECTED")
        assert r.impact() == 0.0

    def test_row_provenance_marks_non_deterministic(self):
        r = ResearchReport(run_date=D, symbol="XAUUSD", bias="BULLISH", confidence=0.9)
        row = r.to_row()
        assert row["provenance"]["deterministic"] is False
        assert row["quality"] == "GENERATED"
        assert row["bias"] == "BULLISH"


class TestAdapterDisabled:
    def test_unavailable_returns_rejected_neutral(self, cfg):
        adapter = TradingAgentsAdapter(cfg)
        report = adapter.run(RUN_DATE)
        # Dependency may or may not be installed, but adapter disabled => rejected run.
        if not cfg.enabled or not HAS_TRADINGAGENTS:
            assert report.quality == "REJECTED"
            assert report.bias == "NEUTRAL"
            assert report.impact() == 0.0

    def test_disabled_adapter_never_raises(self, cfg):
        adapter = TradingAgentsAdapter(cfg)
        for _ in range(2):
            r = adapter.run(RUN_DATE)
            assert r.run_date == D
            assert r.symbol == "XAUUSD"


class TestDecisionMapping:
    def _adapter_with_decision(self, decision, cfg):
        adapter = TradingAgentsAdapter(cfg)
        return adapter._map_decision(D, decision)

    def test_buy_maps_bullish(self, cfg):
        r = TradingAgentsAdapter(cfg)._map_decision(D, {"action": "BUY", "confidence": 0.7})
        assert r.bias == "BULLISH"
        assert r.confidence == pytest.approx(0.7)

    def test_short_maps_bearish(self, cfg):
        r = TradingAgentsAdapter(cfg)._map_decision(D, {"action": "SELL SHORT", "confidence": 65})
        assert r.bias == "BEARISH"
        assert r.confidence == pytest.approx(0.65)

    def test_percent_confidence_normalized(self, cfg):
        r = TradingAgentsAdapter(cfg)._map_decision(D, {"action": "BUY", "confidence": 90})
        assert r.confidence == pytest.approx(0.9)

    def test_unknown_shape_degrades_neutral(self, cfg):
        r = TradingAgentsAdapter(cfg)._map_decision(D, None)
        assert r.bias == "NEUTRAL"
        assert r.quality == "GENERATED"

    def test_hold_maps_neutral(self, cfg):
        r = TradingAgentsAdapter(cfg)._map_decision(D, {"action": "HOLD", "confidence": 0.5})
        assert r.bias == "NEUTRAL"

    def test_run_with_mock_graph(self, cfg):
        cfg.enabled = True
        adapter = TradingAgentsAdapter(cfg)

        class FakeGraph:
            def propagate(self, ticker, dt):
                return (None, {"action": "BUY", "confidence": 0.6, "summary": "gold bid"})

        adapter._graph = FakeGraph()
        report = adapter._map_decision(D, {"action": "BUY", "confidence": 0.6, "summary": "gold bid"})
        assert report.bias == "BULLISH"
        assert report.framework == "tradingagents"
        assert report.impact() == pytest.approx(36.0)


class FakeGraph:
    def propagate(self, ticker, date_str):
        return None, {"action": "SELL", "confidence": 0.4, "summary": "usd strength"}


def test_full_pipeline_with_stub_graph(cfg):
    cfg.enabled = True
    adapter = TradingAgentsAdapter(cfg)
    adapter._graph = FakeGraph()
    if not HAS_TRADINGAGENTS:
        pytest.skip("tradingagents optional dependency not installed")
    report = adapter.run(RUN_DATE)
    assert report.bias == "BEARISH"
    assert report.impact() < 0
    assert report.quality == "GENERATED"