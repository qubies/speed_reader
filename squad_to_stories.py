import json
questions = []

count =0
max_line_len = 40
with open("./squad.json") as f:
    o = json.load(f)["data"]
for topic in o:
    for para in topic["paragraphs"]:
        for k,v in para.items():
            pc = para["context"].split()
            l = len(pc)
            if l > 227 and l < 229:
                lines = []
                current_len = 0
                start = 0
                for i, word in enumerate(pc):
                    current_len += len(word)+1 # doesnt perfectly count spaces... but meh
                    if current_len > max_line_len:
                        current_len = 0
                        lines.append(pc[start:i+1])
                        start=i+1
                if start < l:
                    lines.append(pc[start:])


                questions.append({"Text": lines, "Spans":[[]], "Name":topic["title"], "Questions": []})
                count +=1
                for question in para["qas"]:
                    try:
                        #  print(f"Q: {question['question']} A: {question['answers'][0]['text']}")
                        questions[-1]["Questions"].append({"Text":str(question['question']), 
                            "Answer":0, 
                            "Choices":[str(question['answers'][0]['text']).capitalize(), "wrong_1", "wrong_2", "wrong_3"]})
                    except:
                        continue
with open("./filtered_qs.json", "w+") as f:
    json.dump(questions, f, indent=4)
