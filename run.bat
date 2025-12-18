@echo off
echo Downloading dependencies...
go mod tidy
if %errorlevel% neq 0 (
    echo Failed to download dependencies
    pause
    exit /b 1
)

echo.
echo Starting server on port 8080...
go run main.go
