ROOT_DIR := $(CURDIR)
BUILD_DIR := $(ROOT_DIR)/_output
BIN_DIR := $(BUILD_DIR)/bin
BUILD_SCRIPT := $(ROOT_DIR)/build/build.sh
ARCH ?= amd64
REGISTRY ?= "registry.cn-beijing.aliyuncs.com/yunionio"
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
		git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)

ifneq ($(DLV),)
	GO_BUILD_FLAGS += -gcflags "all=-N -l"
endif
GO_BUILD_FLAGS+=-mod vendor

export GO111MODULE=on

cmdTargets:=$(filter-out ,$(wildcard cmd/*))
rpmTargets:=$(foreach b,$(patsubst cmd/%,%,$(cmdTargets)),$(if $(shell [ -f "$(CURDIR)/build/$(b)/vars" ] && echo 1),rpm/$(b)))

all:
	$(MAKE) $(cmdTargets)
#.PHONY: cmd/*

fmt:

cmd/%: fmt
	go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(shell basename $@) yunion.io/x/notify-plugins/$@

image:
	REGISTRY=${REGISTRY} TAG=${VERSION} ARCH=${ARCH} ${ROOT_DIR}/scripts/docker_push.sh

image-push:
	PUSH=true REGISTRY=${REGISTRY} TAG=${VERSION} ARCH=${ARCH} ${ROOT_DIR}/scripts/docker_push.sh

rpm/%: cmd/%
	$(BUILD_SCRIPT) $*

rpm:
	$(MAKE) $(rpmTargets)

rpmclean:
	rm -fr $(BUILD_DIR)/rpms

ONECLOUD_RELEASE_BRANCH:=release/3.10
GOPROXY ?= direct

mod:
	GOPROXY=$(GOPROXY) GOSUMDB=off go get yunion.io/x/onecloud@$(ONECLOUD_RELEASE_BRANCH)
	GOPROXY=$(GOPROXY) GOSUMDB=off go get -d $(patsubst %,%@master,$(shell GO111MODULE=on go mod edit -print  | sed -n -e 's|.*\(yunion.io/x/[a-z].*\) v.*|\1|p' | grep -v '/onecloud$$'))
	go mod tidy -v
	go mod vendor -v
