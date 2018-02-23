all: windows mac

windows:
	env GOOS=windows GOARCH=amd64 go build -o bms2json_windows bms2json.go

mac:
	env GOOS=darwin GOARCH=amd64 go build -o bms2json_mac bms2json.go

clean:
	rm bms2json_windows bms2json_mac
