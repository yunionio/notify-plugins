
ROOT_DIR := $(CURDIR)
BUILD_DIR := $(ROOT_DIR)/_output
BIN_DIR := $(BUILD_DIR)/bin

ifneq ($(LINUX), )
    export CGO_ENABLED=0
    export GOOS=linux
    export GOARCH=amd64
endif

all: cmd/email cmd/websocket cmd/smsaliyun cmd/dingtalk
#.PHONY: cmd/*

fmt:
cmd/%: fmt
	go build -o $(BIN_DIR)/$(shell basename $@) $(ROOT_DIR)/$@

