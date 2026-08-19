import { NestFactory } from '@nestjs/core';
import { ValidationPipe, Logger } from '@nestjs/common';
import { SwaggerModule, DocumentBuilder } from '@nestjs/swagger';
import { AppModule } from './app.module';
import helmet from 'helmet';
import { metricsMiddleware, getMetrics } from './common/metrics';
import cookieParser = require('cookie-parser');

async function bootstrap() {
  const app = await NestFactory.create(AppModule);

  // Cookie parsing for HttpOnly refresh-token cookie
  app.use(cookieParser());

  // CORS: only allow canonical production origins (SOW §80, security)
  const corsOrigins = process.env.CORS_ORIGINS
    ? process.env.CORS_ORIGINS.split(',').map(s => s.trim())
    : ['https://platform.predictatrade.com', 'https://live.predictatrade.com'];

  app.enableCors({
    origin: corsOrigins,
    credentials: true,
    methods: ['GET', 'POST', 'PATCH', 'PUT', 'DELETE', 'OPTIONS'],
    allowedHeaders: ['Content-Type', 'Authorization', 'X-Correlation-Id'],
    exposedHeaders: ['X-Correlation-Id'],
  });

  // Security headers (SOW §82)
  app.use(metricsMiddleware);
  app.use(helmet({
    contentSecurityPolicy: false, // CSP handled by Nginx for API
    crossOriginEmbedderPolicy: false,
  }));

  // Input validation
  app.useGlobalPipes(new ValidationPipe({
    whitelist: true,
    forbidNonWhitelisted: true,
    transform: true,
    transformOptions: { enableImplicitConversion: true },
  }));

  app.setGlobalPrefix('api/v1');

  // OpenAPI docs (disabled in production behind auth — Nginx blocks /api/docs externally)
  if (process.env.NODE_ENV !== 'production') {
    const config = new DocumentBuilder()
      .setTitle('Predict-A-Trade Control Plane API')
      .setDescription('XAUUSD Intelligence Platform — SaaS Control Plane v1.0.0')
      .setVersion('1.0.0')
      .addBearerAuth()
      .build();
    const document = SwaggerModule.createDocument(app, config);
    SwaggerModule.setup('api/docs', app, document);
  }

  // Bind to 127.0.0.1 only — Nginx is the public ingress
  const host = process.env.CONTROL_HOST || '127.0.0.1';
  // Prometheus metrics endpoint — no auth (scraped by Prometheus)
  const httpAdapter = app.getHttpAdapter();
  const instance = httpAdapter.getInstance();
  instance.get('/metrics', (req: any, res: any) => {
    res.set('Content-Type', 'text/plain; version=0.0.4; charset=utf-8');
    getMetrics().then(m => res.send(m));
  });

  const port = process.env.CONTROL_PORT || 13080;
  await app.listen(port, host);
  Logger.log(`Control plane running on ${host}:${port}`, 'Bootstrap');
}

bootstrap();
