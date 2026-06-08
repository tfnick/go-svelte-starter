@echo off
setlocal

set PORT=%1
if "%PORT%"=="" set PORT=3000

set FRONTEND_PORT=5173
set FRONTEND_URL=http://127.0.0.1:%FRONTEND_PORT%

where npm >nul 2>nul
if errorlevel 1 (
  echo npm was not found. Install Node.js before running dev.bat.
  pause
  exit /b 1
)

echo Preparing Svelte frontend...
pushd frontend
if not exist node_modules\.bin\vite.cmd (
  echo Installing frontend dependencies...
  if exist node_modules (
    call npm install
  ) else if exist package-lock.json (
    call npm ci
  ) else (
    call npm install
  )
  if errorlevel 1 (
    popd
    pause
    exit /b 1
  )
)

call npm run build
if errorlevel 1 (
  popd
  pause
  exit /b 1
)
popd

echo Starting Go backend on http://127.0.0.1:%PORT% ...
start "Go API %PORT%" cmd /k go run . --dev --port %PORT% --frontend-dev-url %FRONTEND_URL%

echo Starting Svelte dev server on %FRONTEND_URL% ...
pushd frontend
start "Svelte Frontend %FRONTEND_PORT%" cmd /k npm run dev -- --port %FRONTEND_PORT%
popd

echo.
echo Development stack is starting.
echo Open %FRONTEND_URL% in your browser. Svelte edits hot reload through Vite.
echo.
pause
