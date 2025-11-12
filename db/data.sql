-- サンプルデータの挿入

-- ジャンルの挿入
INSERT INTO genres (name, display_name) VALUES
('grammar', '文法'),
('vocabulary', '語彙'),
('reading', '読解'),
('listening', 'リスニング'),
('writing', 'ライティング');

-- 問題の挿入（30件以上）
INSERT INTO problems (genre_id, text, answer_file_path) VALUES
(1, 'Choose the correct verb form: She ___ to the store yesterday.', '/answers/problem_001.txt'),
(1, 'Fill in the blank: They ___ playing soccer when it started to rain.', '/answers/problem_002.txt'),
(2, 'What is the synonym of "happy"?', '/answers/problem_003.txt'),
(2, 'Choose the correct word: The weather is very ___ today.', '/answers/problem_004.txt'),
(3, 'Read the passage and answer: What is the main idea?', '/answers/problem_005.txt'),
(3, 'According to the text, when did the event occur?', '/answers/problem_006.txt'),
(4, 'Listen to the audio and choose the correct answer.', '/answers/problem_007.txt'),
(4, 'What did the speaker mention about the schedule?', '/answers/problem_008.txt'),
(5, 'Write a short paragraph about your hometown.', '/answers/problem_009.txt'),
(5, 'Compose an email requesting information.', '/answers/problem_010.txt'),
(1, 'Select the appropriate preposition: She arrived ___ the airport.', '/answers/problem_011.txt'),
(1, 'Identify the error in this sentence.', '/answers/problem_012.txt'),
(2, 'What is the antonym of "difficult"?', '/answers/problem_013.txt'),
(2, 'Choose the word that best fits: The movie was ___', '/answers/problem_014.txt'),
(3, 'What can be inferred from the passage?', '/answers/problem_015.txt'),
(3, 'Find the supporting detail for the main argument.', '/answers/problem_016.txt'),
(4, 'Based on the conversation, what will happen next?', '/answers/problem_017.txt'),
(4, 'Listen and identify the speakers relationship.', '/answers/problem_018.txt'),
(5, 'Write a response to the given situation.', '/answers/problem_019.txt'),
(5, 'Create a descriptive paragraph about a place.', '/answers/problem_020.txt'),
(1, 'Which sentence is grammatically correct?', '/answers/problem_021.txt'),
(1, 'Rewrite the sentence in passive voice.', '/answers/problem_022.txt'),
(2, 'Select the word with similar meaning to "important".', '/answers/problem_023.txt'),
(2, 'Complete the sentence with the correct vocabulary.', '/answers/problem_024.txt'),
(3, 'What is the authors purpose in writing this?', '/answers/problem_025.txt'),
(3, 'Identify the tone of the passage.', '/answers/problem_026.txt'),
(4, 'What time was mentioned in the recording?', '/answers/problem_027.txt'),
(4, 'Choose the statement that matches the audio.', '/answers/problem_028.txt'),
(5, 'Write an opinion essay on the given topic.', '/answers/problem_029.txt'),
(5, 'Draft a letter of complaint.', '/answers/problem_030.txt'),
(1, 'Use the correct conditional form in the sentence.', '/answers/problem_031.txt'),
(2, 'Find the word that does not belong in the group.', '/answers/problem_032.txt'),
(3, 'Summarize the main points of the article.', '/answers/problem_033.txt'),
(4, 'Listen and answer the comprehension questions.', '/answers/problem_034.txt'),
(5, 'Write a narrative about a memorable experience.', '/answers/problem_035.txt');

-- ユーザーの挿入
INSERT INTO users (name) VALUES
('TEST');

-- 結果の挿入
INSERT INTO results (user_id, problem_id, answered_text, score, try_file_path) VALUES
(1, 1, 'AAA', 100, '/test_path');