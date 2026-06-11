@echo off
setlocal

set APP_NAME=svelte-go-starter.exe
set OUT_DIR=bin

where npm >nul 2>nul
if errorlevel 1 (
  echo npm was not found. Install Node.js before building.
  exit /b 1
)

where go >nul 2>nul
if errorlevel 1 (
  echo go was not found. Install Go before building.
  exit /b 1
)

echo Installing frontend dependencies...
pushd frontend
if not exist node_modules\.bin\vite.cmd (
  if exist node_modules (
    call npm install
  ) else if exist package-lock.json (
    call npm ci
  ) else (
    call npm install
  )
  if errorlevel 1 (
    popd
    exit /b 1
  )
)

echo Building Svelte static assets...
call npm run build
if errorlevel 1 (
  popd
  exit /b 1
)
popd

if not exist frontend\dist\index.html (
  echo frontend\dist\index.html is missing. Run npm run build in frontend before go build.
  exit /b 1
)

if not exist %OUT_DIR% mkdir %OUT_DIR%

echo Building Go executable with embedded frontend assets...
go build -a -o %OUT_DIR%\%APP_NAME% .
if errorlevel 1 exit /b 1

set ARTIFACT=%CD%\%OUT_DIR%\%APP_NAME%

echo.
echo Build completed successfully.
echo Output executable:
echo   %ARTIFACT%
echo.
echo This window will close automatically in 5 seconds.
powershell -NoProfile -Command "Start-Sleep -Seconds 5"
