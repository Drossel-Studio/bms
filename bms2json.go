package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	targetPath := "."
	outputPath := ""
	totalCount := 0
	successCount := 0
	failureCount := 0
	if len(os.Args) > 1 {
		targetPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		outputPath = os.Args[2]
	}

	for _, f := range findAllFiles(targetPath) {
		if !strings.Contains(f, ".bms") && !strings.Contains(f, ".bme") {
			continue
		}
		totalCount++
		fmt.Println("Convert: " + f)
		exportPath, result := convert(f, outputPath)
		if result {
			successCount++
		} else {
			failureCount++
		}
		if len(exportPath) > 0 {
			fmt.Println("Export: " + exportPath)
		}
		fmt.Println()
	}
	fmt.Println("===SUMMARY===")
	fmt.Println("TOTAL: " + strconv.Itoa(totalCount))
	fmt.Println("SUCCESS: \033[32m" + strconv.Itoa(successCount) + "\033[0m")
	fmt.Println("FAILURE: \033[31m" + strconv.Itoa(failureCount) + "\033[0m")
}

func findAllFiles(directory string) []string {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		panic(err)
	}
	var paths []string
	for _, file := range files {
		if file.IsDir() {
			paths = append(paths, findAllFiles(filepath.Join(directory, file.Name()))...)
			continue
		}
		paths = append(paths, filepath.Join(directory, file.Name()))
	}
	return paths
}

func convert(f string, outputPath string) (string, bool) {
	exportPath := ""
	result := false
	jsondata, err := readBms(f)
	if err != nil {
		fmt.Println("error:", err)
		return exportPath, false
	}
	path, filename := filepath.Split(f)
	if outputPath == "" {
		path = "./json"
	} else {
		path = outputPath
	}
	pos := strings.LastIndex(filename, ".")
	root := filename[:pos]
	ioutil.WriteFile(path+"/"+root+".json", jsondata, os.ModePerm)
	result = true
	return exportPath, result
}

func readBms(filename string) ([]byte, error) {
	//headerStringList := []string{"genre", "title", "artist", "wav"}
	headerStringList := []string{} //不要なので空にする
	//headerIntegerList := []string{"bpm", "playlevel", "rank"}
	headerIntegerList := []string{"bpm"} //必要なものだけ残した

	header := map[string]interface{}{}
	filedata, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	bms := string(filedata)
	for _, key := range headerStringList {
		header[key] = readHeader(bms, key)
	}
	for _, key := range headerIntegerList {
		header[key] = 0
		header[key], err = strconv.Atoi(readHeader(bms, key))
		if err != nil {
			panic(err)
		}
	}
	mainData := readMain(bms)
	initialBpm := header["bpm"].(int)
	start := readStart(bms, initialBpm)
	bpm := readBpmchange(bms)
	//fmt.Println(bpm)
	noteWeights := calcNoteWeights(bms)
	//for name, weight := range noteWeights {
	//	fmt.Print(name)
	//	fmt.Println(": " + strconv.FormatFloat(weight, 'f', 3, 64))
	//}
	jsonObject := map[string]interface{}{
		"header":       header,
		"main":         mainData,
		"start":        start,
		"bpm":          bpm, // デフォルト値の調整のため、一旦空配列を定義
		"notes_weight": noteWeights,
	}
	jsonBytes, err := json.Marshal(jsonObject)
	if err != nil {
		fmt.Println("error:", err)
		return nil, err
	}
	hasher := md5.New()
	hasher.Write(jsonBytes)
	objectHash := hex.EncodeToString(hasher.Sum(nil))
	fmt.Println("Hash: " + objectHash)
	jsonObject["hash"] = objectHash
	bytes, err := json.Marshal(jsonObject)
	if err != nil {
		fmt.Println("error:", err)
		return nil, err
	}
	return bytes, nil
}

func readHeader(bms string, key string) string {
	head := search(bms, 0, key)
	if head == -1 {
		head = search(bms, 0, strings.ToUpper(key))
	}
	if head == -1 {
		return "NONE"
	}
	if key == "WAV" {
		return getWav(bms, key, head)[0]
	}
	start := head + len(key) + 1
	end := search(bms, head, "\n")
	ret := bms[start:end]
	ret = strings.TrimSpace(ret)
	return ret
}

func getWav(bms string, key string, head int) []string {
	var wav []string
	for head != -1 {
		start := head + len(key) + 3
		end := search(bms, head, "\n")
		wav = append(wav, bms[start:end]+",")
		searchKey := "#" + key
		head = search(bms, head+1, searchKey)
		if head == -1 {
			head = search(bms, 0, strings.ToUpper(searchKey))
		}
	}
	return wav
}

