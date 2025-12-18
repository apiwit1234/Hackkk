# Health Check API Flow

## Endpoint: GET /api/teletubpax/healthcheck

### Flow Diagram
```
Client Request
     |
     v
[GET /api/teletubpax/healthcheck]
     |
     v
[HealthCheckHandler]
     |
     v
[Return JSON Response]
     |
     v
Client receives: {"message": "I'm OK", "status": 200}
```

### Request Flow
1. Client sends GET request to `/api/teletubpax/healthcheck`
2. Router matches the path and calls `HealthCheckHandler`
3. Handler creates Response struct with message "I'm OK"
4. Handler sets Content-Type header to "application/json"
5. Handler writes HTTP 200 status code
6. Handler encodes and sends JSON response

### Error Handling
- **404**: If path doesn't match, `NotFoundHandler` is called
- **400**: Use `BadRequestHandler` for invalid request data
- **500**: Use `InternalServerErrorHandler` for server errors

### Response Time
Expected: < 10ms (simple health check)
