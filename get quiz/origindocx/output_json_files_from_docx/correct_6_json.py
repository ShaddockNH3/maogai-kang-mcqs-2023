import json
import re
import os

def modify_question_json_file(file_path):
    """
    修改指定的JSON文件：
    1. 从 question_text 中提取题目类型 (单选题/多选题) 并填充到 question_type 字段。
    2. 从 question_text 中移除类型标记。
    """
    try:
        # 以读写模式打开文件，先读取内容
        with open(file_path, 'r', encoding='utf-8') as f:
            questions = json.load(f)
    except FileNotFoundError:
        print(f"喵呜！文件 '{file_path}' 没找到呢，请检查一下路径哦。")
        return False
    except json.JSONDecodeError:
        print(f"喵呜！文件 '{file_path}' 的JSON格式好像有点问题，解析失败了。")
        return False
    except Exception as e:
        print(f"喵呜！读取文件 '{file_path}' 时发生了意想不到的错误：{e}")
        return False

    modified_count = 0
    for question in questions:
        text_changed = False
        type_updated = False

        if "question_text" in question and isinstance(question["question_text"], str):
            original_text = question["question_text"]
            text_to_process = original_text
            
            # 尝试匹配题目类型标记，例如 "【单选题】" 或 "【多选题】"
            # 假设标记总是在文本末尾或接近末尾的地方
            type_pattern = r'【(单选题|多选题)】'
            match = re.search(type_pattern, text_to_process)
            
            if match:
                extracted_type = match.group(1) # 获取 "单选题" 或 "多选题"
                
                # 更新 question_type 字段
                # 只有当 question_type 为空或与提取的不一致时才更新，并标记
                if question.get("question_type", "") != extracted_type:
                    question["question_type"] = extracted_type
                    type_updated = True
                
                # 从 question_text 中移除类型标记
                # 使用 re.sub 替换所有匹配到的类型标记（以防万一文本中有多个）
                new_text = re.sub(type_pattern, '', text_to_process).strip()
                # 清理可能因替换产生的多余空格
                new_text = re.sub(r'\s{2,}', ' ', new_text).strip() 
                
                if question["question_text"] != new_text:
                    question["question_text"] = new_text
                    text_changed = True
            
            if text_changed or type_updated:
                modified_count += 1
        else:
            print(f"喵~ 提示：在处理题号 '{question.get('question_number', '未知')}' 时，'question_text' 字段缺失或格式不正确。")


    if modified_count > 0:
        try:
            # 将修改后的内容写回原文件
            with open(file_path, 'w', encoding='utf-8') as f:
                json.dump(questions, f, ensure_ascii=False, indent=4)
            print(f"喵~ 文件 '{file_path}' 已成功更新！总共有 {modified_count} 条记录被修改了哦。")
            return True
        except Exception as e:
            print(f"喵呜！将修改写回文件 '{file_path}' 时发生了错误：{e}")
            return False
    else:
        print(f"喵~ 文件 '{file_path}' 中没有需要按此规则修改的内容，或者所有题目已符合格式。")
        return True # 视为成功，因为没有需要执行的更改

if __name__ == "__main__":
    # 指定要修改的文件名
    # 主人可以把 '6.json' 替换成其他需要修改的JSON文件名哦！
    # 确保这个JSON文件和Python脚本在同一个目录下，或者提供完整路径。
    target_file = "6.json" 

    print(f"喵~ 准备开始修改文件 '{target_file}' ...")
    success = modify_question_json_file(target_file)

    if success:
        print(f"喵~ 文件 '{target_file}' 的处理流程结束啦！")
    else:
        print(f"喵呜~ 文件 '{target_file}' 的处理遇到了一些小状况呢。")