func readMain(bms string) []map[string]interface{} {
	head := search(bms, 0, "MAIN DATA FIELD")
	measure := 0
	var mainData []map[string]interface{}
	for head != -1 {
		//print("MAIN")
		i := 11
		for i < 14 {
			//print(i)
			head = search(bms, head+1, "#")
			if head == -1 {
				break
			}
			//print(head)
			//print("\n")
			lane, err := strconv.Atoi(bms[head+4 : head+4+2])
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			if lane < 11 || lane > 13 {
				//print("NOT LANE")
				continue
			}
			m, err := strconv.Atoi(bms[head+1 : head+1+3])
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			//print(m)
			//print("\n")
			//print(m != measure)
			//print(measure)
			//print("\n")
			//print(lane, i)
			//print("\n")
			//print(lane != i)
			//print("\n")
			if m != measure || lane != i {
				head = head - 1
				i++
				continue
			}
			sliceStart := search(bms, head, ":") + 1
			sliceEnd := search(bms, head, "\n") - 1
			data := sliceTwo(bms[sliceStart:sliceEnd], 10)
			mainData = append(mainData, map[string]interface{}{"line": measure, "channel": lane - 11, "data": data})
			i++
		}
		measure++
	}
	return mainData
}

func search(str string, start int, target string) int {
	i := strings.Index(str[start:], target)
	if i == -1 {
		return -1
	}
	return i + start
}

func sliceTwo(data string, digit int) []int {
	var num []int
	data = strings.Trim(data, "\r")
	for i := 0; i < len(data); i += 2 {
		//fmt.Println(data)
		//fmt.Println([]byte(data))
		//fmt.Println(len(data))
		//fmt.Println(i)
		numText := data[i : i+2]
		if n, err := strconv.ParseInt(numText, digit, 64); err == nil {
			num = append(num, int(n))
		}
	}
	return num
}

func readStart(bms string, initialBpm int) int {
	head := search(bms, 0, "MAIN DATA FIELD")
	for head != -1 {
		head := search(bms, head+1, "#")
		if val, err := strconv.Atoi(bms[head+4 : head+6]); err != nil || val != 1 {
			continue
		}
		line, err := strconv.Atoi(bms[head+1 : head+4])
		if err != nil {
			continue
		}
		sliceStart := head + 7
		sliceEnd := search(bms, head, "\n")
		data := sliceTwo(bms[sliceStart:sliceEnd], 10)
		// 1小節の秒数
		oneLineTime := 60.0 / float64(initialBpm) * 4
		fmt.Println(oneLineTime)
		beforeLineTime := oneLineTime * float64(line)
		i := index(data, 1)
		if i == -1 {
			continue
		}
		currentLineTime := oneLineTime * float64(index(data, 1)) / float64(len(data))
		return int((beforeLineTime + currentLineTime) * 1000)
	}
	panic(fmt.Errorf("startコマンドが存在しません"))
}

func index(ary []int, target int) int {
	for i, val := range ary {
		if val == target {
			return i
		}
	}
	return -1
}

func readBpmchange(bms string) []map[string]interface{} {
	var bpmchange []map[string]interface{}
	bpmchange = make([]map[string]interface{}, 0) // 空配列を扱えるように代入する
	head := search(bms, 0, "MAIN DATA FIELD")
	for head != -1 {
		head = search(bms, head+1, "#")
		if head == -1 {
			break
		}
		if channel, err := strconv.Atoi(bms[head+4 : head+6]); err != nil && channel == 3 {
			line, _ := strconv.Atoi(bms[head+1 : head+4])
			index := search(bms, head, ":")
			sliceStart := index + 1
			sliceEnd := search(bms, index, "\n")
			data := sliceTwo(bms[sliceStart:sliceEnd], 16)
			bpmchange = append(bpmchange, map[string]interface{}{"line": line, "data": data})
		}
	}
	return bpmchange
}

