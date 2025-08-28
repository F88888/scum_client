# Python 3.12.11 自动安装脚本
# 需要以管理员权限运行

Write-Host "==================================" -ForegroundColor Green
Write-Host "Python 3.12.11 自动安装脚本" -ForegroundColor Green
Write-Host "==================================" -ForegroundColor Green
Write-Host ""

# 检查是否以管理员权限运行
$currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
$isAdmin = $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host "⚠️  此脚本需要以管理员权限运行" -ForegroundColor Yellow
    Write-Host "请右键点击 PowerShell 并选择 '以管理员身份运行'" -ForegroundColor Yellow
    Write-Host ""
    Read-Host "按任意键退出"
    exit 1
}

# 检测系统架构
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "" }
$pythonUrl = if ($arch -eq "amd64") {
    "https://www.python.org/ftp/python/3.12.10/python-3.12.10-amd64.exe"
} else {
    "https://www.python.org/ftp/python/3.12.10/python-3.12.10.exe"
}

$installerName = if ($arch -eq "amd64") {
    "python-3.12.10-amd64.exe"
} else {
    "python-3.12.10.exe"
}

Write-Host "🔍 系统架构: $(if ($arch -eq 'amd64') { '64位' } else { '32位' })" -ForegroundColor Cyan
Write-Host "📥 下载链接: $pythonUrl" -ForegroundColor Cyan
Write-Host ""

# 检查是否已安装 Python
try {
    $pythonVersion = python --version 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Python 已安装: $pythonVersion" -ForegroundColor Green
        $response = Read-Host "是否继续安装最新版本？(y/N)"
        if ($response -ne 'y' -and $response -ne 'Y') {
            Write-Host "取消安装" -ForegroundColor Yellow
            exit 0
        }
    }
} catch {
    Write-Host "🔍 未检测到 Python，继续安装..." -ForegroundColor Cyan
}

# 下载 Python 安装包
Write-Host "📥 正在下载 Python 3.12.11..." -ForegroundColor Cyan
try {
    Invoke-WebRequest -Uri $pythonUrl -OutFile $installerName -UseBasicParsing
    Write-Host "✅ 下载完成: $installerName" -ForegroundColor Green
} catch {
    Write-Host "❌ 下载失败: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "请手动下载: $pythonUrl" -ForegroundColor Yellow
    Read-Host "按任意键退出"
    exit 1
}

# 安装 Python
Write-Host ""
Write-Host "🛠️  正在安装 Python 3.12.11..." -ForegroundColor Cyan
Write-Host "⚠️  安装过程中会自动添加到 PATH" -ForegroundColor Yellow
Write-Host ""

try {
    # 静默安装，自动添加到 PATH
    $installArgs = @(
        "/quiet",
        "InstallAllUsers=1",
        "PrependPath=1",
        "Include_test=0",
        "Include_tcltk=1",
        "Include_pip=1",
        "Include_dev=1"
    )
    
    Start-Process -FilePath $installerName -ArgumentList $installArgs -Wait -NoNewWindow
    Write-Host "✅ Python 安装完成！" -ForegroundColor Green
    
    # 清理安装包
    Remove-Item $installerName -Force
    Write-Host "🧹 已清理安装包" -ForegroundColor Cyan
    
} catch {
    Write-Host "❌ 安装失败: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "请手动运行安装包: $installerName" -ForegroundColor Yellow
    Read-Host "按任意键退出"
    exit 1
}

# 刷新环境变量
Write-Host ""
Write-Host "🔄 刷新环境变量..." -ForegroundColor Cyan
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")

# 验证安装
Write-Host ""
Write-Host "✅ 验证 Python 安装..." -ForegroundColor Cyan
Start-Sleep -Seconds 2

try {
    $pythonVersion = python --version 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Python 验证成功: $pythonVersion" -ForegroundColor Green
        
        $pipVersion = python -m pip --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✅ pip 验证成功: $pipVersion" -ForegroundColor Green
        }
    } else {
        throw "Python 命令未找到"
    }
} catch {
    Write-Host "❌ Python 验证失败" -ForegroundColor Red
    Write-Host "请重启命令提示符或重启计算机后再试" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "==================================" -ForegroundColor Green
Write-Host "安装完成！" -ForegroundColor Green
Write-Host "==================================" -ForegroundColor Green
Write-Host "下一步："
Write-Host "1. 重启命令提示符" -ForegroundColor Yellow
Write-Host "2. 运行 scum_client.exe" -ForegroundColor Yellow
Write-Host "3. 程序将自动设置 OCR 环境" -ForegroundColor Yellow
Write-Host ""
Read-Host "按任意键退出"
