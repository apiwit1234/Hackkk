# CDK Infrastructure

This directory contains AWS CDK code to deploy the Bedrock API to AWS Lambda.

## Structure

```
cdk/
├── app.py              # CDK app entry point
├── cdk.json            # CDK configuration
├── requirements.txt    # Python dependencies
└── stacks/
    └── api_stack.py    # Lambda + API Gateway stack
```

## Configuration

Edit `cdk.json` to customize your deployment:

```json
{
  "context": {
    "aws_region": "us-east-1",
    "embedding_model": "amazon.titan-embed-text-v2",
    "knowledge_base_id": "YOUR_KB_ID_HERE",
    "max_question_length": "1000",
    "retry_attempts": "3"
  }
}
```

## CDK Commands

```bash
# Install dependencies
pip install -r requirements.txt

# Synthesize CloudFormation template
cdk synth

# View differences
cdk diff

# Deploy
cdk deploy

# Destroy all resources
cdk destroy
```

## Stack Resources

The `BedrockApiStack` creates:

1. **Lambda Function**
   - Runtime: Custom (Go binary)
   - Memory: 512 MB
   - Timeout: 30 seconds
   - Architecture: x86_64

2. **IAM Role**
   - Bedrock InvokeModel permissions
   - Bedrock Retrieve permissions (Knowledge Base)
   - CloudWatch Logs permissions

3. **API Gateway HTTP API**
   - Catch-all route: `/{proxy+}`
   - CORS enabled
   - Lambda integration

4. **CloudWatch Log Group**
   - 7-day retention
   - Automatic log streaming

## Outputs

After deployment, the stack outputs:

- `ApiUrl`: Your API Gateway endpoint URL
- `LambdaFunctionName`: Lambda function name for debugging

## Customization

### Change Lambda Memory/Timeout

Edit `stacks/api_stack.py`:

```python
api_lambda = lambda_.Function(
    # ...
    timeout=Duration.seconds(60),  # Increase timeout
    memory_size=1024,               # Increase memory
)
```

### Add Custom Domain

```python
from aws_cdk import aws_certificatemanager as acm
from aws_cdk import aws_route53 as route53

# Add certificate and domain configuration
```

### Add API Authentication

```python
from aws_cdk import aws_apigatewayv2_authorizers as authorizers

# Add Cognito or Lambda authorizer
```

### Enable X-Ray Tracing

```python
api_lambda = lambda_.Function(
    # ...
    tracing=lambda_.Tracing.ACTIVE,
)
```

## Multi-Environment Deployment

Deploy to different environments:

```bash
# Development
cdk deploy -c environment=dev -c knowledge_base_id=DEV_KB_ID

# Production
cdk deploy -c environment=prod -c knowledge_base_id=PROD_KB_ID
```

## Cost Optimization

- Lambda memory: Start with 512 MB, adjust based on CloudWatch metrics
- Log retention: 7 days default, reduce for cost savings
- API Gateway: HTTP API is cheaper than REST API
- Consider provisioned concurrency for production (eliminates cold starts)

## Security Best Practices

1. **Least Privilege IAM**: Only Bedrock permissions granted
2. **VPC**: Consider deploying Lambda in VPC for private resources
3. **Secrets**: Use AWS Secrets Manager for sensitive config
4. **API Keys**: Add API Gateway API keys for production
5. **WAF**: Add AWS WAF for DDoS protection

## Monitoring

CloudWatch metrics to monitor:

- Lambda: Invocations, Duration, Errors, Throttles
- API Gateway: Count, IntegrationLatency, 4XXError, 5XXError
- Custom: Add CloudWatch alarms for critical metrics

## Troubleshooting

### Deployment fails with "No export named..."
- Run `cdk bootstrap` first

### Lambda timeout errors
- Increase timeout in `api_stack.py`
- Check Bedrock API latency

### Permission denied errors
- Verify Knowledge Base ID is correct
- Check IAM role has correct permissions
- Ensure region matches your resources

### Cold start issues
- Consider provisioned concurrency
- Optimize Lambda package size
- Use ARM64 architecture (faster cold starts)
