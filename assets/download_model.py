#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
PaddleOCR Model Downloader
Downloads the English OCR model for PaddleOCR
"""

import urllib.request
import tarfile
import os
import sys

def download_model():
    """Download and extract the OCR model"""
    
    print('Downloading en_PP-OCRv4_mobile_rec_infer model...')
    
    url = 'https://paddle-model-ecology.bj.bcebos.com/paddlex/official_inference_model/paddle3.0.0/en_PP-OCRv4_mobile_rec_infer.tar'
    filename = 'en_PP-OCRv4_mobile_rec_infer.tar'
    
    try:
        # 确保 paddle_models 目录存在
        models_dir = 'paddle_models'
        if not os.path.exists(models_dir):
            print(f'Creating directory: {models_dir}')
            os.makedirs(models_dir)
        
        # 检查模型是否已经存在
        model_path = os.path.join(models_dir, 'en_PP-OCRv4_mobile_rec_infer')
        if os.path.exists(model_path):
            print(f'Model already exists at: {model_path}')
            return True
        
        # Download the model
        print(f'Downloading from: {url}')
        urllib.request.urlretrieve(url, filename)
        print(f'Model download completed: {filename}')
        
        # Extract the model to paddle_models directory
        print('Extracting model files...')
        with tarfile.open(filename, 'r') as tar:
            tar.extractall(models_dir)
        
        # Clean up
        os.remove(filename)
        print('Model extraction completed!')
        print(f'Model directory: {model_path}')
        
        # 验证模型文件是否正确提取
        if os.path.exists(model_path):
            print('✓ Model successfully downloaded and extracted')
            return True
        else:
            print('✗ Model extraction verification failed')
            return False
        
    except Exception as e:
        print(f'Download failed: {e}')
        return False

if __name__ == '__main__':
    success = download_model()
    if not success:
        sys.exit(1)
