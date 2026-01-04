@echo off
REM Development build script: templ generate + tailwind build + go build
REM This ensures that templ changes trigger Tailwind CSS rebuild

setlocal enabledelayedexpansion

echo [1/3] Generating templ...
templ generate ./...
if !errorlevel! neq 0 (
    echo ERROR: templ generate failed
    exit /b 1
)

echo [2/3] Building Tailwind CSS...
call npm run build:css
if !errorlevel! neq 0 (
    echo ERROR: CSS build failed
    exit /b 1
)

echo [3/3] Building Go binary...
go build -o ./tmp/pehlione-web.exe ./cmd/web
if !errorlevel! neq 0 (
    echo ERROR: Go build failed
    exit /b 1
)

echo SUCCESS: Build complete
exit /b 0
