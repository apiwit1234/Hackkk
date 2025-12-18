# Bedrock Question Search API

A Go-based REST API that uses AWS Bedrock for intelligent question search using embeddings and knowledge base retrieval.

## Features

- Question search using AWS Bedrock embeddings (Titan)
- Knowledge base integration for semantic search
- RESTful API with health check endpoint
- Configurable retry logic and throttling
- Comprehensive error handling
- AWS Lambda deployment ready

## Local Development

### Prerequisites

- Go 1.23+
- AWS credentials configured
- Access to AWS Bedrock and a Knowledge Base

### Setup

1. Copy environment variables:
   ```bash
   copy .env.example .env
   ```

2. Update `.env` with your AWS configuration:
   ```
   AWS_REGION=us-east-1
   BEDROCK_EMBEDDING_MODEL=amazon.titan-embed-text-v2
   BEDROCK_KB_ID=YOUR_KNOWLEDGE_BASE_ID
   ```

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Run locally:
   ```bash
   go run main.go
   ```

5. Test:
   ```bash
   curl http://localhost:8080/api/teletubpax/healthcheck
   ```

### Running Tests

```bash
go test ./...
```

## AWS Deployment

Deploy to AWS Lambda + API Gateway for a serverless, scalable API.

### Quick Deploy

1. **Install prerequisites:**
   - AWS CLI configured (`aws configure`)
   - AWS CDK installed (`npm install -g aws-cdk`)
   - Python 3.x

2. **Update configuration:**
   Edit `cdk/cdk.json` with your Knowledge Base ID

3. **Deploy:**
   ```bash
   deploy.bat
   ```

4. **Test your deployed API:**
   ```bash
   curl https://YOUR_API_URL/api/teletubpax/healthcheck
   ```

See [QUICKSTART.md](QUICKSTART.md) for detailed deployment instructions.

## API Endpoints

### Health Check
```
GET /api/teletubpax/healthcheck
```

Response:
```json
{
  "message": "I'm OK",
  "status": 200
}
```

### Question Search
```
POST /api/teletubpax/question-search
Content-Type: application/json

{
  "question": "Your question here"
}
```

## Project Structure

```
.
├── aws/                    # AWS Bedrock client implementations
├── config/                 # Configuration management
├── errors/                 # Custom error types
├── routing/                # HTTP routing and handlers
├── services/               # Business logic
├── utils/                  # Utility functions (retry, etc.)
├── cdk/                    # AWS CDK infrastructure code
├── main.go                 # Local development entry point
├── lambda_main.go          # Lambda entry point
└── deploy.bat              # Deployment script
```

## Architecture

### Local Development
- Standard Go HTTP server on port 8080
- Direct AWS SDK calls to Bedrock

### AWS Deployment
- **Lambda Function**: Runs Go binary with custom runtime
- **API Gateway**: HTTP API for routing
- **IAM Role**: Bedrock permissions
- **CloudWatch**: Logging and monitoring

## Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `AWS_REGION` | AWS region | us-east-1 |
| `BEDROCK_EMBEDDING_MODEL` | Bedrock embedding model | amazon.titan-embed-text-v2 |
| `BEDROCK_KB_ID` | Knowledge Base ID | Required |
| `MAX_QUESTION_LENGTH` | Max question length | 1000 |
| `RETRY_ATTEMPTS` | Number of retries | 3 |
| `LOG_LEVEL` | Logging level (DEBUG, INFO, WARN, ERROR) | ERROR |

## Cost Estimation

AWS Lambda deployment costs (approximate):

- **Lambda**: $0.20 per 1M requests + compute time
- **API Gateway**: $1.00 per 1M requests
- **Bedrock**: Pay per token/request
- **CloudWatch Logs**: $0.50/GB

First 1M Lambda requests per month are free.

## Logging

The API includes CloudWatch logging with configurable log levels:

### Log Levels
- **DEBUG**: Detailed debugging information
- **INFO**: General information (requests, responses, configuration)
- **WARN**: Warnings (validation errors, throttling)
- **ERROR**: Errors only (default for production)

### Configuration
Set the `LOG_LEVEL` environment variable:
```bash
LOG_LEVEL=ERROR  # Default - only errors
LOG_LEVEL=WARN   # Warnings and errors
LOG_LEVEL=INFO   # Info, warnings, and errors
LOG_LEVEL=DEBUG  # All logs
```

### Log Groups
- **Lambda**: `/aws/lambda/{function-name}` (uses standard output, CloudWatch handles automatically)
- **Container/Local**: `/teletubpax-api/local` (sends to CloudWatch Logs)

### Structured Logging
Error logs include structured fields for easy filtering:
```json
{
  "level": "ERROR",
  "message": "Question search failed after retries",
  "error": "ThrottlingException",
  "duration_ms": 5000,
  "retry_count": 3
}
```

## Monitoring

After deployment, monitor your API:

- **CloudWatch Logs**: `/aws/lambda/BedrockApiStack-BedrockApiFunction-*`
- **CloudWatch Insights**: Query structured logs for analysis
- **API Gateway Metrics**: Request count, latency, errors
- **Lambda Metrics**: Invocations, duration, errors

### Example CloudWatch Insights Queries

```
# Find all errors
fields @timestamp, message, error
| filter level = "ERROR"
| sort @timestamp desc

# Track request duration
fields @timestamp, duration_ms
| filter message = "Question search completed successfully"
| stats avg(duration_ms), max(duration_ms), min(duration_ms)

# Monitor throttling
fields @timestamp, message
| filter level = "WARN" and message like /throttled/
```

## Security

- Lambda runs with minimal IAM permissions
- Only Bedrock access granted
- CORS enabled (configure as needed)
- Consider adding API authentication for production

## Troubleshooting

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed troubleshooting guide.

## License

MIT
