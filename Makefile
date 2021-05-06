WORKDIR      := $(shell pwd)
NATIVEOS	 := $(shell go version | awk -F '[ /]' '{print $$4}')
NATIVEARCH	 := $(shell go version | awk -F '[ /]' '{print $$5}')
INTEGRATION  := mssql
BINARY_NAME   = nri-$(INTEGRATION)
GO_FILES     := ./src/
GOTOOLS       = gopkg.in/alecthomas/gometalinter.v2 \
		github.com/axw/gocov/gocov \
		github.com/AlekSi/gocov-xml \

all: build

build: check-version clean validate test compile

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: Removing binaries and coverage file..."
	@rm -rfv bin coverage.xml

tools: check-version
	@echo "=== $(INTEGRATION) === [ tools ]: Installing tools required by the project..."
	@GO111MODULE=off go get $(GOTOOLS)
	@GO111MODULE=off gometalinter.v2 --install

tools-update: check-version
	@echo "=== $(INTEGRATION) === [ tools-update ]: Updating tools required by the project..."
	@GO111MODULE=off go get -u $(GOTOOLS)
	@GO111MODULE=off gometalinter.v2 --install

deps: tools

validate: 
	@echo "=== $(INTEGRATION) === [ validate ]: Validating source code running gometalinter..."
	@GO111MODULE=off gometalinter.v2 --config=.gometalinter.json $(GO_FILES)...

validate-all: 
	@echo "=== $(INTEGRATION) === [ validate ]: Validating source code running gometalinter..."
	@GO111MODULE=off gometalinter.v2 --config=.gometalinter.json --enable=interfacer --enable=gosimple $(GO_FILES)...

compile: 
	@echo "=== $(INTEGRATION) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) $(GO_FILES)

compile-only: 
	@echo "=== $(INTEGRATION) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) $(GO_FILES)

test: deps
	@echo "=== $(INTEGRATION) === [ test ]: Running unit tests..."
	@gocov test -race $(GO_FILES)... | gocov-xml > coverage.xml

# Include thematic Makefiles
include $(CURDIR)/build/ci.mk
include $(CURDIR)/build/release.mk

check-version:
ifdef GOOS
ifneq "$(GOOS)" "$(NATIVEOS)"
	$(error GOOS is not $(NATIVEOS). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif
ifdef GOARCH
ifneq "$(GOARCH)" "$(NATIVEARCH)"
	$(error GOARCH variable is not $(NATIVEARCH). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif

.PHONY: all build clean tools tools-update deps validate compile test check-version
