
ROOT_DIR := $(CURDIR)
BUILD_DIR := $(ROOT_DIR)/_output
BIN_DIR := $(BUILD_DIR)/bin

ifneq ($(LINUX), )
	ENV += GOOS=linux GOARCH=amd64
endif

ifneq ($(DLV),)
	GO_BUILD_FLAGS += -gcflags "all=-N -l"
endif

all: cmd/email cmd/websocket cmd/smsaliyun cmd/dingtalk
#.PHONY: cmd/*

fmt:
cmd/%: fmt
	$(ENV) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(shell basename $@) $(ROOT_DIR)/$@

