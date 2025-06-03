#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import json
import re
from pdf2image import convert_from_path, pdfinfo_from_path
import pytesseract

def parse_question_block(block_text):
    """
    解析代表单个问题的文本块，并提取其组成部分。

    参数:
        block_text (str): 单个问题的 OCR 文本块。

    返回:
        dict: 包含已解析问题数据的字典，如果解析失败则返回 None。
              结构:
              {
                  "question_number": str,  # 题号
                  "question_type": str,  # 题目类型，例如："单选题" 或 "多选题"
                  "question_text": str,  # 题目文本
                  "options": {dict},     # 选项，例如：{"A": "选项A的文本", ...}
                  "correct_answer": str, # 正确答案（官方答案或用户答案作为备选）
                  "error_count": int,    # 错误次数，初始化为 0
                  "correct_count": int   # 正确次数，初始化为 0
              }
    """
    # 初始化单个问题的数据结构
    question_data = {
        "question_number": "",
        "question_type": "",
        "question_text": "",
        "options": {},
        "user_answer_internal": "", # 内部存储用户答案，用于逻辑判断，不直接输出到最终JSON
        "correct_answer": "",
        "error_count": 0,
        "correct_count": 0
    }

    # 1. 从头部提取题号、题目类型及剩余内容。
    #    正则表达式捕获: group(1) = 题号, group(2) = 类型, group(3) = 从题目文本开始的剩余文本块。
    #    示例: "1. (单选题) 题目文本和选项..."
    header_match = re.search(r"^\s*(\d+)\s*?\.\s*?\((单选题|多选题)\)\s*([\s\S]*)", block_text)
    if not header_match:
        # 如果文本块与预期的题目头部格式不匹配 (例如，"1. (单选题)...")，则返回 None。
        return None

    question_data["question_number"] = header_match.group(1).strip()
    question_data["question_type"] = header_match.group(2).strip()
    
    # 这是紧跟在题目类型标记 (例如"(单选题)") 之后的内容。
    # 它包括题目描述、选项以及答案区域。
    content_after_type_marker = header_match.group(3)

    # 2. 分离并提取题目文本 (题干)。
    #    题目文本应在第一个选项 (例如 "A.")、"我的答案" 或 "正确答案" 标签之前结束。
    
    # 初始假设题目文本的结束位置是 content_after_type_marker 的末尾。
    end_of_question_text_idx = len(content_after_type_marker)

    # 查找选项区域的起始位置 (例如，以 "A." 开头的新行)。
    # 选项通常在新行开始。
    first_option_marker_match = re.search(r"\n\s*[A-Z]\.", content_after_type_marker)
    if first_option_marker_match:
        end_of_question_text_idx = min(end_of_question_text_idx, first_option_marker_match.start())

    # 查找 "我的答案" 标签的起始位置。
    user_answer_label_match = re.search(r"我的答案", content_after_type_marker)
    if user_answer_label_match:
        end_of_question_text_idx = min(end_of_question_text_idx, user_answer_label_match.start())

    # 查找 "正确答案" 标签的起始位置。
    correct_answer_label_match = re.search(r"正确答案", content_after_type_marker)
    if correct_answer_label_match:
        end_of_question_text_idx = min(end_of_question_text_idx, correct_answer_label_match.start())
        
    # 提取原始的题目文本。
    question_text_raw = content_after_type_marker[:end_of_question_text_idx].strip()
    # 标准化空白字符：将换行符替换为空格，然后将多个连续空格压缩为单个空格。
    question_data["question_text"] = re.sub(r'\s+', ' ', question_text_raw.replace('\n', ' ')).strip()

    # 3. 提取选择题选项 (A, B, C, D 等)。
    #    选项位于题目文本之后，"我的答案" 或 "正确答案" 之前。

    # 定义用于提取选项的文本块的起始索引。
    options_block_start_idx = end_of_question_text_idx
    
    # 如果找到了第一个选项标记，并且它确实在题目文本之后开始，
    # 则更新选项块的起始索引。这确保我们在正确的位置查找选项。
    if first_option_marker_match and first_option_marker_match.start() >= end_of_question_text_idx:
         options_block_start_idx = first_option_marker_match.start()

    # 定义用于提取选项的文本块的结束索引。
    options_block_end_idx = len(content_after_type_marker)
    if user_answer_label_match:
        options_block_end_idx = min(options_block_end_idx, user_answer_label_match.start())
    
    # 如果 "正确答案" 出现在 "我的答案" 之前 (或者 "我的答案" 不存在)，它也会限制选项块的范围。
    if correct_answer_label_match and \
       (not user_answer_label_match or correct_answer_label_match.start() < user_answer_label_match.start()):
        options_block_end_idx = min(options_block_end_idx, correct_answer_label_match.start())

    options_text_content = content_after_type_marker[options_block_start_idx:options_block_end_idx].strip()
    
    # 用于匹配单个选项的正则表达式：例如 "A. 选项文本"。选项文本可能跨越多行。
    # 捕获: group(1) = 选项字母 (例如 "A"), group(2) = 选项文本。
    # 正向预查 `(?=\n\s*[A-Z]\.\s*|$)` 确保选项文本在下一个选项标记或文本块末尾之前结束。
    option_pattern = re.compile(r"^\s*([A-Z])\.\s*([\s\S]+?)(?=\n\s*[A-Z]\.\s*|$)", re.MULTILINE)

    for match in option_pattern.finditer(options_text_content):
        option_letter = match.group(1).strip()
        # 标准化选项文本：替换换行符，压缩空格。
        option_text_raw = match.group(2).strip().replace('\n', ' ')
        option_text_clean = re.sub(r'\s+', ' ', option_text_raw).strip()
        question_data["options"][option_letter] = option_text_clean

    # 4. 提取用户作答。
    #    在原始的 `block_text` 中搜索，以保证稳健性，因为答案标签相对于整个题目块的位置更固定。
    #    允许提取多个字母的答案 (例如，用于多选题的 "ABC")。
    user_answer_value = ""
    # 此正则表达式查找标签同行或下一行的答案字母。
    ua_match = re.search(r"我的答案\s*[:：]?\s*(?:([A-Z]+)|(?:\s*\n+\s*([A-Z]+)))", block_text)
    if ua_match:
        # group(1) 对应同一行的匹配，group(2) 对应下一行(或多行后)的匹配。
        user_answer_value = ua_match.group(1) or ua_match.group(2)
        if user_answer_value:
            user_answer_value = user_answer_value.strip()
    question_data["user_answer_internal"] = user_answer_value # 存储用于后续逻辑处理

    # 5. 提取官方正确答案。
    official_correct_answer_value = ""
    # 使用与用户答案相似的正则表达式，同样允许多字母答案。
    ca_match = re.search(r"正确答案\s*(?:([A-Z]+)|(?:\s*\n+\s*([A-Z]+)))", block_text)
    if ca_match:
        official_correct_answer_value = ca_match.group(1) or ca_match.group(2)
        if official_correct_answer_value:
            official_correct_answer_value = official_correct_answer_value.strip()
            # 清理可能 همراه 答案字母捕获到的额外文本，例如 "答案解析"。
            # 例如，如果 OCR 结果为 "ABC答案解析"，此步骤会提取 "ABC"。
            clean_answer_letters = re.match(r"([A-Z]+)", official_correct_answer_value)
            if clean_answer_letters:
                official_correct_answer_value = clean_answer_letters.group(1)

    # 6. 根据用户需求确定最终的 "correct_answer" 字段值。
    if official_correct_answer_value:
        question_data["correct_answer"] = official_correct_answer_value
    elif user_answer_value: # 没有官方答案，但存在用户答案。
        question_data["correct_answer"] = user_answer_value
    else: # 既没有官方答案，也没有用户答案。
        question_data["correct_answer"] = "" # 按要求留空。
    
    # 返回前删除临时的内部用户答案字段，因为它不在最终请求的JSON结构中。
    del question_data["user_answer_internal"]

    return question_data


