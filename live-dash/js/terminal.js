/* PAT Terminal — XAUUSD Terminal UI Controller (Matching https://live.predictatrade.com/) */

window.PAT = window.PAT || {};

class TerminalUI {
  constructor() {
    this.priceChart = null;
    this.neuralEngine = null;

    this.initTheme();
    this.initEngines();
    this.renderAllPanels();
    this.bindEvents();
    this.startLiveSimulation();
  }

  // Theme Management
  initTheme() {
    const theme = localStorage.getItem('pat_theme') || 'dark';
    if (theme === 'light') {
      document.body.classList.add('light');
    } else {
      document.body.classList.remove('light');
    }
  }

  toggleTheme() {
    const isLight = document.body.classList.contains('light');
    if (isLight) {
      document.body.classList.remove('light');
      localStorage.setItem('pat_theme', 'dark');
    } else {
      document.body.classList.add('light');
      localStorage.setItem('pat_theme', 'light');
    }

    if (this.priceChart) this.priceChart.render();
  }

  initEngines() {
    if (PAT.PriceChartEngine) {
      this.priceChart = new PAT.PriceChartEngine('priceChartCanvas', 'domTableBody');
    }
    if (PAT.NeuralNetworkEngine) {
      this.neuralEngine = new PAT.NeuralNetworkEngine('neuralCanvas');
    }
  }

  renderAllPanels() {
    this.renderHeaderAndNews();
    this.renderAccountPanel();
    this.renderSignalFeed();
    this.renderNeuralPanel();
    this.renderMtfAndGates();
    this.renderAllIndicators();
    this.renderStrategyEngines();
    this.renderAgentsStatus();
  }

  renderHeaderAndNews() {
    const clock = document.getElementById('clockUTC');
    if (clock) {
      const updateClock = () => {
        const now = new Date();
        clock.innerText = now.toLocaleTimeString('en-GB', { timeZone: 'UTC', hour12: false }) + ' UTC';
      };
      updateClock();
      setInterval(updateClock, 1000);
    }

    const newsTrack = document.getElementById('newsTickerTrack');
    if (newsTrack) {
      newsTrack.innerHTML = PAT.newsEvents.map(n => `<span style="margin-right:24px;">${n}</span>`).join('');
    }
  }

  renderAccountPanel() {
    const priceEl = document.getElementById('bigPrice');
    const bidEl = document.getElementById('pBid');
    const askEl = document.getElementById('pAsk');
    const spreadEl = document.getElementById('pSpread');
    const volEl = document.getElementById('pVol');
    const regimeEl = document.getElementById('pRegime');
    const sessionEl = document.getElementById('pSession');
    const mtfEl = document.getElementById('pMtfBias');
    const confEl = document.getElementById('pConfidence');

    if (priceEl) priceEl.innerText = PAT.state.price.toFixed(2);
    if (bidEl) bidEl.innerText = PAT.state.bid.toFixed(2);
    if (askEl) askEl.innerText = PAT.state.ask.toFixed(2);
    if (spreadEl) spreadEl.innerText = PAT.state.spread.toFixed(2);
    if (volEl) volEl.innerText = PAT.state.volume;
    if (regimeEl) regimeEl.innerText = PAT.state.regime;
    if (sessionEl) sessionEl.innerText = PAT.state.session;
    if (mtfEl) mtfEl.innerText = `+${PAT.state.mtfScore} LONG`;
    if (confEl) confEl.innerText = `${PAT.state.confidence}%`;
  }

  renderSignalFeed() {
    const container = document.getElementById('signalScrollFeed');
    if (!container) return;

    container.innerHTML = PAT.signals.map(s => `
      <div class="signal-feed-card">
        <div class="sig-head-row">
          <span class="sig-name">${s.id}</span>
          <span class="sig-dir ${s.state}">${s.state}</span>
        </div>
        <div class="sig-details-row">
          <span>Score: <strong style="color:var(--text-bright);">${s.score}</strong></span>
          <span>E: <strong>${s.entry}</strong> | SL: <strong>${s.sl}</strong></span>
          <span style="color:var(--amber); font-weight:800;">${s.target}</span>
        </div>
      </div>
    `).join('');
  }

