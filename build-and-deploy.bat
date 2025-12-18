@echo off
echo Installing Lambda dependencies...
go get github.com/aws/aws-lambda-go/lambda
go get github.com/aws/aws-lambda-go/events
go get github.com/awslabs/aws-lambda-go-api-proxy/httpadapter

echo Building Go Lambda function...

REM Create build directory
if not exist lambda-build mkdir lambda-build

REM Build Go binary for Lambda (Linux AMD64)
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -tags lambda -o lambda-build\bootstrap lambda_main.go

if %errorlevel% neq 0 (
    echo Build failed!
    exit /b %errorlevel%
)

echo Go binary built successfully

REM Navigate to CDK directory
cd cdk

echo Installing CDK dependencies...
pip install -r requirements.txt

echo Deploying CDK stack...
cdk deploy --require-approval never

echo Deployment complete!
cd ..
