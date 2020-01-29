SHELL := /bin/bash
KURL_UTIL_IMAGE := replicated/kurl-util:latest

export GO111MODULE=on

GIT_TREE = $(shell git rev-parse --is-inside-work-tree 2>/dev/null)
ifneq "$(GIT_TREE)" ""
define GIT_UPDATE_INDEX_CMD
git update-index --assume-unchanged
endef
define GIT_SHA
`git rev-parse HEAD`
endef
else
define GIT_UPDATE_INDEX_CMD
echo "Not a git repo, skipping git update-index"
endef
define GIT_SHA
""
endef
endif

clean:
	rm -rf ./bin

test:
	go test ./cmd/...

build: bin/join bin/yamlutil

bin/join:
	go build -o bin/join cmd/join/main.go

bin/yamlutil:
	go build -o bin/yamlutil cmd/yamlutil/main.go

.PHONY: kurl-util-image
kurl-util-image:
	docker build -t $(KURL_UTIL_IMAGE) -f deploy/Dockerfile --build-arg commit="${GIT_SHA}" .
