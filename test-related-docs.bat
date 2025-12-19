@echo off
REM Test script for question search with related documents

echo Building and starting server...
go build -o teletubpax-api.exe
if errorlevel 1 (
    echo Build failed!
    pause
    exit /b 1
)

echo Starting server in background...
start /B teletubpax-api.exe

echo Waiting for server to start...
timeout /t 3 /nobreak >nul

echo.
echo Testing Question Search API with enableRelateDocument=true
echo.

curl -X POST "http://localhost:8080/api/teletubpax/question-search?enableRelateDocument=true" ^
  -H "Content-Type: application/json" ^
  -d "{\"question\":\"ประกัน\"}" ^
  -w "\n\nHTTP Status: %%{http_code}\n"

echo.
echo.
echo Test completed!
echo.
echo Press any key to stop the server...
pause >nul

taskkill /F /IM teletubpax-api.exe 2>nul
echo Server stopped
