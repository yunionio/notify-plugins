ROOT_DIR := $(CURDIR)
BUILD_DIR := $(ROOT_DIR)/_output
BIN_DIR := $(BUILD_DIR)/bin

REGISTRY ?= "registry.cn-beijing.aliyuncs.com/yunionio"
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
		git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)

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

image: all
	docker build -f Dockerfile -t $(REGISTRY)/notify-plugins:$(VERSION) .

image-push: image
	docker push $(REGISTRY)/notify-plugins:$(VERSION)
