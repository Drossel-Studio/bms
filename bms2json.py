#!/usr/bin/env python
# -*- coding: UTF-8 -*-

import json
import os.path
import sys


def read_header(bms, key, is_int):
    head = bms.find(key)
    if head == -1:
        head = bms.find(key.upper())
    if head == -1:
        return "NONE"
    if key == "WAV":
        wav = []
        while head != -1:
            start = head + len(key) + 3
            end = bms.find("\n", head)
            wav += bms[start:end] + ","
            search_key = "#{}".format(key)
            head = bms.find(search_key, head + 1)
            if head == -1:
                head = bms.find(search_key.upper())
        return wav
    start = head + len(key) + 1
    end = bms.find("\n", head)
    ret = bms[start:end]
    if is_int is True:
        ret = int(ret)
    return ret


def slice_two(data, digit=10):
    num = []
    for i in range(0, len(data), 2):
        num_text = data[i:i + 2]
        if num_text.isdigit():
            num.append(int(num_text, digit))
    return num


def read_main(bms):
    head = bms.find("MAIN DATA FIELD")
    measure = 0
    main_data = []
    while head != -1:
        for i in range(11, 14):
            head = bms.find("#", head + 1)
            if head == -1:
                break
            lane = int(bms[head + 4:head + 4 + 2])
            if lane not in range(11, 14):
                continue
            if int(bms[head + 1:head + 1 + 3]) != measure or lane != i:
                head = head - 1
                continue
            slice_start = bms.find(":", head) + 1
            slice_end = bms.find("\n", head)
            data = slice_two(bms[slice_start:slice_end])
            main_object = {"line": measure, "channel": lane - 11, "data": data}
            main_data.append(main_object)
        measure += 1
    return main_data


def read_start(bms, initialBpm):
    if initialBpm is None:
        print("Error: BPMが不正です")
        exit(1)
    head = bms.find("MAIN DATA FIELD")
    while head != -1:
        head = bms.find("#", head + 1)
        if int(bms[head + 4:head + 6]) != 1:
            continue
        line = int(bms[head + 1:head + 4])
        slice_start = head + 7
        slice_end = bms.find("\n", head)
        data = slice_two(bms[slice_start:slice_end], 10)
        # 1小節の秒数
        one_line_time = 60.0 / initialBpm * 4
        before_line_time = one_line_time * line
        current_line_time = one_line_time * data.index(1) / len(data)
        return int((before_line_time + current_line_time) * 1000)


def read_bpmchange(bms):
    bpmchange = []
    head = bms.find("MAIN DATA FIELD")
    while head != -1:
        head = bms.find("#", head + 1)
        if head == -1:
            break
        if int(bms[head + 4:head + 6]) == 3:
            line = int(bms[head + 1:head + 4])
            index = bms.find(":", head)
            slice_start = index + 1
            slice_end = bms.find("\n", index)
            data = slice_two(bms[slice_start:slice_end], 16)
            bpmchange.append({"line": line, "data": data})
    return bpmchange


def printNoteRate(name, sum, allsum):
    rate = float(sum) / allsum * 100.0
    print(f"{name:<8}: {sum:>3} ({rate:.1f}%)")


def calc_notes_weight(bms):
    head = bms.find("MAIN DATA FIELD")
    # notesnum[i] 添え字が実際のbmsファイルのノーツ番号と対応
    notesnum = [0, 0, 0, 0, 0, 0, 0, 0]
    while head != -1:
        head = bms.find("#", head + 1)
        if head == -1:
            break
        lane = int(bms[head + 4:head + 4 + 2])
        if lane not in range(11, 14):
            continue
        slice_start = bms.find(":", head) + 1
        slice_end = bms.find("\n", head)
        data = slice_two(bms[slice_start:slice_end])
        for notes in data:
            if notes == 0:
                continue
            notesnum[notes] += 1

    notessum = sum(notesnum)
    noteType = {
        "normal": 2,
        "red": 3,
        "long": 4,
        "slide": 6,
        "special": 7
    }
    print("---notesrate-------------")
    for k, v in noteType.items():
        printNoteRate(k, notesnum[v], notessum)
    print("-------------------------")

    notes_weight = {
        "normal": 1,
        "each": 2,
        "long": 2,
        "slide": 0.5,
        "special": 5
    }
    if (notesnum[5] + notesnum[6]) == 0:
        return notes_weight

    slide_weight = (notes_weight["normal"] * notesnum[2] + notes_weight["each"]
                    * notesnum[3] * 0.6) / (notesnum[5] + notesnum[6])
    if slide_weight < 0.5:
        notes_weight[3] = round(slide_weight, 3)
        print("slide_weight is corrected")
    return notes_weight


def read_bms(filename):
    header_string_list = ["genre", "title", "artist", "wav"]
    header_integer_list = ["bpm", "playlevel", "rank"]

    header = {}
    main = []
    start = 0
    bpm = []
    notes_weight = {}
    bms = open(filename).read()
    for key in header_string_list:
        header[key] = read_header(bms, key, False)
    for key in header_integer_list:
        header[key] = read_header(bms, key, True)
    main = read_main(bms)
    start = read_start(bms, header["bpm"])
    bpm = read_bpmchange(bms)
    notes_weight = calc_notes_weight(bms)
    print(notes_weight)
    json_object = {
        "header": header,
        "main": main,
        "start": start,
        "bpm": bpm,
        "notes_weight": notes_weight
    }
    return json.dumps(json_object, ensure_ascii=False)


def find_all_files(directory):
    for root, _, files in os.walk(directory):
        yield root
        for file in files:
            yield os.path.join(root, file)


def convert(f):
    exportPath = ""
    try:
        jsondata = read_bms(f)
        path, filename = os.path.split(f)
        root, _ = os.path.splitext(filename)
        exportPath = os.path.join(path, root + ".json")
        output = open(exportPath, 'w')
        output.write(jsondata)
        output.close()
    except Exception:
        print("Error:", sys.exc_info()[0])
    return exportPath


if __name__ == "__main__":
    PATH = ""
    if len(sys.argv) > 1:
        PATH = sys.argv[1]
    else:
        print("Error:引数にフォルダパスを指定してください")
        exit(0)

    for f in find_all_files(PATH):
        if ".bms" not in f and ".bme" not in f:
            continue
        print("Convert:%s" % f)
        exportPath = convert(f)
        if len(exportPath) > 0:
            print("Export:%s" % exportPath)
        print()
