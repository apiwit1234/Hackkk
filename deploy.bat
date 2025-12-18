@echo off
echo ========================================
echo   Bedrock API Deployment Script
echo ========================================
echo.

REM Check if this is first deployment
if not exist "cdk\cdk.out" (
    echo First time deployment detected...
    echo.
    echo IMPORTANT: Update your Knowledge Base ID in cdk\cdk.json
    echo Press any key when ready, or Ctrl+C to cancel...
    pause >nul
    
    echo.
    echo Bootstrapping CDK (one-time setup)...
    cd cdk
    cdk bootstrap
    if %errorlevel% neq 0 (
        echo CDK bootstrap failed! Check your AWS credentials.
        cd ..
        exit /b %errorlevel%
    )
    cd ..
)

echo.
echo Step 1: Installing Go dependencies...
go get github.com/aws/aws-lambda-go/lambda
go get github.com/aws/aws-lambda-go/events
go get github.com/awslabs/aws-lambda-go-api-proxy/httpadapter
go mod tidy

echo.
echo Step 2: Building Lambda binary...
if not exist lambda-build mkdir lambda-build

set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -tags lambda -o lambda-build\bootstrap lambda_main.go

if %errorlevel% neq 0 (
    echo Build failed!
    exit /b %errorlevel%
)

echo Build successful!

echo.
echo Step 3: Installing CDK dependencies...
cd cdk
pip install -q -r requirements.txt

echo.
echo Step 4: Deploying to AWS...
cdk deploy --require-approval never

if %errorlevel% neq 0 (
    echo Deployment failed!
    cd ..
    exit /b %errorlevel%
)

cd ..

echo.
echo ========================================
echo   Deployment Complete!
echo ========================================
echo.
echo Your API is now live. Check the output above for the API URL.
echo.
echo Test with:
echo   curl YOUR_API_URL/api/teletubpax/healthcheck
echo.
