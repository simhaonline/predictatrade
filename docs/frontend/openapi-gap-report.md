# OpenAPI Gap Report

## Status

The NestJS control plane exposes an OpenAPI specification at `control/openapi.json`. The Go realtime engine does not expose Swagger/OpenAPI (it uses raw `http.ServeMux` handlers).

## Known Gaps

1. **Device-auth controller**: The `DevicesController` uses raw `@Body() body: any` instead of typed DTOs. The activate/refresh/heartbeat endpoints lack proper Swagger annotations.
2. **Licensing controller**: `registerDevice` and `addMtAccount` use `@Body() body: any` instead of typed DTOs.
3. **Billing controller**: The webhook endpoint uses `@Body() body: any, @Headers() headers: any`.
4. **Operations controller**: Strategy enable/disable and AI model endpoints use inline `@Body() body: { reason: string }` instead of DTOs.
5. **Go realtime engine**: No Swagger/OpenAPI annotations. All endpoints are documented via gateway/http.go source code.
6. **Response types**: Some endpoints return raw database query results without explicit response DTOs.

## Mitigation

The frontend uses explicit TypeScript interfaces for all API responses, defined inline in each page component. The `src/generated/schema.ts` file contains the OpenAPI-generated types from the NestJS spec. Frontend adapter types are used where the OpenAPI spec is incomplete.

**No backend modifications were made to resolve these gaps.** The frontend adapts to the existing backend contract.
