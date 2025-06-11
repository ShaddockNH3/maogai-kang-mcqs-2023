import re
import json
import os
from docx import Document # 导入 python-docx 库

# user_provided_text_for_0_docx = """...""" # 这一大段现在可以删掉了，我们会从文件读取

def read_docx_file(file_path):
    """
    从指定的 .docx 文件中读取所有段落的文本内容。
    """
    try:
        doc = Document(file_path)
        full_text = []
        for para in doc.paragraphs:
            full_text.append(para.text)
        return '\n'.join(full_text)
    except Exception as e:
        print(f"错误：读取或解析 DOCX 文件 '{file_path}' 时失败 - {e}")
        return None

def parse_questions_from_text(text_content):
    """
    从给定的文本内容中解析题目信息。
    (此函数与上一版本基本相同，核心解析逻辑不变)
    """
    questions = []
    if not text_content: # 如果文本内容为空（例如文件读取失败）
        return questions

    # 匹配每个题目的起始标志，例如 "1、1." 或 "10、10."
    # 使用正向预查来分割文本，使得每个块都以题目号开始
    # 在文本开头加一个换行符，以确保第一个题目也能被正确分割
    raw_question_blocks = re.split(r'\n(?=\s*\d+、\d+\.)', "\n" + text_content.strip())

    for block in raw_question_blocks:
        block = block.strip()
        if not block:
            continue

        question_data = {
            "question_number": "",
            "question_type": "",
            "question_text": "",
            "options": {},
            "correct_answer": "",
            "error_count": 0,
            "correct_count": 0
        }
        
        lines = block.split('\n')
        
        first_line = lines.pop(0).strip()
        match_first_line = re.match(r'(\d+)、\d+\.\s*(.*)', first_line)
        if not match_first_line:
            continue 
        
        question_data["question_number"] = match_first_line.group(1)
        current_question_text_parts = [match_first_line.group(2).strip()]
        
        parsing_options = False
        current_option_letter = None
        current_option_text_accumulator = []

        for line_idx, line_content in enumerate(lines):
            line = line_content.strip()
            if not line: 
                continue

            if not question_data["question_type"]:
                type_score_pattern = r'【(单选题|多选题)】\s*（\d+分）'
                match_type_score = re.search(type_score_pattern, line)
                if match_type_score:
                    question_data["question_type"] = match_type_score.group(1)
                    line = re.sub(type_score_pattern, '', line).strip()
                    if not line: continue 
            
            match_correct_answer = re.match(r'正确答案:\s*([A-Z]+)', line)
            if match_correct_answer:
                if current_option_letter: 
                    option_text = "".join(current_option_text_accumulator).strip()
                    question_data["options"][current_option_letter] = re.sub(r'\s+', ' ', option_text)
                question_data["correct_answer"] = match_correct_answer.group(1)
                parsing_options = True 
                break 

            match_option = re.match(r'([A-Z])\.(.*)', line)
            if match_option:
                parsing_options = True
                if current_option_letter: 
                    option_text = "".join(current_option_text_accumulator).strip()
                    question_data["options"][current_option_letter] = re.sub(r'\s+', ' ', option_text)
                
                current_option_letter = match_option.group(1)
                current_option_text_accumulator = [match_option.group(2).strip()]
                continue 
            elif parsing_options and current_option_letter:
                current_option_text_accumulator.append(" " + line) 
            elif not parsing_options:
                current_question_text_parts.append(line)
        
        if current_option_letter and not question_data["correct_answer"]:
             option_text = "".join(current_option_text_accumulator).strip()
             question_data["options"][current_option_letter] = re.sub(r'\s+', ' ', option_text)

        full_question_text = " ".join(current_question_text_parts).strip()
        type_score_pattern_for_cleanup = r'【(单选题|多选题)】\s*（\d+分）'
        if not question_data["question_type"]: 
            match_type_in_text = re.search(type_score_pattern_for_cleanup, full_question_text)
            if match_type_in_text:
                question_data["question_type"] = match_type_in_text.group(1)
        full_question_text = re.sub(type_score_pattern_for_cleanup, '', full_question_text).strip()
        question_data["question_text"] = re.sub(r'\s+', ' ', full_question_text).strip()

        if question_data["question_number"] == "40":
            if "【多选题】" in block: 
                question_data["question_type"] = "多选题"
            if "E" in question_data["options"] and "多选题" in question_data["options"]["E"]:
                del question_data["options"]["E"]

        if question_data["question_number"]:
            questions.append(question_data)
            
    return questions

def main():
    """
    主函数，负责读取输入文件、调用解析函数并写入JSON文件。
    """
    input_dir = "."  # 假设 .docx 文件与脚本在同一目录下，或者指定其他目录
    output_dir = "output_json_files_from_docx" 
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)

    for i in range(9): # 对应文件 0.docx 到 8.docx
        file_base_name = str(i)
        # 构建 .docx 文件的路径
        docx_file_path = os.path.join(input_dir, f"{file_base_name}.docx")
        output_json_path = os.path.join(output_dir, f"{file_base_name}.json")
        
        questions_for_this_file = []

        if os.path.exists(docx_file_path):
            print(f"正在读取并解析 '{docx_file_path}'...")
            text_content_from_docx = read_docx_file(docx_file_path)
            if text_content_from_docx:
                questions_for_this_file = parse_questions_from_text(text_content_from_docx)
                print(f"从 '{docx_file_path}' 解析到 {len(questions_for_this_file)} 个题目。")
            else:
                print(f"未能从 '{docx_file_path}' 读取到内容。")
        else:
            print(f"提示: 文件 '{docx_file_path}' 未找到，将生成空的 '{file_base_name}.json'。")
        
        try:
            with open(output_json_path, 'w', encoding='utf-8') as f:
                json.dump(questions_for_this_file, f, ensure_ascii=False, indent=4)
            print(f"已成功将结果写入到: {output_json_path}")
        except Exception as e:
            print(f"错误: 写入JSON文件 '{output_json_path}' 时失败 - {e}")

if __name__ == '__main__':
    main()