# Makefile for test tailer command

BINARY=tailer

SRC_FILES=${BINARY}.go

# Builds the project
build:
	go build

# Builds distribution
dist:
	GOOS=darwin GOARCH=arm64 go build -o bin/${BINARY}.darwin-arm64 ${SRC_FILES}
	GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY}.darwin-amd64 ${SRC_FILES}
	GOOS=linux GOARCH=arm64 go build -o bin/${BINARY}.linux-arm64 ${SRC_FILES}
	GOOS=linux GOARCH=amd64 go build -o bin/${BINARY}.linux-amd64 ${SRC_FILES}
	GOOS=windows GOARCH=amd64 go build -o bin/${BINARY}.windows-amd64.exe ${SRC_FILES}

# Cleans our project: deletes binaries
clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

.PHONY: clean