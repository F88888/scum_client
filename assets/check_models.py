#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
PaddleOCR 模型检查和修复工具
检查现有模型缓存并提供修复建议
"""

import os
import sys
import shutil

def check_models():
    """检查 PaddleOCR 模型状态"""
    print("=== PaddleOCR 模型状态检查 ===")
    
    # 检查自定义模型目录
    custom_models = [
        "paddle_models/en_PP-OCRv4_mobile_rec_infer",
        "paddle_models/PP-OCRv5_mobile_det",
    ]
    
    custom_found = False
    for model_path in custom_models:
        if os.path.exists(model_path):
            print(f"✓ 找到自定义模型: {model_path}")
            files = os.listdir(model_path)
            print(f"  模型文件: {files}")
            custom_found = True
        else:
            print(f"✗ 未找到自定义模型: {model_path}")
    
    # 检查系统缓存目录
    home_dir = os.path.expanduser("~")
    system_caches = [
        os.path.join(home_dir, ".paddleocr"),
        os.path.join(home_dir, ".paddlex", "official_models"),
        os.path.join(home_dir, ".cache", "paddleocr"),
    ]
    
    print("\n--- 系统缓存目录检查 ---")
    cache_found = False
    for cache_dir in system_caches:
        if os.path.exists(cache_dir):
            print(f"✓ 找到缓存目录: {cache_dir}")
            try:
                for root, dirs, files in os.walk(cache_dir):
                    if files:
                        print(f"  缓存内容: {root}")
                        for file in files[:5]:  # 只显示前5个文件
                            print(f"    - {file}")
                        if len(files) > 5:
                            print(f"    ... 还有 {len(files) - 5} 个文件")
                        cache_found = True
                        break
            except PermissionError:
                print(f"  权限不足，无法访问缓存内容")
                cache_found = True
        else:
            print(f"✗ 未找到缓存目录: {cache_dir}")
    
    # 给出建议
    print("\n--- 建议 ---")
    if custom_found:
        print("✓ 建议: 使用已有的自定义模型，避免重复下载")
    elif cache_found:
        print("✓ 建议: 系统缓存中有模型，但可能版本不匹配，建议重新配置")
    else:
        print("! 建议: 运行 download_model.py 下载自定义模型")
    
    return custom_found, cache_found

def clean_cache(confirm=False):
    """清理 PaddleOCR 缓存"""
    if not confirm:
        response = input("\n是否清理所有 PaddleOCR 缓存? (y/N): ")
        if response.lower() != 'y':
            print("取消清理操作")
            return False
    
    print("\n=== 清理 PaddleOCR 缓存 ===")
    
    home_dir = os.path.expanduser("~")
    cache_dirs = [
        os.path.join(home_dir, ".paddleocr"),
        os.path.join(home_dir, ".paddlex"),
        os.path.join(home_dir, ".cache", "paddleocr"),
    ]
    
    cleaned = False
    for cache_dir in cache_dirs:
        if os.path.exists(cache_dir):
            try:
                shutil.rmtree(cache_dir)
                print(f"✓ 已清理缓存目录: {cache_dir}")
                cleaned = True
            except Exception as e:
                print(f"✗ 清理失败: {cache_dir} - {e}")
        else:
            print(f"- 缓存目录不存在: {cache_dir}")
    
    if cleaned:
        print("✓ 缓存清理完成，下次运行将重新下载模型")
    else:
        print("- 没有找到需要清理的缓存")
    
    return cleaned

def main():
    """主函数"""
    if len(sys.argv) > 1 and sys.argv[1] == "--clean":
        clean_cache(confirm=True)
    else:
        custom_found, cache_found = check_models()
        
        if not custom_found and cache_found:
            print("\n检测到系统缓存但无自定义模型")
            response = input("是否清理缓存并重新下载? (y/N): ")
            if response.lower() == 'y':
                if clean_cache(confirm=True):
                    print("\n运行以下命令下载自定义模型:")
                    print("python download_model.py")

if __name__ == '__main__':
    main()
