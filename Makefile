OUTBIN?=bin
GO ?= go

$(OUTBIN):
	mkdir -p ./${OUTBIN}

.PHONY: setup
setup: $(OUTBIN) ## setup go modules
	go mod tidy

.PHONY: build
build: setup ## build the binary
	$(GO) build -o $(OUTBIN)/main ./...
