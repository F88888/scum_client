@echo off
echo ================================
echo   SCUM Client OCR Simple Setup
echo ================================
echo.

echo NOTICE: The main program now handles OCR setup automatically!
echo.
echo RECOMMENDED: Just run scum_client.exe
echo - It will automatically download embedded Python
echo - Install all required dependencies  
echo - Set up the OCR service
echo.

set /p choice="Continue with manual setup anyway? (y/N): "
if /i "%choice%" neq "y" (
    echo.
    echo Run scum_client.exe for automatic setup!
    pause
    exit /b 0
)

echo.
echo Manual Setup Mode - Using System Python
echo =======================================

echo Step 1: Checking Python...
python --version >nul 2>&1
if errorlevel 1 goto no_python

echo Python found! Version:
python --version
goto setup_deps

:no_python
echo.
echo ERROR: Python not found!
echo.
echo Please install Python first:
echo 1. Download: https://www.python.org/ftp/python/3.12.10/python-3.12.10-amd64.exe
echo 2. During installation, check "Add Python to PATH"
echo 3. Restart command prompt and run this script again
echo.
echo OR simply run scum_client.exe for automatic setup!
pause
exit /b 1

:setup_deps
echo.
echo Step 2: Creating directory structure...
if not exist "py_embed" mkdir py_embed
if not exist "py_embed\Scripts" mkdir py_embed\Scripts

echo.
echo Step 3: Installing packages (this may take a few minutes)...
echo Installing PaddlePaddle...
python -m pip install --upgrade pip --quiet
python -m pip install paddlepaddle==3.0.0 --user --quiet
if errorlevel 1 (
    echo ERROR: PaddlePaddle installation failed!
    echo Try running scum_client.exe instead for automatic setup
    pause
    exit /b 1
)

echo Installing PaddleOCR and other dependencies...
python -m pip install paddleocr flask requests pillow --user --quiet
if errorlevel 1 (
    echo ERROR: Package installation failed!
    echo Try running scum_client.exe instead for automatic setup
    pause
    exit /b 1
)

echo.
echo Step 4: Setting up model directory...
if not exist "paddle_models" mkdir paddle_models

REM Check system default model cache
set "DEFAULT_CACHE=%USERPROFILE%\.paddlex\official_models"
set "CUSTOM_MODEL=paddle_models\en_PP-OCRv4_mobile_rec_infer"

echo.
echo Step 5: Checking model status...
echo   System cache: %DEFAULT_CACHE%
echo   Custom model: %CUSTOM_MODEL%

if exist "%CUSTOM_MODEL%" (
    echo SUCCESS: Custom model exists
) else if exist "%DEFAULT_CACHE%" (
    echo SUCCESS: System cache model exists, no re-download needed
) else (
    echo Downloading custom model...
    if exist "download_model.py" (
        python download_model.py
        if errorlevel 1 (
            echo WARNING: Custom model download failed - will auto-download system default on first use
        )
    ) else (
        echo INFO: Will auto-download system default model on first OCR use
    )
)

echo.
echo ================================
echo Simple Setup Completed!
================================
echo.
echo IMPORTANT: This manual setup uses system Python
echo For best results, use scum_client.exe (automatic embedded Python)
echo.
echo You can now run the main program: scum_client.exe
pause