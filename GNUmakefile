# Auto-install https://github.com/makeplus/makes at specific commit:
MAKES := .cache/makes
MAKES-LOCAL := .cache/local
MAKES-COMMIT ?= 654f7c57ca30a2b08cb4aab8bb0c0d509510ad81
$(shell [ -d $(MAKES) ] || ( \
  git clone -q https://github.com/makeplus/makes $(MAKES) && \
  git -C $(MAKES) reset -q --hard $(MAKES-COMMIT)))
ifneq ($(shell git -C $(MAKES) rev-parse HEAD), \
       $(shell git -C $(MAKES) rev-parse $(MAKES-COMMIT)))
$(error $(MAKES) is not at the correct commit: $(MAKES-COMMIT))
endif
include $(MAKES)/init.mk
include $(MAKES)/clean.mk

# Only auto-install go if no go exists or GO-VERSION specified:
ifeq (,$(shell command -v go))
GO-VERSION ?= 1.24.0
endif
GO-VERSION-NEEDED := $(GO-VERSION)

# yaml-test-suite info:
YTS-URL ?= https://github.com/yaml/yaml-test-suite
YTS-TAG ?= data-2022-01-17
YTS-DIR := yts/testdata/$(YTS-TAG)

CLI-BINARY := go-yaml

MAKES-NO-CLEAN := true
MAKES-CLEAN := $(CLI-BINARY)
MAKES-REALCLEAN := $(dir $(YTS-DIR))

# Setup and include go.mk and shell.mk:
GO-FILES := $(shell find -not \( -path ./.cache -prune \) -name '*.go' | sort)
GO-CMDS-SKIP := test fmt vet
ifndef GO-VERSION-NEEDED
GO-NO-DEP-GO := true
endif
GO-CMDS-RULES := true

include $(MAKES)/go.mk

ifdef GO-VERSION-NEEDED
GO-DEPS += $(GO)
else
SHELL-DEPS := $(filter-out $(GO),$(SHELL-DEPS))
endif

SHELL-NAME := makes go-yaml
include $(MAKES)/clean.mk
include $(MAKES)/shell.mk

MAKES-CLEAN := $(dir $(YTS-DIR))

v ?=
count ?= 1


# Test rules:
test: $(GO-DEPS)
	go test$(if $v, -v)

test-data: $(YTS-DIR)

test-all: test test-yts-all

test-yts: $(GO-DEPS) $(YTS-DIR)
	go test$(if $v, -v) ./yts -count=$(count)

test-yts-all: $(GO-DEPS) $(YTS-DIR)
	@echo 'Testing yaml-test-suite'
	@RUNALL=1 bash -c "$$yts_pass_fail"

test-yts-fail: $(GO-DEPS) $(YTS-DIR)
	@echo 'Testing yaml-test-suite failures'
	@RUNFAILING=1 bash -c "$$yts_pass_fail"

fmt: golangci-lint
	$< fmt ./...

lint: golangci-lint
	$< run

fumpt: $(GO)
	@go install mvdan.cc/gofumpt@latest
	gofumpt -l -w $(GO-FILES)

vet: $(GO)
	go vet ./...

cli: $(CLI-BINARY)

$(CLI-BINARY): $(GO)
	go build -o $@ ./cmd/$@

# Setup rules:
$(YTS-DIR):
	git clone -q $(YTS-URL) $@
	git -C $@ checkout -q $(YTS-TAG)

define yts_pass_fail
( result=.cache/local/tmp/yts-test-results
  go test ./yts -count=1 -v |
    awk '/     --- (PASS|FAIL): / {print $$2, $$3}' > $$result
  known_count=$$(grep -c '' yts/known-failing-tests)
  pass_count=$$(grep -c '^PASS:' $$result)
  fail_count=$$(grep -c '^FAIL:' $$result)
  echo "PASS: $$pass_count"
  echo "FAIL: $$fail_count (known: $$known_count)"
  if [[ $$RUNFAILING ]] && [[ $$pass_count -gt 0 ]]; then
    echo "ERROR: Found passing tests among expected failures:"
    grep '^PASS:' $$result
    exit 1
  fi
  if [[ $$fail_count != "$$known_count" ]]; then
    echo "ERROR: FAIL count differs from expected value of $$known_count"
    exit 1
  fi
)
endef
export yts_pass_fail

GOLANGCI-LINT-INSTALL := \
  https://github.com/golangci/golangci-lint/raw/main/install.sh

golangci-lint: $(GO-DEPS)
	@[[ -f $$(go env GOPATH)/bin/$@ ]] || \
	  curl -sSfL $(GOLANGCI-LINT-INSTALL) | \
	    sh -s -- -b $$(go env GOPATH)/bin
