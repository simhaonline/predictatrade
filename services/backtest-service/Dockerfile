# Predict-A-Trade Online Backtesting Service
FROM python:3.12-slim

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PAT_RESEARCH_SRC=/app/research/src

# System deps for psycopg2.
RUN apt-get update && apt-get install -y --no-install-recommends gcc libpq-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY services/backtest-service/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

# Mount the research engine sources (read-only at runtime; build step copies them in).
COPY research/src /app/research/src
COPY services/backtest-service /app/backtest-service

EXPOSE 8088
WORKDIR /app/backtest-service
CMD ["sh", "-c", "uvicorn main:app --host 0.0.0.0 --port 8088"]
