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

.PHONY: kurl-util-image
kurl-util-image:
	docker build -t $(KURL_UTIL_IMAGE) -f deploy/Dockerfile --build-arg commit="${GIT_SHA}" .

.PHONY: clean
clean:
	rm -rf ./bin

.PHONY: deps
deps:
	go get -u golang.org/x/lint/golint
	go get -u github.com/sqs/goreturns

.PHONY: lint
lint:
	golint -set_exit_status ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test: lint vet
	go test ./cmd/...

.PHONY: build
build: bin/join bin/yamlutil

bin/join:
	go build -o bin/join cmd/join/main.go

bin/yamlutil:
	go build -o bin/yamlutil cmd/yamlutil/main.go

.PHONY: fmt
fmt:
	go fmt ./...
	goreturns -w .
