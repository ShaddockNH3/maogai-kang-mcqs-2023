import os
import json
import random
import time # 用于可能的暂停或延迟

class MainController:
    def __init__(self, data_dir="clean_outputs", incorrect_file="incorrect_questions.json"):
        """
        初始化主控制器。

        参数:
            data_dir (str): 存储章节题目JSON文件的目录名。
            incorrect_file (str): 存储错题记录的JSON文件名。
        """
        self.data_directory = data_dir
        self.incorrect_questions_file = incorrect_file
        self.all_questions_data = {}  # 存储所有章节的题目数据，键为章节号(字符串)
        self.incorrect_questions = [] # 存储错题列表

        if not os.path.exists(self.data_directory):
            print(f"喵呜！错误：数据目录 '{self.data_directory}' 未找到。请确保题目JSON文件已放置在该目录下哦。")
            # 程序可能无法正常运行，这里仅打印错误

        self.load_all_questions()
        self.load_incorrect_questions()

    def load_all_questions(self):
        """
        从JSON文件加载所有章节的题目数据到 self.all_questions_data。
        """
        print("喵~ 正在努力加载题库中...")
        for i in range(9): # 对应 0.json 到 8.json
            file_path = os.path.join(self.data_directory, f"{i}.json")
            try:
                with open(file_path, 'r', encoding='utf-8') as f:
                    self.all_questions_data[str(i)] = json.load(f)
            except FileNotFoundError:
                print(f"喵~ 提示：章节 {i} 的题库文件 ('{file_path}') 没找到呢，跳过这个章节啦。")
                self.all_questions_data[str(i)] = [] 
            except json.JSONDecodeError:
                print(f"喵呜！错误：解析章节 {i} 的题库文件 ('{file_path}') 失败了，文件格式可能不太对哦。")
                self.all_questions_data[str(i)] = []
            except Exception as e:
                print(f"喵呜！加载章节 {i} ('{file_path}') 时发生了意想不到的问题: {e}")
                self.all_questions_data[str(i)] = []
        print("喵~ 题库加载完毕！一切准备就绪！")


    def load_incorrect_questions(self):
        """
        从JSON文件加载错题记录到 self.incorrect_questions。
        """
        try:
            if os.path.exists(self.incorrect_questions_file):
                with open(self.incorrect_questions_file, 'r', encoding='utf-8') as f:
                    self.incorrect_questions = json.load(f)
            else:
                self.incorrect_questions = []
        except json.JSONDecodeError:
            print(f"喵呜！错误：解析错题记录文件 ('{self.incorrect_questions_file}') 失败了，错题簿暂时是空的哦。")
            self.incorrect_questions = []
        except Exception as e:
            print(f"喵呜！加载错题记录时发生了意想不到的问题: {e}")
            self.incorrect_questions = []


    def save_incorrect_questions(self):
        """
        将当前的错题列表保存到JSON文件。
        """
        try:
            with open(self.incorrect_questions_file, 'w', encoding='utf-8') as f:
                json.dump(self.incorrect_questions, f, ensure_ascii=False, indent=4)
        except Exception as e:
            print(f"喵呜！错误：保存错题记录到 '{self.incorrect_questions_file}' 时失败了: {e}")

    def save_chapter_data(self, chapter_key_str):
        """
        将指定章节的题目数据 (包含更新的对错次数) 保存回其JSON文件。
        """
        if chapter_key_str not in self.all_questions_data:
            print(f"喵呜！错误：尝试保存一个不存在的章节数据 (章节键: {chapter_key_str})。")
            return

        file_path = os.path.join(self.data_directory, f"{chapter_key_str}.json")
        try:
            os.makedirs(self.data_directory, exist_ok=True) # 确保目录存在
            with open(file_path, 'w', encoding='utf-8') as f:
                json.dump(self.all_questions_data[chapter_key_str], f, ensure_ascii=False, indent=4)
        except Exception as e:
            print(f"喵呜！错误：保存章节 {chapter_key_str} 的数据到 '{file_path}' 时失败了: {e}")

    def _get_questions_for_processing(self, chapter_choice_str, order_choice_str):
        """
        根据用户选择的章节和顺序模式，准备题目列表以供处理。
        """
        questions_to_process = []
        target_chapter_keys = []

        if chapter_choice_str == '9': 
            target_chapter_keys = sorted(self.all_questions_data.keys())
        elif chapter_choice_str in self.all_questions_data:
            target_chapter_keys = [chapter_choice_str]
        else:
            print("喵~ 无效的章节选择呢。")
            return []

        for chap_key in target_chapter_keys:
            chapter_questions = self.all_questions_data.get(chap_key, [])
            for idx, question in enumerate(chapter_questions):
                questions_to_process.append({
                    'question_obj': question,
                    'chapter_key': chap_key,
                    'original_idx_in_chapter': idx
                })
        
        if not questions_to_process:
            print("喵~ 提示：所选范围内没有题目哦。")
            return []

        if order_choice_str == '1': 
            random.shuffle(questions_to_process)
        
        return questions_to_process

    def _display_question_and_options(self, question_obj):
        """
        辅助函数，用于显示题目文本和选项。
        """
        print(f"\n题目序号: {question_obj.get('question_number', 'N/A')}")
        print(f"类型: {question_obj.get('question_type', 'N/A')}")
        print(f"题目: {question_obj.get('question_text', 'N/A')}")
        options = question_obj.get('options', {})
        if options:
            print("选项:")
            for opt_letter, opt_text in sorted(options.items()):
                print(f"  {opt_letter}. {opt_text}")
        else:
            print("  (喵~ 这个题目好像没有选项信息)")


    def quick_review_mode(self):
        """
        执行速刷模式。
        """
        print("\n--- 速刷模式 ---")
        order_choice = input("请选择模式：0. 正序  1. 随机 (输入其他返回主菜单): ")
        if order_choice not in ['0', '1']:
            return

        chapter_choice = input("请选择章节：0-8 选择单个章节, 9 选择全部章节 (输入其他返回主菜单): ")
        if not (chapter_choice.isdigit() and (0 <= int(chapter_choice) <= 9)):
            return
        
        questions_for_review = self._get_questions_for_processing(chapter_choice, order_choice)

        if not questions_for_review:
            print("喵~ 没有可供速刷的题目呢。")
            input("按 Enter键 返回主菜单...")
            return
        
        user_quit = False
        for item in questions_for_review:
            question = item['question_obj']
            self._display_question_and_options(question)
            
            action = input("按 Enter键 查看答案 (输入 'q' 或 'quit' 可提前返回主菜单): ").strip().lower()
            if action in ['q', 'quit']:
                print("喵~ 已退出速刷模式。")
                user_quit = True
                break
            
            print(f"正确答案: {question.get('correct_answer', 'N/A')}")
            print("-" * 20)
            # time.sleep(0.5) # 如果觉得信息滚动太快可以取消注释

        if not user_quit:
            print("\n速刷完成！真是太棒了喵！ (๑•̀ㅂ•́)و✧")
        input("按 Enter键 返回主菜单...")


    def quiz_mode(self):
        """
        执行答题模式。
        """
        print("\n--- 答题模式 ---")
        order_choice = input("请选择模式：0. 正序  1. 随机 (输入其他返回主菜单): ")
        if order_choice not in ['0', '1']:
            return

        chapter_choice = input("请选择章节：0-8 选择单个章节, 9 选择全部章节 (输入其他返回主菜单): ")
        if not (chapter_choice.isdigit() and (0 <= int(chapter_choice) <= 9)):
            return

        questions_for_quiz = self._get_questions_for_processing(chapter_choice, order_choice)

        if not questions_for_quiz:
            print("喵~ 没有可供答题的题目呢。")
            input("按 Enter键 返回主菜单...")
            return

        total_answered = 0
        total_correct = 0
        chapters_involved = set() 
        user_quit = False

        for item in questions_for_quiz:
            question = item['question_obj'] 
            chapter_key = item['chapter_key']

            self._display_question_and_options(question)
            
            user_answer_input = input("请输入你的答案 (例如: A, B, ABC) (输入 'q' 或 'quit' 可提前退出答题): ").strip()
            if user_answer_input.lower() in ['q', 'quit']:
                print("喵~ 已中途退出答题模式。")
                user_quit = True
                break 
            
            user_answer = user_answer_input.upper()
            total_answered += 1
            chapters_involved.add(chapter_key)

            correct_answer_str = str(question.get('correct_answer', '')).strip().upper()

            if user_answer == correct_answer_str:
                print("回答正确！太厉害了喵！(๑•̀ㅂ•́)و✧")
                total_correct += 1
                question['correct_count'] = question.get('correct_count', 0) + 1
            else:
                print("回答错误！喵呜~ (｡•́︿•̀｡)")
                print(f"正确答案是: {correct_answer_str}")
                question['error_count'] = question.get('error_count', 0) + 1
                
                incorrect_q_entry = {
                    "question_number": question.get('question_number', 'N/A'),
                    "question_type": question.get('question_type', 'N/A'),
                    "question_text": question.get('question_text', 'N/A'),
                    "options": question.get('options', {}),
                    "correct_answer": correct_answer_str,
                    "original_chapter": chapter_key
                }
                is_duplicate = any(
                    iq.get("question_text") == incorrect_q_entry["question_text"] and \
                    iq.get("original_chapter") == incorrect_q_entry["original_chapter"]
                    for iq in self.incorrect_questions
                )
                if not is_duplicate:
                    self.incorrect_questions.append(incorrect_q_entry)
            print("-" * 20)
            # time.sleep(0.5)

        print("\n答题会话结束！")
        if total_answered > 0: # 只有在回答了题目后才显示统计和保存
            print(f"本次总共做了 {total_answered} 题，答对 {total_correct} 题，答错 {total_answered - total_correct} 题。")

            for chap_key in chapters_involved:
                self.save_chapter_data(chap_key)
            
            self.save_incorrect_questions()
            print("喵~ 答题记录和错题本已更新！")
        elif not user_quit : # 没有答题且不是用户主动退出（比如题目列表为空）
             print("本次没有回答任何题目。")
        
        input("按 Enter键 返回主菜单...")


    def incorrect_questions_mode(self):
        """
        执行错题回顾模式 (仅随机)。
        """
        print("\n--- 错题回顾模式 ---")
        self.load_incorrect_questions() 

        if not self.incorrect_questions:
            print("错题簿是空的哦！太棒了，说明主人没有错题！ (＾▽＾)")
            input("按 Enter键 返回主菜单...")
            return

        print(f"喵~ 当前错题簿中共有 {len(self.incorrect_questions)} 道错题。将随机展示。")
        
        questions_to_review = random.sample(self.incorrect_questions, len(self.incorrect_questions)) 
        user_quit = False

        for idx, question in enumerate(questions_to_review):
            print(f"\n错题回顾 ({idx + 1}/{len(questions_to_review)})")
            print(f"来源章节: {question.get('original_chapter', 'N/A')}")
            self._display_question_and_options(question)
            
            action = input("按 Enter键 查看答案 (输入 'q' 或 'quit' 可提前返回主菜单): ").strip().lower()
            if action in ['q', 'quit']:
                print("喵~ 已退出错题回顾。")
                user_quit = True
                break
                
            print(f"正确答案: {question.get('correct_answer', 'N/A')}")
            print("-" * 20)
            # time.sleep(0.5)

        if not user_quit:
            print("\n错题回顾完成！希望对主人有帮助喵！")
        input("按 Enter键 返回主菜单...")


    def control_mode(self):
        """
        执行控制模式。
        """
        print("\n--- 控制模式 ---")
        print("1. 清理数据 (重置所有题目的对错次数，并删除错题簿)")
        print("0. 返回主菜单")
        choice = input("请选择操作: ")

        if choice == '1':
            confirm = input("喵呜！警告：此操作将重置所有题目的对错统计并删除错题簿，数据无法恢复！\n确定要清理吗？ (输入 'yes' 确认，其他任意键取消): ").strip().lower()
            if confirm == 'yes':
                print("正在努力清理数据中...")
                updated_chapters = set()
                for chap_key, questions_list in self.all_questions_data.items():
                    if not questions_list: continue
                    changed_in_chapter = False
                    for question in questions_list:
                        if question.get('correct_count', 0) != 0 or question.get('error_count', 0) != 0:
                            question['correct_count'] = 0
                            question['error_count'] = 0
                            changed_in_chapter = True
                    if changed_in_chapter:
                        updated_chapters.add(chap_key)
                
                for chap_key in updated_chapters:
                    self.save_chapter_data(chap_key)

                self.incorrect_questions = []
                
                if os.path.exists(self.incorrect_questions_file):
                    try:
                        os.remove(self.incorrect_questions_file)
                        print(f"错题簿文件 '{self.incorrect_questions_file}' 已成功删除。")
                    except Exception as e:
                        print(f"喵呜！错误：删除错题簿文件时失败了: {e}")
                else:
                    print("错题簿文件本来就不存在，无需删除哦。")
                
                print("数据清理完成！所有题目的对错次数已重置，错题簿也干干净净啦！")
            else:
                print("取消清理操作。数据安然无恙喵~")
        elif choice == '0':
            return
        else:
            print("无效输入哦，请输入菜单中的数字。")
        
        input("按 Enter键 返回主菜单...")


    def main_menu(self):
        """
        显示主菜单并处理用户选择。
        """
        while True:
            print("\n===============================")
            print("  欢迎使用喵喵学习小助手！ V1.1  ")
            print("===============================")
            print("  主人，今天想做些什么呢？")
            print("  1. 速刷模式 (快速回顾题目和答案)")
            print("  2. 答题模式 (模拟答题并记录对错)")
            print("  3. 错题回顾 (复习答错的题目)")
            print("  4. 控制模式 (管理数据)")
            print("  0. 退出程序")
            print("-------------------------------")

            choice = input("请输入模式对应的数字: ").strip()

            if choice == '1':
                self.quick_review_mode()
            elif choice == '2':
                self.quiz_mode()
            elif choice == '3':
                self.incorrect_questions_mode()
            elif choice == '4':
                self.control_mode()
            elif choice == '0':
                print("\n喵~ 感谢主人的使用，期待下次再见！ (づ｡◕‿‿◕｡)づ ~ByeBye~")
                break
            else:
                print("喵呜~ 输入好像有点不对哦，请输入菜单中显示的数字呀！")
            
            # time.sleep(0.5) # 短暂延迟，让用户看清界面切换

if __name__ == "__main__":
    controller = MainController() 
    controller.main_menu()