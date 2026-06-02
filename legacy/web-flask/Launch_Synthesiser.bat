@echo off
setlocal

rem Move to the directory where this script is located
pushd "%~dp0"
for %%I in ("%CD%\..\..") do set "PROJECT_ROOT=%%~fI"
set "VUE_DIR=%PROJECT_ROOT%\frontend"
set "FLASK_BACKEND_URL=http://127.0.0.1:8000"
set "VUE_FRONTEND_URL=http://127.0.0.1:5173"

rem Check if the virtual environment exists
if not exist "%PROJECT_ROOT%\.venv\" (
    echo ERROR: Virtual environment ^(.venv^) not found in the project folder.
    echo Please read the "User_Guide.txt" for setup instructions.
    echo Press any key to exit...
    pause >nul
    popd
    exit /b 1
)

if not exist "%PROJECT_ROOT%\.venv\Scripts\python.exe" (
    echo ERROR: "%PROJECT_ROOT%\.venv\Scripts\python.exe" was not found.
    echo Please recreate the virtual environment and reinstall the dependencies.
    echo Press any key to exit...
    pause >nul
    popd
    exit /b 1
)

where npm >nul 2>nul
if errorlevel 1 (
    echo ERROR: npm was not found. Install Node.js and npm before launching the Vue frontend.
    echo Press any key to exit...
    pause >nul
    popd
    exit /b 1
)

if not exist "%VUE_DIR%\node_modules\" (
    echo ERROR: Vue dependencies are not installed.
    echo Run this once from the repository root:
    echo   cd frontend ^&^& npm ci
    echo Press any key to exit...
    pause >nul
    popd
    exit /b 1
)

echo Starting OctaBit...
echo ------------------------------------------------
echo Flask API backend: %FLASK_BACKEND_URL%
echo Vue frontend:      %VUE_FRONTEND_URL%
echo ------------------------------------------------
echo TO STOP: Press Ctrl+C in this window and close the Flask API window.
echo ------------------------------------------------

set "PORT=8000"
set "WEB_FLASK_OPEN_BROWSER=0"

pushd "%PROJECT_ROOT%"
start "OctaBit Flask API" "%PROJECT_ROOT%\.venv\Scripts\python.exe" legacy\web-flask\app.py
popd

rem Open the browser in the background after a short delay
start "" powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Sleep -Seconds 3; Start-Process '%VUE_FRONTEND_URL%'"

rem Run the Vue development server in the foreground
pushd "%VUE_DIR%"
npm run dev
set "EXIT_CODE=%ERRORLEVEL%"
popd

popd
exit /b %EXIT_CODE%
