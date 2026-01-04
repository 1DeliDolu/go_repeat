@echo off
REM Change to the script directory
cd /d "%~dp0"

REM Run Air from the correct directory
air -c .air.toml %*
