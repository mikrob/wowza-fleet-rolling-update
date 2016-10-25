BINARY=wowza_rolling_update

PHONY: all

test:
	go test  -v ./...

all:
	go build -o ${BINARY} wowza.go
	sudo cp ${BINARY} /usr/local/bin/
