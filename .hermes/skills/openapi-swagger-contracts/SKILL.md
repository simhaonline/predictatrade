---
name: openapi-swagger-contracts
description: "Generate and validate OpenAPI/Swagger documentation."
---

# openapi-swagger-contracts

Use when generating or validating API docs.

## Generate
NestJS: cd control && npm run swagger:generate
Go: cd realtime && swag init -g cmd/realtime-engine/main.go

## Contract Gates
1. Every endpoint has request/response schema
2. DTOs have @ApiProperty with examples
3. Auth documented (JWT, API key)
4. Error responses: 400-500 with examples
5. Pagination params documented
6. Deprecated endpoints marked
7. Version in path: /api/v1/
8. No undocumented internal endpoints in external docs

## Validate
npx swagger-cli validate openapi.json