def process_pdf_to_json_data(pdf_path):
    """
    处理单个 PDF 文件，提取所有问题，并以字典列表的形式返回。

    参数:
        pdf_path (str): PDF 文件的路径。

    返回:
        list: 字典列表，其中每个字典代表一个已解析的问题。
              如果处理失败或未找到问题，则返回空列表。
    """
    all_questions_in_pdf = [] # 用于存储此PDF中所有提取到的问题数据
    try:
        # 尝试获取 PDF 信息；这也作为对 Poppler 可访问性的早期检查。
        # 如果 Poppler 未找到或无法工作，pdfinfo_from_path 会引发错误。
        pdfinfo_from_path(pdf_path, poppler_path=None) 
        images = convert_from_path(pdf_path, poppler_path=None)
    except Exception as e:
        # 捕获在 PDF 到图片转换过程中发生的错误 (例如 Poppler 相关问题)。
        print(f"错误：转换PDF '{pdf_path}' 为图片时失败：{e}。"
              "请确保 Poppler 工具已正确安装并配置在系统的 PATH 环境变量中。")
        return []

    full_text_from_pdf = "" # 用于累积来自所有页面的 OCR 文本。
    print(f"正在使用 OCR 处理 '{pdf_path}' 中的 {len(images)} 页...")
    for i, image in enumerate(images):
        try:
            # 对当前页面图片执行 OCR。
            # 'chi_sim' 表示简体中文。
            # Tesseract 的页面分割模式 (Page Segmentation Mode, psm) 可以调整以获得更好的结果:
            #   --psm 3: 全自动页面分割 (默认)。通常适用于各种布局。
            #   --psm 4: 假设为单列可变大小文本。
            #   --psm 6: 假设为单个统一的文本块。适用于干净、简单的页面。
            #   --psm 11: 稀疏文本，带方向和脚本检测 (OSD)。
            # 如果默认的 OCR 质量较低，建议尝试调整这些模式。
            page_text = pytesseract.image_to_string(image, lang='chi_sim', config='--psm 3')
            full_text_from_pdf += page_text + "\n\n"  # 在不同页面的文本之间添加分隔符。
        except pytesseract.TesseractNotFoundError:
            # 当 Tesseract OCR 执行文件本身未找到时，会发生此错误。
            print("错误：未找到 Tesseract OCR 执行文件。"
                  "请安装 Tesseract 并确保其路径已添加到系统 PATH 环境变量中，或者在脚本中设置 pytesseract.tesseract_cmd。")
            return [] # 关键错误，无法继续处理。
        except Exception as e:
            # 捕获当前页面 OCR 过程中的其他相关错误。
            print(f"错误：对 '{pdf_path}' 的第 {i+1} 页进行 OCR 处理时失败：{e}")
            continue # 如果可能，尝试处理其他页面。
    
    if not full_text_from_pdf.strip():
        # 如果从所有页面均未识别到任何文本内容。
        print(f"提示：OCR 处理后，未从 '{pdf_path}' 中识别到任何文本内容。")
        return []

    # 将整个 PDF 的 OCR 文本分割成多个块，每个块可能代表一个新问题的开始。
    # 正则表达式使用正向预查 `(?=...)` 来分割文本，这会保留分隔符
    # (即问题开始的模式) 作为后续块的起始部分。
    # 此模式查找 "数字. (类型)" 的格式，例如："1. (单选题)"。
    question_blocks_raw = re.split(r'(?=\s*\d+\s*?\.\s*?\((?:单选题|多选题)\))', full_text_from_pdf)

    for raw_block_text in question_blocks_raw:
        clean_block_text = raw_block_text.strip() # 移除文本块前后的空白字符。
        if not clean_block_text:
            # 跳过因分割操作可能产生的空文本块。
            continue
        
        # 确保文本块确实以问题的头部模式开始。
        # re.split 产生的第一个块可能是第一个问题之前的内容。
        if not re.match(r"^\s*\d+\s*?\.\s*?\((?:单选题|多选题)\)", clean_block_text):
            # 此文本块看起来不像问题的开头，跳过它。
            # print(f"跳过疑似非题目起始的文本块: {clean_block_text[:100]}...") # 用于调试
            continue

        parsed_question = parse_question_block(clean_block_text)
        if parsed_question:
            # 如果解析成功，则将提取到的问题数据添加到列表中。
            all_questions_in_pdf.append(parsed_question)
            
    return all_questions_in_pdf

