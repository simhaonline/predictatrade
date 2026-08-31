FROM python:3.12-slim

# Production watchdog for the realtime engine. Zero runtime deps (stdlib only).
WORKDIR /app

COPY monitor.py /app/monitor.py

# No exposed ports; it only polls the engine and pushes to ntfy.
CMD ["python3", "-u", "/app/monitor.py"]
