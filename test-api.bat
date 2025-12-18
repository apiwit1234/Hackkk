@echo off
setlocal enabledelayedexpansion

echo ========================================
echo   API Testing Script
echo ========================================
echo.

if "%1"=="" (
    echo Usage: test-api.bat YOUR_API_URL
    echo.
    echo Example:
    echo   test-api.bat https://abc123.execute-api.us-east-1.amazonaws.com
    echo.
    echo Get your API URL from CDK deployment output
    exit /b 1
)

set API_URL=%1

echo Testing API at: %API_URL%
echo.

echo ----------------------------------------
echo Test 1: Health Check
echo ----------------------------------------
curl -s -w "\nStatus: %%{http_code}\n" %API_URL%/api/teletubpax/healthcheck
echo.

echo ----------------------------------------
echo Test 2: Question Search (Valid)
echo ----------------------------------------
curl -s -w "\nStatus: %%{http_code}\n" ^
  -X POST %API_URL%/api/teletubpax/question-search ^
  -H "Content-Type: application/json" ^
  -d "{\"question\": \"What is AWS Bedrock?\"}"
echo.

echo ----------------------------------------
echo Test 3: Question Search (Empty)
echo ----------------------------------------
curl -s -w "\nStatus: %%{http_code}\n" ^
  -X POST %API_URL%/api/teletubpax/question-search ^
  -H "Content-Type: application/json" ^
  -d "{\"question\": \"\"}"
echo.

echo ----------------------------------------
echo Test 4: Invalid Endpoint
echo ----------------------------------------
curl -s -w "\nStatus: %%{http_code}\n" %API_URL%/api/invalid
echo.

echo ========================================
echo   Testing Complete
echo ========================================
