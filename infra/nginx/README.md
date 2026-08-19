# Predict-A-Trade Nginx Configuration

## Service Ports (Internal Only — 13080-13090 reserved for Predict-A-Trade)

| Service | Port | Host |
|---------|------|------|
| NestJS Control Plane | 13080 | 127.0.0.1 |
| Go Real-Time Engine (HTTP + WS) | 13081 | 127.0.0.1 |
| Next.js Frontend | 13082 | 127.0.0.1 |
| Status Page | 13083 | 127.0.0.1 |
| 13084-13090 | Reserved | 127.0.0.1 |

All services bind to 127.0.0.1. Nginx is the only public ingress.