  renderNeuralPanel() {
    const grid = document.getElementById('neuralIndGrid');
    if (!grid) return;

    const items = [
      { k: "RSI (14)", v: "64.3" },
      { k: "ATR (14)", v: "1.8" },
      { k: "EMA 9/21", v: "2,514 / 2,512" },
      { k: "SMA 200", v: "2,495.06" },
      { k: "ADX (14)", v: "48.6" },
      { k: "BB Upper", v: "2,522.22" },
      { k: "MACD", v: "+2.2" },
      { k: "Stoch", v: "74.3" },
      { k: "CCI", v: "51.3" },
      { k: "VWAP", v: "2,508.67" }
    ];

    grid.innerHTML = items.map(i => `
      <div class="neural-ind-cell">
        <span style="color:var(--text-dim);">${i.k}</span>
        <span style="font-weight:700; color:var(--text-bright);">${i.v}</span>
      </div>
    `).join('');
  }

  renderMtfAndGates() {
    const mtfGrid = document.getElementById('mtfGridCells');
    if (mtfGrid) {
      mtfGrid.innerHTML = Object.entries(PAT.mtfStates).map(([tf, val]) => `
        <div class="mtf-cell">
          <div class="mtf-lbl">${tf}</div>
          <div class="mtf-val ${val}">${val}</div>
        </div>
      `).join('');
    }

    const gatesList = document.getElementById('hardGatesList');
    if (gatesList) {
      gatesList.innerHTML = PAT.hardGates.map(g => `
        <div class="gate-item-row">
          <span class="gate-id">${g.id}</span>
          <span class="gate-status ${g.status}">${g.status}</span>
        </div>
      `).join('');
    }
  }

  renderAllIndicators() {
    const grid = document.getElementById('allIndicatorsGrid');
    if (!grid) return;

    grid.innerHTML = PAT.allIndicators.map(i => `
      <div class="ind-cell-full">
        <span class="ind-name">${i.name}</span>
        <span class="ind-val">${i.val}</span>
      </div>
    `).join('');
  }

  renderStrategyEngines() {
    const cards = document.getElementById('strategyEngineCards');
    if (!cards) return;

    cards.innerHTML = PAT.strategyEngines.map(e => `
      <div class="eng-card">
        <div class="eng-title">${e.name}</div>
        <div class="eng-status">${e.status}</div>
        <div style="font-size:8px; color:var(--text-dim);">${e.decision}</div>
        <div class="eng-score">${e.score}</div>
        <div style="font-size:7px; color:var(--blue);">${e.tf}</div>
        <div style="font-size:7px; color:var(--text-dim);">${e.stats}</div>
      </div>
    `).join('');
  }

  renderAgentsStatus() {
    const table = document.getElementById('agentsTableBody');
    if (!table) return;

    table.innerHTML = PAT.agentsStatus.map(a => `
      <tr>
        <td style="color:var(--text-dim);">${a.k}</td>
        <td>${a.v}</td>
      </tr>
    `).join('');
  }

  bindEvents() {
    const themeBtn = document.getElementById('themeToggle');
    if (themeBtn) {
      themeBtn.onclick = () => this.toggleTheme();
    }
  }

  startLiveSimulation() {
    setInterval(() => {
      const delta = (Math.random() - 0.48) * 0.20;
      PAT.state.bid = parseFloat((PAT.state.bid + delta).toFixed(2));
      PAT.state.ask = parseFloat((PAT.state.bid + 0.30).toFixed(2));
      PAT.state.price = PAT.state.bid;

      this.renderAccountPanel();
      if (this.priceChart) this.priceChart.render();
    }, 2000);
  }
}

window.addEventListener('DOMContentLoaded', () => {
  window.patUI = new TerminalUI();
});
