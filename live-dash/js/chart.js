/* PAT Terminal — XAUUSD Price Charting & Ladder DOM Engine */

window.PAT = window.PAT || {};

class PriceChartEngine {
  constructor(canvasId, domTableId) {
    this.canvas = document.getElementById(canvasId);
    this.domTable = document.getElementById(domTableId);
    if (!this.canvas) return;

    this.ctx = this.canvas.getContext('2d');
    this.initCandles();
    this.initResizing();
    this.render();
  }

  initCandles() {
    this.candles = [];
    let base = 2510.00;
    const now = Date.now();
    for (let i = 0; i < 60; i++) {
      const open = base;
      const noise = (Math.random() - 0.48) * 1.2;
      const close = open + noise;
      const high = Math.max(open, close) + Math.random() * 0.8;
      const low = Math.min(open, close) - Math.random() * 0.8;
      this.candles.push({ open, high, low, close, time: now - (60 - i) * 5 * 60000 });
      base = close;
    }
  }

  initResizing() {
    const resize = () => {
      if (!this.canvas.parentElement) return;
      const rect = this.canvas.parentElement.getBoundingClientRect();
      this.canvas.width = rect.width * window.devicePixelRatio;
      this.canvas.height = rect.height * window.devicePixelRatio;
      this.ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
      this.render();
    };
    window.addEventListener('resize', resize);
    setTimeout(resize, 50);
  }

  getThemeColors() {
    const isLight = document.body.classList.contains('light');
    return {
      bg: isLight ? '#fffefa' : '#0f1520',
      grid: isLight ? 'rgba(23, 23, 20, 0.06)' : 'rgba(255, 255, 255, 0.05)',
      text: isLight ? '#68665e' : '#9ba3b4',
      bull: isLight ? '#15803d' : '#00c853',
      bear: isLight ? '#dc2626' : '#ff1744',
      ema9: '#00e5ff',
      ema21: '#ff9100',
      vwap: '#2962ff'
    };
  }

  render() {
    if (!this.canvas || !this.ctx) return;
    const width = this.canvas.width / window.devicePixelRatio;
    const height = this.canvas.height / window.devicePixelRatio;
    const colors = this.getThemeColors();

    this.ctx.clearRect(0, 0, width, height);

    if (this.candles.length === 0) return;

    let minPrice = Infinity;
    let maxPrice = -Infinity;
    this.candles.forEach(c => {
      if (c.low < minPrice) minPrice = c.low;
      if (c.high > maxPrice) maxPrice = c.high;
    });

    const padding = (maxPrice - minPrice) * 0.08 || 2.0;
    minPrice -= padding;
    maxPrice += padding;

    const chartPaddingRight = 50;
    const chartWidth = width - chartPaddingRight;
    const candleWidth = Math.max(2, (chartWidth / this.candles.length) - 2);

    const priceToY = (p) => height - ((p - minPrice) / (maxPrice - minPrice)) * (height - 20) - 10;

    // Grid
    this.ctx.strokeStyle = colors.grid;
    this.ctx.lineWidth = 1;
    this.ctx.fillStyle = colors.text;
    this.ctx.font = '9px JetBrains Mono, monospace';

    for (let i = 0; i <= 5; i++) {
      const p = minPrice + ((maxPrice - minPrice) / 5) * i;
      const y = priceToY(p);

      this.ctx.beginPath();
      this.ctx.moveTo(0, y);
      this.ctx.lineTo(chartWidth, y);
      this.ctx.stroke();

      this.ctx.fillText(p.toFixed(2), chartWidth + 6, y + 3);
    }

    // Candlesticks
    this.candles.forEach((c, idx) => {
      const x = (idx / this.candles.length) * chartWidth + candleWidth / 2;
      const openY = priceToY(c.open);
      const closeY = priceToY(c.close);
      const highY = priceToY(c.high);
      const lowY = priceToY(c.low);

      const isBull = c.close >= c.open;
      const color = isBull ? colors.bull : colors.bear;

      this.ctx.strokeStyle = color;
      this.ctx.lineWidth = 1;
      this.ctx.beginPath();
      this.ctx.moveTo(x, highY);
      this.ctx.lineTo(x, lowY);
      this.ctx.stroke();

      this.ctx.fillStyle = color;
      const bodyY = Math.min(openY, closeY);
      const bodyH = Math.max(1.5, Math.abs(closeY - openY));
      this.ctx.fillRect(x - candleWidth / 2, bodyY, candleWidth, bodyH);
    });

    this.renderDOM();
  }

  renderDOM() {
    if (!this.domTable) return;
    const p = PAT.state.price || 2515.00;

    this.domTable.innerHTML = `
      <div class="dom-row ask"><span>A4</span><span>${(p + 1.20).toFixed(2)}</span></div>
      <div class="dom-row ask"><span>A3</span><span>${(p + 0.90).toFixed(2)}</span></div>
      <div class="dom-row ask"><span>A2</span><span>${(p + 0.60).toFixed(2)}</span></div>
      <div class="dom-row ask"><span>A1</span><span>${(p + 0.30).toFixed(2)}</span></div>
      <div class="dom-spread">MID ${p.toFixed(2)}</div>
      <div class="dom-row bid"><span>B1</span><span>${(p - 0.30).toFixed(2)}</span></div>
      <div class="dom-row bid"><span>B2</span><span>${(p - 0.60).toFixed(2)}</span></div>
      <div class="dom-row bid"><span>B3</span><span>${(p - 0.90).toFixed(2)}</span></div>
      <div class="dom-row bid"><span>B4</span><span>${(p - 1.20).toFixed(2)}</span></div>
    `;
  }
}

window.PAT.PriceChartEngine = PriceChartEngine;
