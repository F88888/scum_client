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
from PIL import Image, ImageEnhance, ImageFilter
import numpy as np
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
            try:
                dirs[:] = [d for d in dirs if os.access(os.path.join(root, d), os.R_OK)]
            except Exception as e:
                logger.debug(f"遍历目录时出错: {e}")
                continue
            
            for filename in files:
                if not (filename.endswith('.tar.gz') or filename.endswith('.tar')):
                    continue
                    
                filepath = os.path.join(root, filename)
                
                # 检查文件是否可访问
                if not os.access(filepath, os.R_OK):
                    logger.debug(f"无权限访问文件，跳过: {filepath}")
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
                            logger.debug(f"无法删除文件 {filepath}: {e}")
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
                            logger.debug(f"无法删除损坏的文件 {filepath}: {del_e}")
                        
                except Exception as e:
                    logger.debug(f"检查文件时出错 {filepath}: {e}")
                    
    except Exception as e:
        logger.debug(f"遍历缓存目录时出错: {e}")
    
    if cleaned_count > 0:
        logger.info(f"共清理了 {cleaned_count} 个损坏的模型文件")
    else:
        logger.info("未发现损坏的模型文件")
    
    return cleaned_count

def get_script_directory():
    """
    获取脚本所在目录的绝对路径
    
    @description: 获取当前脚本文件的绝对路径，兼容各种运行方式
    @return: script_dir string 脚本目录的绝对路径
    """
    try:
        if getattr(sys, 'frozen', False):
            # 如果是打包后的可执行文件
            script_dir = os.path.dirname(sys.executable)
        else:
            # 如果是普通 Python 脚本
            script_dir = os.path.dirname(os.path.abspath(__file__))
        
        return script_dir
    except Exception as e:
        logger.warning(f"获取脚本目录失败: {e}, 使用当前工作目录")
        return os.getcwd()

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
        
        # 获取脚本目录
        script_dir = get_script_directory()
        logger.info(f"脚本目录: {script_dir}")
        
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
            logger.debug(f"清理损坏模型文件时出错（继续初始化）: {e}")
        
        # 设置环境变量指定模型缓存路径
        os.environ['PADDLEOCR_MODEL_PATH'] = paddle_cache_dir
        os.environ['PADDLEOCR_HOME'] = paddle_cache_dir
        
        # 检查是否存在已下载的自定义模型
        # 优先使用绝对路径，然后尝试相对于脚本目录的路径
        custom_model_paths = [
            os.path.join(script_dir, "paddle_models", "en_PP-OCRv4_mobile_rec_infer"),
            os.path.join(script_dir, "paddle_models", "PP-OCRv5_mobile_det"),
            os.path.abspath("paddle_models/en_PP-OCRv4_mobile_rec_infer"),
            os.path.abspath("paddle_models/PP-OCRv5_mobile_det"),
        ]
        
        custom_model_found = False
        custom_model_path = None
        
        for model_path in custom_model_paths:
            if os.path.exists(model_path) and os.path.isdir(model_path):
                # 检查目录是否包含模型文件
                try:
                    model_files = [f for f in os.listdir(model_path) if f.endswith(('.pdmodel', '.pdiparams'))]
                    if model_files:
                        custom_model_found = True
                        custom_model_path = model_path
                        logger.info(f"找到自定义模型: {model_path}")
                        break
                except Exception as e:
                    logger.debug(f"检查模型目录时出错 {model_path}: {e}")
        
        # 初始化 PaddleOCR
        logger.info("开始加载 PaddleOCR 模型...")
        
        if custom_model_found:
            logger.info(f"使用自定义识别模型: {custom_model_path}")
            try:
                ocr = PaddleOCR(
                    use_gpu=False,
                    lang='ch',                      # 中英文混合识别
                    rec_model_dir=custom_model_path,
                    show_log=False,
                    use_angle_cls=True,             # 启用角度分类，提高倾斜/模糊文字识别准确度
                    det=True,
                    rec=True,
                    # 性能优化参数
                    use_mp=True,                    # 启用多进程加速
                    total_process_num=2,            # 进程数量
                    det_db_thresh=0.4,              # 提高检测阈值，提高模糊文字检测准确度（从0.3提高到0.4）
                    det_db_box_thresh=0.5,          # 框选阈值
                    det_db_unclip_ratio=1.6,        # 文本框扩展比例
                    max_batch_size=10,              # 批处理大小
                    rec_batch_num=6,                # 识别批处理数量
                    use_dilation=True,              # 启用膨胀，提高模糊文字检测准确度
                    det_db_score_mode='slow'        # 使用慢速评分模式，提高准确度
                )
            except Exception as e:
                logger.warning(f"加载自定义模型失败: {e}，将使用默认模型")
                custom_model_found = False
        
        if not custom_model_found:
            logger.info("使用默认中英文混合识别模型（mobile版本，首次使用将自动下载）")
            ocr = PaddleOCR(
                use_gpu=False,
                lang='ch',                      # 中英文混合识别（支持中文、英文、数字）
                show_log=False,
                use_angle_cls=True,             # 启用角度分类，提高倾斜/模糊文字识别准确度
                det=True,                       # 启用文字检测
                rec=True,                       # 启用文字识别
                # 性能优化参数
                use_mp=True,                    # 启用多进程加速
                total_process_num=2,            # 进程数量（避免过多占用资源）
                det_db_thresh=0.4,              # 提高检测阈值，提高模糊文字检测准确度（从0.3提高到0.4）
                det_db_box_thresh=0.5,          # 框选阈值
                det_db_unclip_ratio=1.6,        # 文本框扩展比例
                max_batch_size=10,              # 批处理大小
                rec_batch_num=6,                # 识别批处理数量
                # 使用轻量级模型
                det_model_dir=None,             # 使用默认检测模型
                rec_model_dir=None,             # 使用默认识别模型
                use_dilation=True,              # 启用膨胀，提高模糊文字检测准确度
                det_db_score_mode='slow'        # 使用慢速评分模式，提高准确度
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
            if 'paddle_cache_dir' in locals() and os.path.exists(paddle_cache_dir):
                # 只清理 whl 目录下的模型文件
                whl_dir = os.path.join(paddle_cache_dir, "whl")
                if os.path.exists(whl_dir):
                    shutil.rmtree(whl_dir)
                    logger.info(f"已清理模型缓存目录: {whl_dir}")
                    logger.info("请重启服务以重新下载模型")
        except Exception as clean_error:
            logger.error(f"清理缓存失败: {clean_error}")
        
        return False

def preprocess_image(img):
    """
    预处理图片（图像增强以提高模糊文字识别准确度）
    
    @description: 对图片进行锐化、对比度增强等处理，提高模糊文字的识别准确度，不改变图片尺寸
    @param: img Image PIL图片对象
    @return: img_array ndarray 处理后的numpy数组
    """
    # 确保图片是 RGB 模式
    if img.mode != 'RGB':
        img = img.convert('RGB')
    
    # 1. 锐化处理（Unsharp Masking）- 提高文字边缘清晰度
    # 使用轻微锐化，避免过度处理导致噪点
    img = img.filter(ImageFilter.UnsharpMask(radius=1, percent=150, threshold=3))
    
    # 2. 对比度增强 - 提高文字与背景的对比度
    enhancer = ImageEnhance.Contrast(img)
    img = enhancer.enhance(1.3)  # 增强30%对比度
    
    # 3. 轻微锐化边缘（针对模糊文字）
    # 使用边缘增强滤波器
    img = img.filter(ImageFilter.EDGE_ENHANCE_MORE)
    
    # 4. 转换为 numpy 数组，保持原始分辨率
    img_array = np.array(img)
    
    return img_array

def process_ocr_result(result, target_text=None):
    """
    处理 OCR 识别结果
    
    @description: 提取OCR识别结果中的文字和坐标，可选检查是否包含目标文字
    @param: result list OCR识别结果
    @param: target_text string 目标文字（可选）
    @return: response dict 处理后的响应数据
    """
    if not result:
        return {
            "code": 200, 
            "data": "", 
            "items": [],
            "message": "没有识别到文字"
        }
    
    # 提取所有文字和坐标
    texts = []
    items = []
    
    for line in result:
        if len(line) >= 2:
            # line[0] 是坐标信息（四个顶点）
            # line[1] 是 [文字内容, 置信度]
            coordinates = line[0]  # [[x1,y1], [x2,y2], [x3,y3], [x4,y4]]
            text = line[1][0].strip()
            confidence = line[1][1]
            
            if text:
                texts.append(text)
                items.append({
                    "text": text,
                    "confidence": float(confidence),
                    "box": coordinates,  # 文字框的四个顶点坐标
                    "position": {
                        "left": int(min(coord[0] for coord in coordinates)),
                        "top": int(min(coord[1] for coord in coordinates)),
                        "right": int(max(coord[0] for coord in coordinates)),
                        "bottom": int(max(coord[1] for coord in coordinates))
                    }
                })
    
    # 合并所有文字
    full_text = " ".join(texts).strip()
    
    # 如果指定了目标文字，检查是否匹配
    if target_text:
        # 忽略大小写和空格进行比较
        clean_full_text = full_text.replace(" ", "").lower()
        clean_target_text = target_text.replace(" ", "").lower()
        
        if clean_target_text in clean_full_text:
            return {
                "code": 100, 
                "data": target_text, 
                "items": items,
                "full_text": full_text,
                "message": "识别成功"
            }
        else:
            return {
                "code": 200, 
                "data": full_text, 
                "items": items,
                "message": "未找到目标文字"
            }
    
    return {
        "code": 100, 
        "data": full_text, 
        "items": items,
        "message": "识别成功"
    }

@app.route('/api/ocr', methods=['POST'])
def ocr_recognition():
    """
    OCR 识别接口
    
    @Tags OCR
    @Summary OCR文字识别（含坐标）
    @Description 接收Base64编码的图片，返回识别的文字内容、坐标和置信度
    @Accept application/json
    @Produce application/json
    @Success 200 {object} response.Response{data=string,items=array} "识别成功，返回文字、坐标、置信度"
    @Failure 400 {object} response.Response "请求错误"
    @Failure 500 {object} response.Response "服务器错误"
    @Router /api/ocr [post]
    
    返回数据格式：
    {
        "code": 100,
        "data": "合并的完整文字",
        "items": [
            {
                "text": "识别的文字",
                "confidence": 0.98,
                "box": [[x1,y1], [x2,y2], [x3,y3], [x4,y4]],  // 四个顶点坐标
                "position": {
                    "left": x1,
                    "top": y1,
                    "right": x2,
                    "bottom": y2
                }
            }
        ],
        "message": "识别成功"
    }
    """
    global ocr
    
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
    try:
        logger.info("开始执行 OCR 识别...")
        
        # 预处理图片（转换为 numpy 数组，保持原始分辨率）
        img_array = preprocess_image(img)
        
        # 执行 OCR 识别（使用 cls=False 加快速度）
        result = ocr.ocr(img_array, cls=False)

        # 处理识别结果
        if result and len(result) > 0:
            response = process_ocr_result(result[0], target_text)
            item_count = len(response.get('items', []))
            logger.info(f"OCR 识别完成: {response['message']}, 识别到 {item_count} 个文本块, 完整文字: {response['data']}")
        else:
            response = {"code": 200, "data": "", "items": [], "message": "没有识别到文字"}

        return jsonify(response)
        
    except Exception as e:
        logger.error(f"OCR 识别失败: {e}")
        logger.error(f"错误详情:\n{traceback.format_exc()}")
        return jsonify({"code": 500, "data": "", "message": f"OCR 识别失败: {str(e)}"})

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
        "version": "1.3.0",
        "language": "中英文混合识别",
        "supported_languages": ["中文", "英文", "数字", "标点符号"],
        "features": [
            "文字识别",
            "坐标定位",
            "置信度评分",
            "多文本块识别"
        ],
        "optimizations": [
            "多进程加速",
            "原始分辨率识别（保证准确度）",
            "图像增强预处理（锐化、对比度增强）",
            "角度分类支持（提高倾斜文字识别）",
            "优化检测阈值（提高模糊文字识别准确度）"
        ],
        "status": "ready" if ocr_initialized else "initializing",
        "endpoints": {
            "POST /api/ocr": "OCR 文字识别（含坐标、置信度）",
            "GET /health": "健康检查",
            "GET /": "服务信息"
        },
        "response_format": {
            "data": "合并的完整文字",
            "items": [
                {
                    "text": "识别的文字",
                    "confidence": "置信度(0-1)",
                    "box": "四个顶点坐标",
                    "position": "矩形边界框(left,top,right,bottom)"
                }
            ]
        }
    })

if __name__ == '__main__':
    # 获取并设置工作目录为脚本所在目录
    try:
        script_dir = get_script_directory()
        if script_dir:
            os.chdir(script_dir)
            logger.info(f"工作目录已设置为: {script_dir}")
    except Exception as e:
        logger.warning(f"无法设置工作目录: {e}")
        logger.info(f"当前工作目录: {os.getcwd()}")
    
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
