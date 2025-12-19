@echo off
REM Test script for document details API endpoint

echo Testing Document Details API...
echo.

REM Set your API Gateway URL here
set API_URL=https://bzeewve97h.execute-api.us-east-1.amazonaws.com/api/teletubpax/last-update-document

echo Calling: %API_URL%
echo.

curl -X GET "%API_URL%" ^
  -H "Content-Type: application/json" ^
  -H "Accept: application/json" ^
  -w "\n\nHTTP Status: %%{http_code}\n" ^
  -s

echo.
echo.
echo Test completed!
pause
