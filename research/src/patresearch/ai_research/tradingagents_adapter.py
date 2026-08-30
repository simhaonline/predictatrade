"""TradingAgents adapter — optional LLM research bridge for the Intelligence Plane.

Wraps the open-source TradingAgents framework (TauricResearch, Apache-2.0) as an
OPTIONAL research-plane job producing a structured daily XAUUSD bias report
persisted to ``trading.ai_research_reports``.

Design contract (AGENTS.md boundaries):
- Research plane only. This module NEVER touches the Go realtime hot path and
  is never an execution authority. Its output is ONE input component
  (``institutional_research``) of the IGS composite with a small bounded weight.
- TradingAgents is an OPTIONAL dependency: import guarded, tests use a stub
  decision object so CI never requires LLM keys or network access.
- Honest provenance: the report records framework, model, decision text and a
  quality flag; LLM output is labeled GENERATED, never AUTHORITATIVE.
- Non-determinism: LLM sampling makes outputs vary run-to-run. Downstream
  consumers must treat bias as a slow-moving soft signal (TTL 3 days in IGS),
  never as a deterministic factor.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from datetime import UTC, date, datetime
from typing import Any

logger = logging.getLogger(__name__)

#: Guarded optional import — the research plane stays installable without the
#: (heavy) TradingAgents/LangGraph dependency tree.
try:  # pragma: no cover - exercised only when dependency installed
    from tradingagents.graph.trading_graph import TradingAgentsGraph  # type: ignore

    HAS_TRADINGAGENTS = True
except ImportError:  # pragma: no cover
    TradingAgentsGraph = None  # type: ignore[assignment,misc]
    HAS_TRADINGAGENTS = False


@dataclass(frozen=True)
class ResearchReport:
    """Structured institutional-research bias produced by the adapter."""

    run_date: str
    symbol: str
    framework: str = "tradingagents"
    framework_version: str = ""
    model: str = ""
    bias: str = "NEUTRAL"  # BULLISH | BEARISH | NEUTRAL
    confidence: float = 0.0  # self-reported, 0..1 — NEVER treated as calibrated probability
    summary: str = ""
    bull_thesis: str = ""
    bear_thesis: str = ""
    risks: str = ""
    key_drivers: tuple[str, ...] = ()
    quality: str = "GENERATED"

    def normalized(self) -> ResearchReport:
        """Coerce bias/confidence into bounded, DB-safe values."""
        bias = (self.bias or "NEUTRAL").strip().upper()
        if bias not in {"BULLISH", "BEARISH", "NEUTRAL"}:
            bias = "NEUTRAL"
        conf = min(max(float(self.confidence or 0.0), 0.0), 1.0)
        return ResearchReport(
            run_date=self.run_date,
            symbol=self.symbol,
            framework=self.framework,
            framework_version=self.framework_version,
            model=self.model,
            bias=bias,
            confidence=round(conf, 4),
            summary=self.summary or "",
            bull_thesis=self.bull_thesis or "",
            bear_thesis=self.bear_thesis or "",
            risks=self.risks or "",
            key_drivers=tuple(self.key_drivers or ()),
            quality=self.quality if self.quality in {"GENERATED", "REVIEWED", "REJECTED"} else "GENERATED",
        )

    def impact(self) -> float:
        """Map normalized bias to a bounded IGS impact score (-100..+100).

        The AI-research component carries a small IGS base weight (Tier B), so
        even a full-conviction report moves the composite modestly.
        """
        if self.normalized().quality == "REJECTED":
            return 0.0
        base = {"BULLISH": 60.0, "BEARISH": -60.0, "NEUTRAL": 0.0}[self.normalized().bias]
        return round(base * self.normalized().confidence, 2)

    def to_row(self) -> dict[str, Any]:
        n = self.normalized()
        return {
            "run_date": n.run_date,
            "symbol": n.symbol,
            "framework": n.framework,
            "framework_version": n.framework_version,
            "model": n.model,
            "bias": n.bias,
            "confidence": n.confidence,
            "summary": n.summary,
            "bull_thesis": n.bull_thesis,
            "bear_thesis": n.bear_thesis,
            "risks": n.risks,
            "key_drivers": list(n.key_drivers),
            "provenance": {
                "generated_at": datetime.now(UTC).isoformat(),
                "deterministic": False,
                "note": "LLM research opinion — research artifact, not an execution authority",
            },
            "quality": n.quality,
        }


@dataclass
class TradingAgentsAdapterConfig:
    """Runtime configuration for the TradingAgents bridge."""

    enabled: bool = False
    symbol: str = "XAUUSD"
    provider_symbol: str = "GLD"  # TradingAgents uses Yahoo-style tickers
    llm_provider: str = "openai"
    deep_think_llm: str = ""
    quick_think_llm: str = ""
    max_debate_rounds: int = 1
    temperature: float = 0.0
    timeout_seconds: int = 300


class TradingAgentsAdapter:
    """Thin adapter: TradingAgents decision -> ResearchReport.

    The framework's decision dict (bias-ish fields vary by version) is mapped
    defensively — unknown shapes degrade to a NEUTRAL report rather than
    guessing. Import failures are explicit: the adapter refuses to run without
    the optional dependency instead of fabricating a bias.
    """

    def __init__(self, config: TradingAgentsAdapterConfig) -> None:
        self._config = config
        self._graph = None

    def available(self) -> bool:
        return HAS_TRADINGAGENTS and self._config.enabled

    def _ensure_graph(self) -> bool:
        if self._graph is not None:
            return True
        if not HAS_TRADINGAGENTS:
            return False
        config: dict[str, Any] = {
            "llm_provider": self._config.llm_provider,
            "max_debate_rounds": self._config.max_debate_rounds,
            "temperature": self._config.temperature,
        }
        if self._config.deep_think_llm:
            config["deep_think_llm"] = self._config.deep_think_llm
        if self._config.quick_think_llm:
            config["quick_think_llm"] = self._config.quick_think_llm
        self._graph = TradingAgentsGraph(debug=False, config=config)
        return True

    def run(self, run_date: date | None = None) -> ResearchReport:
        """Execute one research run for the configured symbol.

        Returns a Report regardless of outcome; failures produce a NEUTRAL,
        REJECTED-quality report so downstream IGS math stays total.
        """
        run_on = (run_date or datetime.now(UTC).date()).isoformat()
        if not self.available():
            return ResearchReport(
                run_date=run_on,
                symbol=self._config.symbol,
                quality="REJECTED",
                summary="tradingagents not installed or adapter disabled — no LLM research performed",
            )
        try:
            if not self._ensure_graph():
                raise RuntimeError("TradingAgentsGraph unavailable")
            _, decision = self._graph.propagate(self._config.provider_symbol, run_on)  # type: ignore[union-attr]
            return self._map_decision(run_on, decision=decision)
        except Exception as exc:  # noqa: BLE001 — research tool must never crash the plane
            logger.warning("TradingAgents run failed: %s", exc)
            return ResearchReport(
                run_date=run_on,
                symbol=self._config.symbol,
                quality="REJECTED",
                summary=f"TradingAgents run failed: {exc}",
            )

    def _map_decision(self, run_date_str: str, decision: Any) -> ResearchReport:
        d = decision if isinstance(decision, dict) else {}
        raw_action = str(d.get("action", d.get("decision", ""))).upper()
        bias = "NEUTRAL"
        if "BUY" in raw_action or "LONG" in raw_action:
            bias = "BULLISH"
        elif "SELL" in raw_action or "SHORT" in raw_action:
            bias = "BEARISH"

        raw_conf = d.get("confidence", 0.0)
        try:
            confidence = float(raw_conf)
            if confidence > 1.0:  # frameworks sometimes report 0..100
                confidence /= 100.0
        except (TypeError, ValueError):
            confidence = 0.0

        return ResearchReport(
            run_date=run_date_str,
            symbol=self._config.symbol,
            framework="tradingagents",
            framework_version=self._framework_version(),
            model=str(self._config.deep_think_llm or self._config.llm_provider),
            bias=bias,
            confidence=confidence,
            summary=str(d.get("summary", d.get("reasoning", "")))[:2000],
            bull_thesis=str(d.get("bull_thesis", d.get("bullish_argument", "")))[:2000],
            bear_thesis=str(d.get("bear_thesis", d.get("bearish_argument", "")))[:2000],
            risks=str(d.get("risks", d.get("risk_assessment", "")))[:2000],
            key_drivers=tuple(str(k) for k in d.get("key_drivers", [])[:10]),
            quality="GENERATED",
        )

    @staticmethod
    def _framework_version() -> str:
        try:  # pragma: no cover - depends on optional install
            import importlib.metadata as im

            return im.version("tradingagents")
        except Exception:  # noqa: BLE001
            return ""


__all__ = [
    "HAS_TRADINGAGENTS",
    "ResearchReport",
    "TradingAgentsAdapter",
    "TradingAgentsAdapterConfig",
]