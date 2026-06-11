@echo off
setlocal

set APP_NAME=svelte-go-starter.exe
set SOURCE_EXE=bin\%APP_NAME%
set VERIFY_DIR=tmp\verify-empty
set VERIFY_PORT=3099

if not exist %SOURCE_EXE% (
  echo %SOURCE_EXE% is missing. Run build.bat first.
  exit /b 1
)

if exist %VERIFY_DIR% rmdir /s /q %VERIFY_DIR%
mkdir %VERIFY_DIR%
mkdir %VERIFY_DIR%\data
copy %SOURCE_EXE% %VERIFY_DIR%\%APP_NAME% >nul

powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$verify = Resolve-Path '%VERIFY_DIR%';" ^
  "$exe = Join-Path $verify '%APP_NAME%';" ^
  "$stdout = Join-Path $verify 'server.out.log';" ^
  "$stderr = Join-Path $verify 'server.err.log';" ^
  "$p = Start-Process -FilePath $exe -ArgumentList '--port','%VERIFY_PORT%','--db','data/app.db','--shared-db','data/shared.db' -WorkingDirectory $verify -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru -WindowStyle Hidden;" ^
  "try {" ^
  "  $ready = $false;" ^
  "  for ($i = 0; $i -lt 40; $i++) {" ^
  "    Start-Sleep -Milliseconds 500;" ^
  "    try { $r = Invoke-WebRequest -Uri 'http://127.0.0.1:%VERIFY_PORT%/' -UseBasicParsing -TimeoutSec 2; if ($r.StatusCode -eq 200) { $ready = $true; break } } catch {}" ^
  "  }" ^
  "  if (-not $ready) { throw 'server did not become ready' }" ^
  "  $root = Invoke-WebRequest -Uri 'http://127.0.0.1:%VERIFY_PORT%/' -UseBasicParsing;" ^
  "  $login = Invoke-WebRequest -Uri 'http://127.0.0.1:%VERIFY_PORT%/app/login' -UseBasicParsing;" ^
  "  $api = Invoke-WebRequest -Uri 'http://127.0.0.1:%VERIFY_PORT%/api/auth/status' -UseBasicParsing;" ^
  "  if ($root.Content.Contains('<div id=\"app\"></div>')) { throw 'root should serve server-rendered marketing HTML' }" ^
  "  if (-not $root.Content.Contains('<link rel=\"canonical\"')) { throw 'root did not serve marketing SEO tags' }" ^
  "  if (-not $login.Content.Contains('<div id=\"app\"></div>')) { throw 'SPA route did not serve embedded Svelte index' }" ^
  "  if ($api.Content -notmatch 'logged_in') { throw 'API status endpoint did not respond as expected' }" ^
  "} finally {" ^
  "  if ($p -and -not $p.HasExited) { Stop-Process -Id $p.Id -Force }" ^
  "}"

if errorlevel 1 exit /b 1

echo Verified %SOURCE_EXE% from %VERIFY_DIR% without frontend files on disk.
