#!/bin/bash
# Move to the directory where this script is located
cd "$(dirname "$0")"
PROJECT_ROOT="$(cd ../.. && pwd)"

# Check if the virtual environment exists
if [ ! -d "${PROJECT_ROOT}/.venv" ]; then
    echo "ERROR: Virtual environment (.venv) not found in the project folder."
    echo "Please read the 'User_Guide.txt' for setup instructions."
    echo "Press any key to exit..."
    read -n 1
    exit 1
fi

echo "Starting browser MIDI Synthesiser..."
echo "------------------------------------------------"
echo "WEB UI IS ACTIVE at http://127.0.0.1:5002"
echo "------------------------------------------------"
echo "TO STOP: Press Ctrl+C in this window."
echo "------------------------------------------------"

# Open the browser in the background after a 2-second delay
(sleep 2 && open "http://127.0.0.1:5002") &

# Run the server in the FOREGROUND
# This ensures that Ctrl+C sends the signal directly to Python
"${PROJECT_ROOT}/.venv/bin/python3" app.py