func calcNoteWeights(bms string) map[string]float64 {
	head := search(bms, 0, "MAIN DATA FIELD")
	// notesnum[i] 添え字が実際のbmsファイルのノーツ番号と対応
	notesnum := []float64{0, 0, 0, 0, 0, 0, 0, 0}
	for head != -1 {
		head = search(bms, head+1, "#")
		if head == -1 {
			break
		}
		lane, err := strconv.Atoi(bms[head+4 : head+4+2])
		if err != nil || lane < 11 || lane > 13 {
			continue
		}
		sliceStart := search(bms, head, ":") + 1
		sliceEnd := search(bms, head, "\n")
		data := sliceTwo(bms[sliceStart:sliceEnd], 10)
		for _, notes := range data {
			if notes == 0 {
				continue
			}
			notesnum[notes]++
		}
	}
	var notessum float64
	for _, nn := range notesnum {
		notessum += nn
	}
	noteType := map[string]int{
		"normal":  2,
		"red":     3,
		"long":    4,
		"slide":   6,
		"special": 7,
	}
	fmt.Println("---notesrate-------------")
	for k, v := range noteType {
		printNoteRate(k, notesnum[v], notessum)
	}
	fmt.Println("-------------------------")

	noteWeights := map[string]float64{
		"normal":  1,
		"each":    2,
		"long":    2,
		"slide":   0.5,
		"special": 5,
	}
	if (notesnum[5] + notesnum[6]) == 0 {
		return noteWeights
	}

	slideWeight := (noteWeights["normal"]*notesnum[2] + noteWeights["each"]*notesnum[3]*0.6) / (notesnum[5] + notesnum[6])
	if slideWeight < 0.5 {
		noteWeights["slide"] = round(slideWeight, 3)
		print("slide_weight is corrected")
	}
	return noteWeights
}

func printNoteRate(name string, sum float64, allsum float64) {
	rate := sum / allsum * 100.0
	fmt.Println(name + ": " + strconv.FormatFloat(sum, 'f', 0, 64) + " (" + strconv.FormatFloat(rate, 'f', 1, 64) + "%)")
}

func round(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Floor(f*shift+.5) / shift
}

/*
def getWav(bms, key, head):
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


def read_header(bms, key, is_int):
    head = bms.find(key)
    if head == -1:
        head = bms.find(key.upper())
    if head == -1:
        return "NONE"
    if key == "WAV":
        return getWav(bms, key, head)
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
        #print("MAIN")
        i = 11
        while i < 14:
            #print(i)
            head = bms.find("#", head + 1)
            if head == -1:
                break
            #print(head)
            lane = int(bms[head + 4:head + 4 + 2])
            if lane not in range(11, 14):
                #print("NOT LANE")
                continue
            #print(int(bms[head + 1:head + 1 + 3]) != measure)
            #print(lane, i)
            #print(lane != i)
            if int(bms[head + 1:head + 1 + 3]) != measure or lane != i:
                head = head - 1
                i += 1
                continue
            slice_start = bms.find(":", head) + 1
            slice_end = bms.find("\n", head)
            data = slice_two(bms[slice_start:slice_end])
            main_object = {"line": measure, "channel": lane - 11, "data": data}
            main_data.append(main_object)
            i += 1
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
    objectHash = hashlib.md5(str(json_object).encode('utf-8')).hexdigest()
    print(f"Hash: {objectHash}")
    json_object["hash"] = objectHash
    return json.dumps(json_object, ensure_ascii=False)


def find_all_files(directory):
    for root, _, files in os.walk(directory):
        yield root
        for file in files:
            yield os.path.join(root, file)


def convert(f, outputPath=None):
    exportPath = ""
    result = False
    try:
        jsondata = read_bms(f)
        path, filename = os.path.split(f)
        if outputPath is None:
            path = "./json"
        else:
            path = outputPath
        root, _ = os.path.splitext(filename)
        exportPath = os.path.join(path, root + ".json")
        output = open(exportPath, 'w')
        output.write(jsondata)
        output.close()
        result = True
    except UnicodeDecodeError:
        print(f"\033[31mError: 譜面ファイルのエンコードがutf-8ではありません\033[0m")
    except Exception:
        print(f"\033[31mError: {sys.exc_info()[0]}\033[0m")
    return exportPath, result


if __name__ == "__main__":
    PATH = "."
    OUTPUT = None
    totalCount = 0
    successCount = 0
    failureCount = 0
    if len(sys.argv) > 1:
        PATH = sys.argv[1]
    if len(sys.argv) > 2:
        OUTPUT = sys.argv[2]

    for f in find_all_files(PATH):
        if ".bms" not in f and ".bme" not in f:
            continue
        totalCount += 1
        print(f"Convert: {f}")
        exportPath, result = convert(f, OUTPUT)
        if result:
            successCount += 1
        else:
            failureCount += 1
        if len(exportPath) > 0:
            print(f"Export: {exportPath}")
        print()
    print("===SUMMARY===")
    print(f"TOTAL: {totalCount}")
    print(f"SUCCESS: \033[32m{successCount}\033[0m")
    print(f"FAILURE: \033[31m{failureCount}\033[0m")
*/
