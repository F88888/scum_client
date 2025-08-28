@echo off
setlocal enabledelayedexpansion

echo ================================
echo    SCUM Client OCR Setup
echo ================================
echo.

echo NOTICE: This script is for manual setup only.
echo The main program (scum_client.exe) now automatically:
echo   - Downloads embedded Python to py_embed/
echo   - Installs PaddlePaddle and PaddleOCR dependencies
echo   - Sets up the OCR service
echo.

echo Recommended: Just run scum_client.exe directly!
echo.

set /p choice="Do you want to continue manual setup? (y/N): "
if /i "!choice!" neq "y" (
    echo.
    echo Run scum_client.exe to automatically set up everything!
    pause
    exit /b 0
)

echo.
echo ================================
echo Manual Setup Mode
echo ================================
echo.

REM Check if py_embed already exists
if exist "py_embed" (
    echo Found existing py_embed directory.
    set /p cleanup="Delete and recreate? (y/N): "
    if /i "!cleanup!" equ "y" (
        echo Removing existing py_embed...
        rmdir /s /q py_embed
        echo Removed.
    ) else (
        echo Keeping existing py_embed directory.
    )
)

REM Check for system Python as fallback
set PYTHON_CMD=
echo Checking for system Python installation...

REM Try python command
python --version >nul 2>&1
if !errorlevel! equ 0 (
    set PYTHON_CMD=python
    goto python_found
)

REM Try python3 command
python3 --version >nul 2>&1
if !errorlevel! equ 0 (
    set PYTHON_CMD=python3
    goto python_found
)

REM Try py launcher
py --version >nul 2>&1
if !errorlevel! equ 0 (
    set PYTHON_CMD=py
    goto python_found
)

REM No Python found
echo.
echo ERROR: No Python installation found!
echo.
echo RECOMMENDATIONS:
echo 1. Run scum_client.exe - it will download embedded Python automatically
echo 2. Or install Python manually:
echo    - Download: https://www.python.org/ftp/python/3.12.10/python-3.12.10-amd64.exe
echo    - Check "Add Python to PATH" during installation
echo    - Restart command prompt and run this script again
echo.
pause
exit /b 1

:python_found
for /f "tokens=2" %%v in ('!PYTHON_CMD! --version 2^>^&1') do set PYTHON_VERSION=%%v
echo SUCCESS: Python found: !PYTHON_CMD!
echo SUCCESS: Version: !PYTHON_VERSION!
echo.

REM Create py_embed directory structure to match embedded Python
if not exist "py_embed" (
    echo Creating py_embed directory structure...
    mkdir py_embed
    mkdir py_embed\Scripts
    mkdir py_embed\Lib
    echo Created py_embed directory structure.
)

REM Create a simple python launcher in Scripts
echo Creating Python launcher in py_embed\Scripts\...
echo @echo off > py_embed\Scripts\python.bat
echo !PYTHON_CMD! %%* >> py_embed\Scripts\python.bat

REM Install dependencies using system Python
echo.
echo Installing PaddlePaddle and dependencies...
echo This may take several minutes on first run...
echo.

echo Step 1: Upgrading pip...
!PYTHON_CMD! -m pip install --upgrade pip --quiet

echo Step 2: Installing PaddlePaddle...
!PYTHON_CMD! -m pip install paddlepaddle==3.0.0 -i https://pypi.tuna.tsinghua.edu.cn/simple --user
if !errorlevel! neq 0 (
    echo Trying without mirror...
    !PYTHON_CMD! -m pip install paddlepaddle==3.0.0 --user
    if !errorlevel! neq 0 (
        echo ERROR: PaddlePaddle installation failed!
        pause
        exit /b 1
    )
)

echo Step 3: Installing PaddleOCR and other dependencies...
!PYTHON_CMD! -m pip install paddleocr flask requests pillow -i https://pypi.tuna.tsinghua.edu.cn/simple --user
if !errorlevel! neq 0 (
    echo Trying without mirror...
    !PYTHON_CMD! -m pip install paddleocr flask requests pillow --user
    if !errorlevel! neq 0 (
        echo ERROR: Dependencies installation failed!
        pause
        exit /b 1
    )
)

echo SUCCESS: All dependencies installed
echo.

REM Check system default model cache
set "DEFAULT_CACHE=%USERPROFILE%\.paddlex\official_models"
set "CUSTOM_MODEL=paddle_models\en_PP-OCRv4_mobile_rec_infer"

echo Checking model status...
echo   System cache dir: %DEFAULT_CACHE%
echo   Custom model dir: %CUSTOM_MODEL%

REM Create model directory
if not exist "paddle_models" (
    echo Creating model directory...
    mkdir paddle_models
)

REM Check model availability (priority: custom model > system cache)
if exist "%CUSTOM_MODEL%" (
    echo SUCCESS: Custom model exists, will use: %CUSTOM_MODEL%
) else if exist "%DEFAULT_CACHE%" (
    echo SUCCESS: System cache model exists, will use default model
    echo INFO: No need to re-download. If you want custom model, run: download_model.py
) else (
    echo INFO: No model found, downloading custom English OCR model...
    cd paddle_models
    
    if exist "..\download_model.py" (
        !PYTHON_CMD! ..\download_model.py
        if !errorlevel! neq 0 (
            echo WARNING: Custom model download failed, will auto-download system default on first run
            cd ..
        ) else (
            echo SUCCESS: Custom model downloaded
            cd ..
        )
    ) else (
        echo INFO: download_model.py not found, will auto-download system default on first run
        echo   To manually download custom model, run: !PYTHON_CMD! download_model.py
        cd ..
    )
)

echo.
echo ================================
echo Manual Setup Completed!
echo ================================
echo Environment: py_embed/ (manual setup with system Python)
echo Model path: paddle_models/
echo.
echo IMPORTANT NOTES:
echo 1. This manual setup uses your system Python installation
echo 2. For best compatibility, use scum_client.exe (automatic embedded Python)
echo 3. If you have issues, delete py_embed/ and run scum_client.exe
echo.
echo You can now run scum_client.exe
pause