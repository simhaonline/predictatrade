/* PAT Terminal — Neural Shell Network Canvas Engine */

window.PAT = window.PAT || {};

class NeuralNetworkEngine {
  constructor(canvasId) {
    this.canvas = document.getElementById(canvasId);
    if (!this.canvas) return;
    this.ctx = this.canvas.getContext('2d');

    this.nodes = [];
    this.initNodes();
    this.animate();
  }

  initNodes() {
    this.width = this.canvas.width = this.canvas.parentElement.clientWidth || 230;
    this.height = this.canvas.height = this.canvas.parentElement.clientHeight || 65;

    this.nodes = [];
    for (let i = 0; i < 22; i++) {
      this.nodes.push({
        x: Math.random() * this.width,
        y: Math.random() * this.height,
        vx: (Math.random() - 0.5) * 0.8,
        vy: (Math.random() - 0.5) * 0.8,
        radius: Math.random() * 1.8 + 1
      });
    }
  }

  animate() {
    if (!this.ctx) return;
    this.ctx.clearRect(0, 0, this.width, this.height);

    const isLight = document.body.classList.contains('light');
    const lineCol = isLight ? 'rgba(36, 99, 255, 0.25)' : 'rgba(41, 98, 255, 0.35)';
    const nodeCol = isLight ? '#2463ff' : '#00c853';

    // Move & draw nodes
    for (let i = 0; i < this.nodes.length; i++) {
      const n = this.nodes[i];
      n.x += n.vx;
      n.y += n.vy;

      if (n.x < 0 || n.x > this.width) n.vx *= -1;
      if (n.y < 0 || n.y > this.height) n.vy *= -1;

      this.ctx.beginPath();
      this.ctx.arc(n.x, n.y, n.radius, 0, Math.PI * 2);
      this.ctx.fillStyle = nodeCol;
      this.ctx.fill();

      // Connect close nodes
      for (let j = i + 1; j < this.nodes.length; j++) {
        const n2 = this.nodes[j];
        const dist = Math.hypot(n2.x - n.x, n2.y - n.y);
        if (dist < 45) {
          this.ctx.beginPath();
          this.ctx.moveTo(n.x, n.y);
          this.ctx.lineTo(n2.x, n2.y);
          this.ctx.strokeStyle = lineCol;
          this.ctx.lineWidth = 0.6;
          this.ctx.stroke();
        }
      }
    }

    requestAnimationFrame(() => this.animate());
  }
}

window.PAT.NeuralNetworkEngine = NeuralNetworkEngine;
