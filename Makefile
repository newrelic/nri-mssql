WORKDIR         := $(shell pwd)
INTEGRATION     := mssql
BINARY_NAME      = nri-$(INTEGRATION)
GO_FILES        := ./src/
GOFLAGS          = -mod=readonly
GO_VERSION		?= $(shell grep '^go ' go.mod | awk '{print $$2}')
BUILDER_IMAGE 	?= "ghcr.io/newrelic/coreint-automation:latest-go$(GO_VERSION)-ubuntu16.04"

all: build

build: clean test compile

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: Removing binaries and coverage file..."
	@rm -rfv bin coverage.xml

test:
	@echo "=== $(INTEGRATION) === [ test ]: Running unit tests..."
	@go test -race ./... -count=1

integration-test:
	@echo "=== $(INTEGRATION) === [ test ]: running integration tests..."
	@docker compose -f tests/docker-compose.yml up -d
	@sleep 120
	@go test -v -tags=integration -count 1 ./tests/mssql_test.go -timeout 180s || (ret=$$?; docker compose -f tests/docker-compose.yml down -v && exit $$ret)
	@docker compose -f tests/docker-compose.yml down -v

compile: 
	@echo "=== $(INTEGRATION) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) $(GO_FILES)

# rt-update-changelog runs the release-toolkit run.sh script by piping it into bash to update the CHANGELOG.md.
# It also passes down to the script all the flags added to the make target. To check all the accepted flags,
# see: https://github.com/newrelic/release-toolkit/blob/main/contrib/ohi-release-notes/run.sh
#  e.g. `make rt-update-changelog -- -v`
rt-update-changelog:
	curl "https://raw.githubusercontent.com/newrelic/release-toolkit/v1/contrib/ohi-release-notes/run.sh" | bash -s -- $(filter-out $@,$(MAKECMDGOALS))

# Include thematic Makefiles
include $(CURDIR)/build/ci.mk
include $(CURDIR)/build/release.mk

.PHONY: all build clean compile test rt-update-changelog
