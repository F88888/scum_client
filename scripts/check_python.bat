@echo off
echo Checking Python installation...
echo.

:: Check python command
echo Testing 'python' command:
python --version 2>nul
if %errorlevel% equ 0 (
    echo ✅ 'python' command works
    python --version
    goto :check_pip
) else (
    echo ❌ 'python' command not found
)

:: Check python3 command
echo.
echo Testing 'python3' command:
python3 --version 2>nul
if %errorlevel% equ 0 (
    echo ✅ 'python3' command works
    python3 --version
    goto :check_pip
) else (
    echo ❌ 'python3' command not found
)

:: Check py launcher
echo.
echo Testing 'py' launcher:
py --version 2>nul
if %errorlevel% equ 0 (
    echo ✅ 'py' launcher works
    py --version
    goto :check_pip
) else (
    echo ❌ 'py' launcher not found
)

echo.
echo ❌ No Python installation found!
echo.
echo Please:
echo 1. Download Python from: https://www.python.org/downloads/
echo 2. During installation, CHECK "Add Python to PATH"
echo 3. Restart your command prompt
echo 4. Run this script again
echo.
pause
exit /b 1

:check_pip
echo.
echo Testing pip installation:
python -m pip --version 2>nul
if %errorlevel% equ 0 (
    echo ✅ pip is working
    python -m pip --version
) else (
    echo ❌ pip not found
)

echo.
echo ================================
echo Python installation check completed!
echo ================================
pause
