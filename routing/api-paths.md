# API Paths Collection

## Health Check
- **Path**: `/api/teletubpax/healthcheck`
- **Method**: `GET`
- **Description**: Check if the API service is running
- **Response**: JSON with status message

### Success Response (200)
```json
{
  "message": "I'm OK",
  "status": 200
}
```

### Error Responses

#### 400 - Bad Request
```json
{
  "error": "Bad request message",
  "status": 400
}
```

#### 404 - Not Found
```json
{
  "error": "Resource not found",
  "status": 404
}
```

#### 500 - Internal Server Error
```json
{
  "error": "Internal server error message",
  "status": 500
}
```
