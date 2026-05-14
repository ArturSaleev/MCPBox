@echo off
setlocal EnableExtensions

set "ROOT=%~dp0"
cd /d "%ROOT%"

set "GOCACHE=%ROOT%.gocache"
set "RELEASE_DIR=%ROOT%release"

if not exist "%RELEASE_DIR%" (
    mkdir "%RELEASE_DIR%"
)

echo [1/3] Building embedded UI...
if exist "%ROOT%html\node_modules\vite\bin\vite.js" (
    pushd "%ROOT%html"
    node .\node_modules\vite\bin\vite.js build
    if errorlevel 1 (
        popd
        echo UI build failed.
        exit /b 1
    )
    popd
) else (
    echo Vite was not found at html\node_modules\vite\bin\vite.js
    echo Run npm install inside html\ before using this script.
    exit /b 1
)

echo [2/3] Cleaning previous release binaries...
del /q "%RELEASE_DIR%\MCPBox-windows-amd64.exe" 2>nul
del /q "%RELEASE_DIR%\MCPBox-windows-arm64.exe" 2>nul
del /q "%RELEASE_DIR%\MCPBox-linux-amd64" 2>nul
del /q "%RELEASE_DIR%\MCPBox-linux-arm64" 2>nul
del /q "%RELEASE_DIR%\MCPBox-macos-amd64" 2>nul
del /q "%RELEASE_DIR%\MCPBox-macos-arm64" 2>nul

echo [3/3] Building Go binaries...
call :build windows amd64 "%RELEASE_DIR%\MCPBox-windows-amd64.exe" || exit /b 1
call :build windows arm64 "%RELEASE_DIR%\MCPBox-windows-arm64.exe" || exit /b 1
call :build linux amd64 "%RELEASE_DIR%\MCPBox-linux-amd64" || exit /b 1
call :build linux arm64 "%RELEASE_DIR%\MCPBox-linux-arm64" || exit /b 1
call :build darwin amd64 "%RELEASE_DIR%\MCPBox-macos-amd64" || exit /b 1
call :build darwin arm64 "%RELEASE_DIR%\MCPBox-macos-arm64" || exit /b 1

echo Release build completed successfully.
exit /b 0

:build
set "GOOS=%~1"
set "GOARCH=%~2"
set "OUTPUT=%~3"

echo   - %GOOS%/%GOARCH%
set "CGO_ENABLED=0"
go build -o "%OUTPUT%" .
if errorlevel 1 (
    echo Build failed for %GOOS%/%GOARCH%.
    exit /b 1
)

exit /b 0
