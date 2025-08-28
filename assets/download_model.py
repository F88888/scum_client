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
        # Download the model
        print(f'Downloading from: {url}')
        urllib.request.urlretrieve(url, filename)
        print(f'Model download completed: {filename}')
        
        # Extract the model
        print('Extracting model files...')
        with tarfile.open(filename, 'r') as tar:
            tar.extractall('.')
        
        # Clean up
        os.remove(filename)
        print('Model extraction completed!')
        print('Model directory: en_PP-OCRv4_mobile_rec_infer')
        
        return True
        
    except Exception as e:
        print(f'Download failed: {e}')
        return False

if __name__ == '__main__':
    success = download_model()
    if not success:
        sys.exit(1)
