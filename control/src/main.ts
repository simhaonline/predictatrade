import { NestFactory } from '@nestjs/core';
import { ValidationPipe, Logger } from '@nestjs/common';
import { SwaggerModule, DocumentBuilder } from '@nestjs/swagger';
import { AppModule } from './app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule, { cors: true });

  // Global validation pipe (SOW Section 80 — schema validation)
  app.useGlobalPipes(new ValidationPipe({
    whitelist: true,
    forbidNonWhitelisted: true,
    transform: true,
    transformOptions: { enableImplicitConversion: true },
  }));

  // Global prefix
  app.setGlobalPrefix('api/v1');

  // OpenAPI documentation (SOW Section 81)
  const config = new DocumentBuilder()
    .setTitle('Predict-A-Trade Control Plane API')
    .setDescription('XAUUSD Intelligence Platform — SaaS Control Plane v1.0.0')
    .setVersion('1.0.0')
    .addBearerAuth()
    .build();
  const document = SwaggerModule.createDocument(app, config);
  SwaggerModule.setup('api/docs', app, document);

  const port = process.env.CONTROL_PORT || 3000;
  await app.listen(port);
  Logger.log(`Control plane running on port ${port}`, 'Bootstrap');
}

bootstrap();
