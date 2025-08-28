# SCUM Client

SCUM 游戏客户端自动化工具，集成 PaddleOCR 图像识别功能。

## 快速开始

### 构建程序
```bash
# 推荐使用构建脚本
scripts\build_with_ocr.bat

# 或手动构建
go build -o scum_client.exe
```

### 运行程序
```bash
scum_client.exe
```

程序会自动：
- 提取必需的 OCR 文件
- 设置 OCR 环境
- 启动游戏监控

## 项目结构

```
scum_client/
├── assets/              # 资源文件
│   ├── ocr_setup.bat   # OCR 环境设置脚本
│   └── ocr_server.py   # OCR HTTP 服务
├── cmd/                # 命令行工具
├── docs/               # 文档
├── examples/           # 示例代码
├── scripts/            # 构建脚本
├── server/             # 服务器相关
├── util/               # 工具函数
├── config.yaml         # 配置文件
├── main.go            # 主程序
└── README.md          # 项目说明
```

## 文档

详细文档请查看 `docs/` 目录：

- [OCR 集成说明](docs/README_OCR.md)
- [构建说明](docs/BUILD_INSTRUCTIONS.md)
- [增强输入说明](docs/README_ENHANCED_INPUT.md)
- [性能优化说明](docs/README_OPTIMIZATION.md)
- [速度优化说明](docs/README_SPEED_OPTIMIZATION.md)

## 系统要求

- Windows 10/11
- Python 3.8+ (如未安装请参考 [Python 安装指南](docs/PYTHON_INSTALLATION_GUIDE.md))
- Go 1.16+ (用于编译)

## 首次运行

如果遇到 "Python not found" 错误：

1. **自动安装** (推荐)：
   ```bash
   # 以管理员权限运行 PowerShell
   scripts\install_python.ps1
   ```

2. **手动安装**：
   - 下载：[Python 3.12.11 (64位)](https://www.python.org/ftp/python/3.12.11/python-3.12.11-amd64.exe)
   - ⚠️ 安装时勾选 "Add Python to PATH"
   - 重启命令提示符

3. **验证安装**：
   ```bash
   scripts\check_python.bat
   ```

## 许可证

本项目遵循项目许可证条款。
