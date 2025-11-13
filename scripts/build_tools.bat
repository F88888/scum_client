@echo off
echo === SCUM 快速命令工具构建脚本 ===

REM 生成UUID作为文件名前缀
for /f "tokens=*" %%i in ('powershell -command "[System.Guid]::NewGuid().ToString()"') do set UUID=%%i

REM 构建RCON快速命令工具
echo 构建 RCON 快速命令工具...
cd cmd\rcon_quick
go build -o ..\..\bin\scum_rcon_quick_%UUID%.exe .
cd ..\..

REM 构建优化快速命令工具
echo 构建优化快速命令工具...
cd cmd\quick_command
go build -o ..\..\bin\scum_quick_command_%UUID%.exe .
cd ..\..

REM 创建输出目录
if not exist "bin" mkdir bin

echo.
echo === 构建完成 ===
echo.
echo 生成的执行文件:
echo   bin\scum_rcon_quick_%UUID%.exe     - RCON快速命令工具
echo   bin\scum_quick_command_%UUID%.exe  - 优化快速命令工具
echo.
echo 使用方法:
echo   RCON工具:
echo     scum_rcon_quick_%UUID%.exe -addr 127.0.0.1:7777 -pass 密码 -i
echo.
echo   优化工具:
echo     scum_quick_command_%UUID%.exe -mode interactive
echo     scum_quick_command_%UUID%.exe -mode server -port 8080
echo     scum_quick_command_%UUID%.exe -mode command -cmd "players"
echo.
pause 