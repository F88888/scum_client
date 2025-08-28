#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
PaddleOCR HTTP 服务
提供图片文字识别的 HTTP API 接口
"""

import os
import sys
import json
import base64
import logging
from io import BytesIO
from flask import Flask, request, jsonify
from PIL import Image
import traceback

# 设置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

app = Flask(__name__)

# 全局变量
ocr = None
ocr_initialized = False

def initialize_paddleocr():
    """初始化 PaddleOCR"""
    global ocr, ocr_initialized
    
    if ocr_initialized:
        return True
        
    try:
        logger.info("正在初始化 PaddleOCR...")
        
        # 导入 PaddleOCR
        from paddleocr import PaddleOCR
        
        # 设置模型缓存目录，避免重复下载
        import os
        home_dir = os.path.expanduser("~")
        paddle_cache_dir = os.path.join(home_dir, ".paddleocr")
        
        # 确保缓存目录存在
        os.makedirs(paddle_cache_dir, exist_ok=True)
        logger.info(f"PaddleOCR 模型缓存目录: {paddle_cache_dir}")
        
        # 设置环境变量指定模型缓存路径
        os.environ['PADDLEOCR_MODEL_PATH'] = paddle_cache_dir
        
        # 检查自定义模型路径
        rec_model_dir = "paddle_models/PP-OCRv5_mobile_det"
        
        if os.path.exists(rec_model_dir):
            logger.info(f"使用自定义英文识别模型: {rec_model_dir}")
            # 使用自定义英文识别模型
            ocr = PaddleOCR(
                use_doc_orientation_classify=False,
                use_doc_unwarping=False,
                use_textline_orientation=False,
                rec_model_dir=rec_model_dir,
                show_log=False  # 减少日志输出
            )
        else:
            logger.info("使用默认英文模型（模型将缓存到用户目录，避免重复下载）")
            # 使用默认模型，指定语言和模型缓存
            ocr = PaddleOCR(
                use_doc_orientation_classify=False,
                use_doc_unwarping=False,
                use_textline_orientation=False,
                lang='en',  # 明确指定英文，避免下载中文模型
                show_log=False,  # 减少日志输出
                use_gpu=False,  # 明确使用CPU，避免GPU相关问题
                model_download_dir=paddle_cache_dir  # 指定模型下载目录
            )
        
        ocr_initialized = True
        logger.info("PaddleOCR 初始化完成")
        return True
        
    except Exception as e:
        logger.error(f"PaddleOCR 初始化失败: {e}")
        logger.error(traceback.format_exc())
        return False

def process_ocr_result(result, target_text=None):
    """处理 OCR 识别结果"""
    if not result:
        return {"code": 200, "data": "", "message": "没有识别到文字"}
    
    # 提取所有文字
    texts = []
    for line in result:
        if len(line) >= 2:
            text = line[1][0].strip()
            if text:
                texts.append(text)
    
    # 合并所有文字
    full_text = " ".join(texts).strip()
    
    # 如果指定了目标文字，检查是否匹配
    if target_text:
        # 忽略大小写和空格进行比较
        clean_full_text = full_text.replace(" ", "").lower()
        clean_target_text = target_text.replace(" ", "").lower()
        
        if clean_target_text in clean_full_text:
            return {"code": 100, "data": target_text, "message": "识别成功"}
        else:
            return {"code": 200, "data": full_text, "message": "未找到目标文字"}
    
    return {"code": 100, "data": full_text, "message": "识别成功"}

@app.route('/api/ocr', methods=['POST'])
def ocr_recognition():
    """OCR 识别接口"""
    global ocr
    
    try:
        # 检查 OCR 是否已初始化
        if not ocr_initialized:
            if not initialize_paddleocr():
                return jsonify({"code": 500, "data": "", "message": "OCR 服务初始化失败"})
        
        # 获取请求数据
        data = request.get_json()
        if not data or 'Base64' not in data:
            return jsonify({"code": 400, "data": "", "message": "缺少 Base64 图片数据"})
        
        # 解码 Base64 图片
        base64_str = data['Base64']
        try:
            img_data = base64.b64decode(base64_str)
            img = Image.open(BytesIO(img_data))
            
            # 转换为 RGB 模式（如果需要）
            if img.mode != 'RGB':
                img = img.convert('RGB')
                
        except Exception as e:
            logger.error(f"图片解码失败: {e}")
            return jsonify({"code": 400, "data": "", "message": "图片格式错误"})
        
        # 获取目标文字（如果有）
        target_text = data.get('target_text', None)
        
        # 执行 OCR 识别
        logger.info("开始执行 OCR 识别...")
        # 移除 cls 参数，使用默认设置
        result = ocr.ocr(img)
        
        # 处理识别结果
        if result and len(result) > 0:
            response = process_ocr_result(result[0], target_text)
            logger.info(f"OCR 识别完成: {response['message']}")
        else:
            response = {"code": 200, "data": "", "message": "没有识别到文字"}
            
        return jsonify(response)
        
    except Exception as e:
        logger.error(f"OCR 识别过程中发生错误: {e}")
        logger.error(traceback.format_exc())
        return jsonify({"code": 500, "data": "", "message": f"服务器内部错误: {str(e)}"})

@app.route('/health', methods=['GET'])
def health_check():
    """健康检查接口"""
    status = "ready" if ocr_initialized else "initializing"
    return jsonify({"status": status, "message": "OCR 服务运行中"})

@app.route('/', methods=['GET'])
def index():
    """首页"""
    return jsonify({
        "service": "PaddleOCR HTTP API",
        "version": "1.0.0",
        "status": "ready" if ocr_initialized else "initializing",
        "endpoints": {
            "POST /api/ocr": "OCR 文字识别",
            "GET /health": "健康检查",
            "GET /": "服务信息"
        }
    })

if __name__ == '__main__':
    logger.info("启动 PaddleOCR HTTP 服务...")
    
    # 预先初始化 OCR
    initialize_paddleocr()
    
    # 启动 Flask 服务
    app.run(
        host='127.0.0.1',
        port=1224,
        debug=False,
        threaded=True
    ) 