# Package configuration
PROJECT = engine
COMMANDS = cmd/srcd
DOCKERFILES ?= cmd/srcd-server/Dockerfile:cli-daemon
PKG_OS ?= darwin linux windows

# Including ci Makefile
CI_REPOSITORY ?= https://github.com/src-d/ci.git
CI_PATH ?= $(shell pwd)/.ci
CI_VERSION ?= v1

TEST_PRUNE_WITH_IMAGE ?= false

GO_TAGS = forceposix

MAKEFILE := $(CI_PATH)/Makefile.main
$(MAKEFILE):
	git clone --quiet --branch $(CI_VERSION) --depth 1 $(CI_REPOSITORY) $(CI_PATH);

-include $(MAKEFILE)

GOTEST_BASE = go test -v -timeout 20m -parallel 1 -count 1 -ldflags "$(LD_FLAGS)"
GOTEST_INTEGRATION = $(GOTEST_BASE) -tags="forceposix integration"
GOTEST_REGRESSION = $(GOTEST_BASE) -tags="forceposix regression"

OS := $(shell uname)

ifeq ($(OS),Darwin)
test-integration-clean:
	$(eval TMPDIR_INTEGRATION_TEST := $(PWD)/integration-test-tmp)
	$(eval GOTEST_INTEGRATION := TMPDIR=$(TMPDIR_INTEGRATION_TEST) $(GOTEST_INTEGRATION))
	rm -rf $(TMPDIR_INTEGRATION_TEST)
	mkdir $(TMPDIR_INTEGRATION_TEST)
else
test-integration-clean:
endif

ifeq ($(OS),Darwin)
test-regression-clean:
	$(eval TMPDIR_REGRESSION_TEST := $(PWD)/regression-test-tmp)
	$(eval GOTEST_REGRESSION := TMPDIR=$(TMPDIR_REGRESSION_TEST) $(GOTEST_REGRESSION))
	rm -rf $(TMPDIR_REGRESSION_TEST)
	mkdir $(TMPDIR_REGRESSION_TEST)
else
test-regression-clean:
endif

test-integration-no-build: test-integration-clean
	TEST_PRUNE_WITH_IMAGE=false $(GOTEST_INTEGRATION) github.com/src-d/engine/cmdtests/
	$(GOTEST_INTEGRATION) github.com/src-d/engine/cmdtests/ -run TestPruneTestSuite/TestRunningContainersWithImages

test-integration: clean build docker-build test-integration-no-build

test-regression-usage:
	@echo
	@echo "Usage: \`PREV_ENGINE_VERSION=<first engine version to compare (default: 'latest')> CURR_ENGINE_VERSION=<second engine version to compare (default: 'local:HEAD')> make test-regression\`"
	@echo "Examples:"
	@echo "- \`make test-regression\`                                                          # tests that latest version is forward-compatible with current (HEAD) version"
	@echo "- \`PREV_ENGINE_VERSION=v0.10.0 make test-regression\`                              # tests that v0.10.0 version is forward-compatible with current (HEAD) version"
	@echo "- \`PREV_ENGINE_VERSION=v0.10.0 CURR_ENGINE_VERSION=v0.11.0 make test-regression\`  # tests that v0.10.0 version is forward-compatible with v0.11.0 version"
	@echo

test-regression: test-regression-usage test-regression-clean
	$(GOTEST_REGRESSION) github.com/src-d/engine/cmdtests/
