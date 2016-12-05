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
        wav = ""
        while head != -1:
            start = head+len(key)+3
            end = bms.find("\n", head)
            wav += bms[start:end] + ","
            search_key = "#{}".format(key)
            head = bms.find(search_key, head+1)
            if head == -1:
                head = bms.find(search_key.upper())
        return wav
    start = head+len(key)+1
    end = bms.find("\n", head)
    ret = bms[start:end]
    if is_int is True:
        ret = int(ret)
    return ret


def slice_two(data, digit=10):
    num = []
    for i in range(0, len(data), 2):
        num.append(int(data[i:i+2], digit))
    return num


def read_main(bms):
    head = bms.find("MAIN DATA FIELD")
    measure = 0
    main_data = []
    while head != -1:
        for i in range(11, 14):
            head = bms.find("#", head+1)
            if head == -1:
                break
            lane = int(bms[head+4:head+4+2])
            if lane not in range(11, 14):
                continue
            if int(bms[head+1:head+1+3]) != measure or lane != i:
                head = head - 1
                continue
            slice_start = bms.find(":", head) + 1
            slice_end = bms.find("\n", head)
            data = slice_two(bms[slice_start:slice_end])
            main_object = {
                "line": measure,
                "channel": lane-11,
                "data": data
            }
            main_data.append(main_object)
        measure += 1
    return main_data


def read_start(bms):
    head = bms.find("MAIN DATA FIELD")
    while head != -1:
        head = bms.find("#", head+1)
        if int(bms[head+4:head+6]) == 1:
            return int(bms[head+1:head+4])


def read_bpmchange(bms):
    bpmchange = []
    head = bms.find("MAIN DATA FIELD")
    while head != -1:
        head = bms.find("#", head+1)
        if head == -1:
            break
        if int(bms[head+4:head+6]) == 3:
            line = int(bms[head+1:head+3])
            index = bms.find(":", head)
            slice_start = index+1
            slice_end = bms.find("\n", index)
            value = slice_two(bms[slice_start:slice_end], 16)
            bpmchange.append({
                "line": line,
                "value": value
            })
    return bpmchange


def read_bms(filename):
    header_string_list = [
        "genre",
        "title",
        "artist",
        "wav"
    ]
    header_integer_list = [
        "bpm",
        "playlevel",
        "rank"
    ]

    header = {}
    main = []
    start = 0
    bpm = []

    bms = open(filename).read()
    for key in header_string_list:
        header[key] = read_header(bms, key, False)
    for key in header_integer_list:
        header[key] = read_header(bms, key, True)
    main = read_main(bms)
    start = read_start(bms)
    bpm = read_bpmchange(bms)

    json_object = {
        "header": header,
        "main": main,
        "start": start,
        "bpm": bpm
    }
    return json.dumps(json_object, ensure_ascii=False)


if __name__ == "__main__":
    PATH = sys.argv[1]
    ROOT, EXT = os.path.splitext(PATH)
    FILES = []
    if os.path.isdir(PATH):
        FILES = os.listdir(PATH)
    else:
        FILES.append(PATH)
    for f in FILES:
        if ".bms" not in f and ".bme" not in f:
            continue
        print("Convert:{}".format(f))
        jsonData = read_bms(f)
        root, _ = os.path.splitext(f)
        output = open(root + ".json", 'w')
        output.write(jsonData)
        output.close()
