@echo off
setlocal

rem Move to the directory where this script is located
pushd "%~dp0"
for %%I in ("%CD%\..\..") do set "PROJECT_ROOT=%%~fI"

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

echo Starting legacy browser MIDI Synthesiser...
echo ------------------------------------------------
echo LEGACY WEB UI IS ACTIVE at http://127.0.0.1:5002
echo ------------------------------------------------
echo TO STOP: Press Ctrl+C in this window.
echo ------------------------------------------------

rem Open the browser in the background after a 2-second delay
start "" powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Sleep -Seconds 2; Start-Process 'http://127.0.0.1:5002'"

rem Run the server in the foreground
"%PROJECT_ROOT%\.venv\Scripts\python.exe" app.py
set "EXIT_CODE=%ERRORLEVEL%"

popd
exit /b %EXIT_CODE%
