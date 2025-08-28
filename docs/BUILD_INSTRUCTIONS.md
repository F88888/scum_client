# SCUM Client 构建说明

## 问题描述
在使用 `go build` 生成可执行文件后，运行时出现以下错误：
```
开始设置 OCR 环境...
OCR 服务启动失败: 自动设置 OCR 环境失败: 安装脚本不存在: ocr_setup.bat
```

**原因**: `go build` 只编译 Go 源代码，不会自动复制项目中的其他文件（如 `ocr_setup.bat` 和 `ocr_server.py`）。

## 解决方案
使用 Go 1.16+ 的 `embed` 包将 OCR 相关文件嵌入到二进制文件中。

### 修改内容

1. **main.go 修改**:
   - 添加 `embed` 导入
   - 使用 `//go:embed` 指令嵌入文件
   - 添加 `extractEmbeddedFiles()` 函数
   - 在程序启动时自动提取文件

2. **新增文件**:
   - `build_with_ocr.bat`: 完整的构建和测试脚本
   - 更新了 `README_OCR.md` 文档

### 使用方法

#### 方法一：使用构建脚本（推荐）
```bash
build_with_ocr.bat
```

#### 方法二：手动构建
```bash
go build -o scum_client.exe
```

### 工作流程
1. 程序启动时检查当前目录
2. 如果缺少 `ocr_setup.bat` 或 `ocr_server.py`，自动提取
3. 正常进行 OCR 环境检查和服务启动

### 优势
- ✅ **单文件分发**: 生成的 exe 包含所有必需文件
- ✅ **自动修复**: 删除 OCR 文件后重新运行会自动恢复
- ✅ **版本一致**: 确保使用正确版本的脚本
- ✅ **部署简单**: 只需复制一个 exe 文件

### 测试方法
1. 构建程序：`build_with_ocr.bat`
2. 将生成的 `scum_client.exe` 复制到新目录
3. 运行程序，确认自动提取 OCR 文件
4. 验证 OCR 功能正常工作

## 注意事项
- 确保 `assets/ocr_setup.bat` 和 `assets/ocr_server.py` 文件存在于源代码目录
- 需要 Go 1.16 或更高版本支持 `embed` 包
- 首次运行仍需要 Python 环境来安装 OCR 依赖