# --- 主程序执行部分 ---
if __name__ == "__main__":
    # 定义用于保存 JSON 输出文件的目录。
    output_directory = "json_outputs"
    # 如果输出目录不存在，则创建它。
    # `exist_ok=True` 参数可以防止在目录已存在时引发错误。
    os.makedirs(output_directory, exist_ok=True)

    # 定义要处理的 PDF 文件名列表 (例如："0.pdf", "1.pdf", ..., "8.pdf")。
    pdf_filenames = [f"{i}.pdf" for i in range(9)] 

    for pdf_filename in pdf_filenames:
        # 检查当前 PDF 文件是否存在于脚本所在目录中。
        if not os.path.exists(pdf_filename):
            print(f"文件 '{pdf_filename}' 未找到。跳过此文件。")
            # 根据之前的行为/请求，为缺失的 PDF 创建一个空的 JSON 文件。
            empty_json_output_path = os.path.join(output_directory, f"{os.path.splitext(pdf_filename)[0]}.json")
            try:
                with open(empty_json_output_path, 'w', encoding='utf-8') as f_empty:
                    json.dump([], f_empty, ensure_ascii=False, indent=4)
                print(f"已为缺失的 PDF '{pdf_filename}' 创建了一个空的JSON文件：'{empty_json_output_path}'")
            except Exception as e:
                print(f"错误：为 '{pdf_filename}' 创建空JSON文件时失败：{e}")
            continue # 继续处理下一个 PDF 文件。

        print(f"\n--- 开始处理 '{pdf_filename}' ---")
        # 处理 PDF 并获取结构化的问题数据。
        extracted_questions_data = process_pdf_to_json_data(pdf_filename)
        
        # 定义输出 JSON 文件的完整路径。
        # JSON 文件名将与 PDF 文件名相对应 (例如："0.json")。
        json_output_path = os.path.join(output_directory, f"{os.path.splitext(pdf_filename)[0]}.json")
        
        # 将提取的数据写入 JSON 文件。
        try:
            with open(json_output_path, 'w', encoding='utf-8') as f_json_out:
                # `ensure_ascii=False` 确保能正确保存非 ASCII 字符 (如中文字符)。
                # `indent=4` 使 JSON 文件具有良好的可读性 (格式化输出)。
                json.dump(extracted_questions_data, f_json_out, ensure_ascii=False, indent=4)
            print(f"成功将 '{pdf_filename}' 的结果保存到 '{json_output_path}'。")
            print(f"从此 PDF 中提取了 {len(extracted_questions_data)} 个题目。")
        except Exception as e:
            # 捕获在写入 JSON 文件过程中可能发生的错误。
            print(f"错误：将 '{pdf_filename}' 的JSON结果写入 '{json_output_path}' 时失败：{e}")

    print(f"\n--- 所有PDF处理完毕。输出文件位于 '{output_directory}' 目录中。 ---")