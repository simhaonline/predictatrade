import * as promClient from 'prom-client';

// Enable default Node.js metrics (GC, memory, event loop)
promClient.collectDefaultMetrics({ register: promClient.register });

// HTTP request metrics
export const httpRequestDuration = new promClient.Histogram({
  name: 'http_request_duration_seconds',
  help: 'Duration of HTTP requests in seconds',
  labelNames: ['method', 'route', 'status'],
  buckets: [0.005, 0.01, 0.05, 0.1, 0.5, 1, 5],
});

export const httpRequestsTotal = new promClient.Counter({
  name: 'http_requests_total',
  help: 'Total HTTP requests',
  labelNames: ['method', 'route', 'status'],
});

export const httpResponseErrors = new promClient.Counter({
  name: 'http_response_errors_total',
  help: 'Total HTTP error responses (5xx)',
  labelNames: ['method', 'route'],
});

// DB metrics
export const dbQueryDuration = new promClient.Histogram({
  name: 'db_query_duration_seconds',
  help: 'Database query duration in seconds',
  labelNames: ['operation'],
  buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5],
});

export function metricsMiddleware(req: any, res: any, next: any) {
  const start = Date.now();
  res.on('finish', () => {
    const duration = (Date.now() - start) / 1000;
    const route = (req.route && req.route.path) || (req.url && req.url.split('?')[0]) || 'unknown';
    const method = req.method || 'UNKNOWN';
    const status = String(res.statusCode);
    httpRequestDuration.labels(method, route, status).observe(duration);
    httpRequestsTotal.labels(method, route, status).inc();
    if (res.statusCode >= 500) {
      httpResponseErrors.labels(method, route).inc();
    }
  });
  next();
}

export async function getMetrics(): Promise<string> {
  return promClient.register.metrics();
}
