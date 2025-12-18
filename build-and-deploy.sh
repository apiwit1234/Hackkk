#!/bin/bash
set -e

echo "Installing Lambda dependencies..."
go get github.com/aws/aws-lambda-go/lambda
go get github.com/aws/aws-lambda-go/events
go get github.com/awslabs/aws-lambda-go-api-proxy/httpadapter

echo "Building Go Lambda function..."

# Create build directory
mkdir -p lambda-build

# Build Go binary for Lambda (Linux AMD64)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda -o lambda-build/bootstrap lambda_main.go

echo "Go binary built successfully"

# Navigate to CDK directory
cd cdk

echo "Installing CDK dependencies..."
pip install -r requirements.txt

echo "Deploying CDK stack..."
cdk deploy --require-approval never

echo "Deployment complete!"
