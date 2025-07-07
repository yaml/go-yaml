M := .cache/makes
M-COMMIT ?= 683675d4c45f90f40215399ebd319805877999c9
$(shell [ -d $M ] || ( \
  git clone -q https://github.com/makeplus/makes $M && \
  git -C $M reset -q --hard $(M-COMMIT)))
ifneq ($(shell git -C $M rev-parse $(M-COMMIT)),\
       $(shell git -C $M rev-parse HEAD))
$(error $M is not at the correct commit: $(M-COMMIT))
endif

include $M/init.mk
include $M/go.mk
SHELL-NAME := go-yaml
include $M/shell.mk

YTS-NAME := yaml-test-suite
YTS-TAG := data-2022-01-17
YTS-DIR := $(YTS-NAME)/testdata/$(YTS-TAG)
YTS-FILE := 229Q
YTS-DEP := $(YTS-DIR)/$(YTS-FILE)

TEST-DEPS := $(GO) $(YTS-DIR) $(YTS-DEP)


test-all: test test-yts-all

test-yts: $(TEST-DEPS)
	go test ./$(YTS-NAME) -count=1

test-yts-all: $(TEST-DEPS)
	RUNALL=1 go test ./$(YTS-NAME) -count=1 -v | \
	  awk '/     --- (PASS|FAIL): / {print $$2}' | \
	  sort | uniq -c

test-yts-failing: $(TEST-DEPS)
	RUNFAILING=1 go test ./$(YTS-NAME) -count=1 -v | \
	  awk '/     --- (PASS|FAIL): / {print $$2}' | \
	  sort | uniq -c

yaml-test-suite/testdata/data-2022-01-17/229Q: $(YTS-DIR)
	git submodule update --init --recursive $<

distclean:
	$(RM) -r $(ROOT)/.cache
