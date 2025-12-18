@echo off
echo Setting up AWS Bedrock configuration...

set AWS_REGION=us-east-1
set BEDROCK_EMBEDDING_MODEL=amazon.titan-embed-text-v2
set BEDROCK_KB_ID=R1DHVCY9K7

REM Set your AWS credentials here
set AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY_ID_HERE
set AWS_SECRET_ACCESS_KEY=YOUR_SECRET_ACCESS_KEY_HERE
REM set AWS_SESSION_TOKEN=YOUR_SESSION_TOKEN_HERE (if using temporary credentials)

echo Configuration set:
echo   AWS_REGION=%AWS_REGION%
echo   BEDROCK_EMBEDDING_MODEL=%BEDROCK_EMBEDDING_MODEL%
echo   BEDROCK_KB_ID=%BEDROCK_KB_ID%
echo   AWS_ACCESS_KEY_ID=%AWS_ACCESS_KEY_ID%
echo.
echo Starting server...
go run main.go
