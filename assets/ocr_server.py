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
import shutil
import tarfile
from io import BytesIO
from paddleocr import PaddleOCR
from flask import Flask, request, jsonify
from PIL import Image
import traceback

# 设置日志
logging.basicConfig(
    level=logging.INFO,
    format='[%(asctime)s] [%(levelname)8s] %(filename)s:%(lineno)d - %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger(__name__)

app = Flask(__name__)

# 全局变量
ocr = None
ocr_initialized = False

def check_and_clean_corrupted_models(cache_dir):
    """
    检查并清理损坏的模型文件
    
    @description: 扫描缓存目录中的tar文件，尝试验证其完整性，删除损坏的文件
    @param: cache_dir string 模型缓存目录
    @return: cleaned_count int 清理的文件数量
    """
    cleaned_count = 0
    
    if not os.path.exists(cache_dir):
        logger.info(f"模型缓存目录不存在: {cache_dir}")
        return cleaned_count
    
    logger.info(f"正在检查模型缓存目录: {cache_dir}")
    
    try:
        # 遍历缓存目录及其子目录
        for root, dirs, files in os.walk(cache_dir):
            # 跳过无权限的目录
            dirs[:] = [d for d in dirs if os.access(os.path.join(root, d), os.R_OK)]
            
            for filename in files:
                if not (filename.endswith('.tar.gz') or filename.endswith('.tar')):
                    continue
                    
                filepath = os.path.join(root, filename)
                
                # 检查文件是否可访问
                if not os.access(filepath, os.R_OK):
                    logger.warning(f"无权限访问文件，跳过: {filepath}")
                    continue
                
                # 检查文件大小
                try:
                    file_size = os.path.getsize(filepath)
                    
                    # 如果文件太小（< 100KB），可能是下载不完整
                    if file_size < 100 * 1024:
                        logger.warning(f"发现可疑的模型文件（文件过小）: {filepath} ({file_size} bytes)")
                        try:
                            os.remove(filepath)
                            cleaned_count += 1
                            logger.info(f"已删除损坏的模型文件: {filepath}")
                        except Exception as e:
                            logger.warning(f"无法删除文件 {filepath}: {e}")
                        continue
                    
                    # 尝试打开 tar 文件验证完整性
                    try:
                        with tarfile.open(filepath, 'r:*') as tar:
                            # 尝试读取成员列表
                            members = tar.getmembers()
                            if len(members) == 0:
                                raise Exception("tar文件为空")
                        logger.debug(f"模型文件完整性验证通过: {filepath}")
                    except Exception as e:
                        logger.warning(f"发现损坏的模型文件: {filepath}, 错误: {e}")
                        try:
                            os.remove(filepath)
                            cleaned_count += 1
                            logger.info(f"已删除损坏的模型文件: {filepath}")
                        except Exception as del_e:
                            logger.warning(f"无法删除损坏的文件 {filepath}: {del_e}")
                        
                except Exception as e:
                    logger.warning(f"检查文件时出错 {filepath}: {e}")
                    
    except Exception as e:
        logger.warning(f"遍历缓存目录时出错: {e}")
    
    if cleaned_count > 0:
        logger.info(f"共清理了 {cleaned_count} 个损坏的模型文件")
    else:
        logger.info("未发现损坏的模型文件")
    
    return cleaned_count

def initialize_paddleocr():
    """
    初始化 PaddleOCR
    
    @description: 初始化PaddleOCR实例，设置模型缓存目录，检查并清理损坏的模型文件
    @return: success bool 初始化是否成功
    """
    global ocr, ocr_initialized
    
    if ocr_initialized:
        return True
        
    try:
        logger.info("正在初始化 PaddleOCR...")
        
        # 设置模型缓存目录，避免重复下载
        home_dir = os.path.expanduser("~")
        paddle_cache_dir = os.path.join(home_dir, ".paddleocr")
        
        # 确保缓存目录存在
        try:
            os.makedirs(paddle_cache_dir, exist_ok=True)
            logger.info(f"PaddleOCR 模型缓存目录: {paddle_cache_dir}")
        except Exception as e:
            logger.warning(f"无法创建缓存目录 {paddle_cache_dir}: {e}")
        
        # 检查并清理损坏的模型文件
        try:
            cleaned_count = check_and_clean_corrupted_models(paddle_cache_dir)
            if cleaned_count > 0:
                logger.info("已清理损坏的模型文件，将重新下载")
        except Exception as e:
            logger.warning(f"清理损坏模型文件时出错（继续初始化）: {e}")
        
        # 设置环境变量指定模型缓存路径
        os.environ['PADDLEOCR_MODEL_PATH'] = paddle_cache_dir
        os.environ['PADDLEOCR_HOME'] = paddle_cache_dir
        
        # 检查是否存在已下载的自定义模型
        custom_model_paths = [
            "paddle_models/en_PP-OCRv4_mobile_rec_infer",  # download_model.py 下载的模型
            "paddle_models/PP-OCRv5_mobile_det",           # 旧版本路径
        ]
        
        custom_model_found = False
        custom_model_path = None
        
        for model_path in custom_model_paths:
            if os.path.exists(model_path) and os.path.isdir(model_path):
                # 检查目录是否包含模型文件
                model_files = [f for f in os.listdir(model_path) if f.endswith(('.pdmodel', '.pdiparams'))]
                if model_files:
                    custom_model_found = True
                    custom_model_path = model_path
                    logger.info(f"找到自定义模型: {model_path}")
                    break
        
        # 初始化 PaddleOCR
        logger.info("开始加载 PaddleOCR 模型...")
        
        if custom_model_found:
            logger.info(f"使用自定义英文识别模型: {custom_model_path}")
            try:
                ocr = PaddleOCR(
                    use_gpu=False,
                    lang='en',
                    rec_model_dir=custom_model_path,
                    show_log=False,
                    use_angle_cls=False,
                    det=True,
                    rec=True
                )
            except Exception as e:
                logger.warning(f"加载自定义模型失败: {e}，将使用默认模型")
                custom_model_found = False
        
        if not custom_model_found:
            logger.info("使用默认英文模型（首次使用将自动下载）")
            ocr = PaddleOCR(
                use_gpu=False,
                lang='en',
                show_log=False,
                use_angle_cls=False,  # 禁用角度分类，加快识别速度
                det=True,              # 启用文字检测
                rec=True               # 启用文字识别
            )
        
        ocr_initialized = True
        logger.info("PaddleOCR 初始化完成")
        return True
        
    except Exception as e:
        logger.error(f"PaddleOCR 初始化失败: {e}")
        logger.error(f"错误详情:\n{traceback.format_exc()}")
        
        # 如果初始化失败，尝试清理所有模型缓存
        try:
            logger.warning("初始化失败，尝试清理所有模型缓存...")
            if os.path.exists(paddle_cache_dir):
                # 只清理 whl 目录下的模型文件
                whl_dir = os.path.join(paddle_cache_dir, "whl")
                if os.path.exists(whl_dir):
                    shutil.rmtree(whl_dir)
                    logger.info(f"已清理模型缓存目录: {whl_dir}")
                    logger.info("请重启服务以重新下载模型")
        except Exception as clean_error:
            logger.error(f"清理缓存失败: {clean_error}")
        
        return False

def process_ocr_result(result, target_text=None):
    """
    处理 OCR 识别结果
    
    @description: 提取OCR识别结果中的文字，可选检查是否包含目标文字
    @param: result list OCR识别结果
    @param: target_text string 目标文字（可选）
    @return: response dict 处理后的响应数据
    """
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
    """
    OCR 识别接口
    
    @Tags OCR
    @Summary OCR文字识别
    @Description 接收Base64编码的图片，返回识别的文字内容
    @Accept application/json
    @Produce application/json
    @Success 200 {object} response.Response{data=string} "识别成功"
    @Failure 400 {object} response.Response "请求错误"
    @Failure 500 {object} response.Response "服务器错误"
    @Router /api/ocr [post]
    """
    global ocr
    
    try:
        # 检查 OCR 是否已初始化
        if not ocr_initialized:
            if not initialize_paddleocr():
                return jsonify({"code": 500, "data": "", "message": "OCR 服务初始化失败，请查看日志"})
        
        # 获取请求数据
        data = request.get_json()
        if not data or 'image' not in data:
            return jsonify({"code": 400, "data": "", "message": "缺少 Base64 图片数据"})
        
        # 解码 Base64 图片
        base64_str = data['image']
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
        result = ocr.ocr(img)
        
        # 处理识别结果
        if result and len(result) > 0:
            response = process_ocr_result(result[0], target_text)
            logger.info(f"OCR 识别完成: {response['message']}, 识别文字: {response['data']}")
        else:
            response = {"code": 200, "data": "", "message": "没有识别到文字"}
            
        return jsonify(response)
        
    except Exception as e:
        logger.error(f"OCR 识别过程中发生错误: {e}")
        logger.error(traceback.format_exc())
        return jsonify({"code": 500, "data": "", "message": f"服务器内部错误: {str(e)}"})

@app.route('/health', methods=['GET'])
def health_check():
    """
    健康检查接口
    
    @Tags 系统
    @Summary 健康检查
    @Description 检查OCR服务运行状态
    @Accept application/json
    @Produce application/json
    @Success 200 {object} response.Response "成功"
    @Router /health [get]
    """
    status = "ready" if ocr_initialized else "initializing"
    return jsonify({"status": status, "message": "OCR 服务运行中"})

@app.route('/', methods=['GET'])
def index():
    """
    首页
    
    @Tags 系统
    @Summary 服务信息
    @Description 获取OCR服务的基本信息和可用端点
    @Accept application/json
    @Produce application/json
    @Success 200 {object} response.Response "成功"
    @Router / [get]
    """
    return jsonify({
        "service": "PaddleOCR HTTP API",
        "version": "1.0.2",
        "status": "ready" if ocr_initialized else "initializing",
        "endpoints": {
            "POST /api/ocr": "OCR 文字识别",
            "GET /health": "健康检查",
            "GET /": "服务信息"
        }
    })

if __name__ == '__main__':
    # 设置工作目录为脚本所在目录
    script_dir = os.path.dirname(os.path.abspath(__file__))
    if script_dir:
        os.chdir(script_dir)
        logger.info(f"工作目录设置为: {script_dir}")
    
    logger.info("="*60)
    logger.info("启动 PaddleOCR HTTP 服务...")
    logger.info("="*60)
    
    # 预先初始化 OCR
    if initialize_paddleocr():
        logger.info("OCR 服务已就绪")
    else:
        logger.error("OCR 服务初始化失败，但服务仍将启动（将在首次请求时重试）")
    
    # 启动 Flask 服务
    logger.info("="*60)
    logger.info("Flask 服务监听: http://127.0.0.1:1224")
    logger.info("="*60)
    
    app.run(
        host='127.0.0.1',
        port=1224,
        debug=False,
        threaded=True
    )
