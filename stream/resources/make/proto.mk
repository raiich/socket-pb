PROTOC_URL := https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protoc-29.3-osx-aarch_64.zip
BUF_VERSION := v1.49.0

LOCAL := $(CURDIR)/../.local
BIN    := $(LOCAL)/bin
GO_BIN := $(LOCAL)/go/bin
TEMP   := $(LOCAL)/temp

GO_INSTALL := GOBIN=$(GO_BIN) go install

PATH := $(BIN):$(GO_BIN):$(PATH)
PROTOC := $(BIN)/protoc
PROTOC_DEP := $(PROTOC) \
  $(GO_BIN)/protoc-gen-go
BUF := $(GO_BIN)/buf

GO_FILES := $(shell find . -path "./generated" -prune -or -name \*.go -print)
PROTO := $(shell find resources/proto -name \*.proto)

generated: $(PROTO) $(PROTOC_DEP)
	mkdir -p generated
	touch generated
	mkdir -p generated/go
	$(PROTOC) -I=resources/proto \
	  --go_out=generated/go --go_opt=paths=source_relative \
	  $(PROTO)

.PHONY: format-proto
format-proto: $(BUF)
	$(BUF) format resources/proto -w

$(BUF):
	mkdir -p $(GO_BIN)
	$(GO_INSTALL) github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)

$(PROTOC):
	mkdir -p $(TEMP)
	curl -sSL -o $(TEMP)/protoc.zip "$(PROTOC_URL)"
	unzip -q $(TEMP)/protoc.zip -d $(LOCAL)

$(GO_BIN)/protoc-gen-go:
	mkdir -p $(GO_BIN)
	$(GO_INSTALL) google.golang.org/protobuf/cmd/protoc-gen-go@latest
