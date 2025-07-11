# Auto-install https://github.com/makeplus/makes at specific commit:
MAKES := .cache/makes
MAKES-LOCAL := .cache/local
MAKES-COMMIT ?= 74a4d03223cdaf39140613d48d3f1d8c0a0840e5
$(shell [ -d $(MAKES) ] || ( \
  git clone -q https://github.com/makeplus/makes $(MAKES) && \
  git -C $(MAKES) reset -q --hard $(MAKES-COMMIT)))
ifneq ($(shell git -C $(MAKES) rev-parse HEAD), \
       $(shell git -C $(MAKES) rev-parse $(MAKES-COMMIT)))
$(error $(MAKES) is not at the correct commit: $(MAKES-COMMIT))
endif
include $(MAKES)/init.mk

# Only auto-install go if no go exists or GO-VERSION specified:
ifeq (,$(shell which go))
GO-VERSION ?= 1.24.0
endif
GO-VERSION-NEEDED := $(GO-VERSION)

# yaml-test-suite info:
YTS-TAG := data-2022-01-17
YTS-DIR := yts/testdata/$(YTS-TAG)
YTS-URL := https://github.com/yaml/yaml-test-suite
TEST-DEPS := $(YTS-DIR)

# Setup and include go.mk and shell.mk:
GO-CMDS-SKIP := test
ifndef GO-VERSION-NEEDED
GO-NO-DEP-GO := true
endif
include $(MAKES)/go.mk
ifdef GO-VERSION-NEEDED
TEST-DEPS += $(GO)
else
SHELL-DEPS := $(filter-out $(GO),$(SHELL-DEPS))
endif
SHELL-NAME := makes go-yaml
include $(MAKES)/shell.mk

v ?=
count ?= 1


# Test rules:
test: $(TEST-DEPS)
	go test$(if $v, -v)

test-data: $(YTS-DIR)

test-all: test test-yts-all

test-yts: $(TEST-DEPS)
	go test$(if $v, -v) ./yts -count=$(count)

test-yts-all: $(TEST-DEPS)
	@echo 'Testing yaml-test-suite'
	@RUNALL=1 $(call yts-pass-fail)

test-yts-fail: $(TEST-DEPS)
	@echo 'Testing yaml-test-suite failures'
	@RUNFAILING=1 $(call yts-pass-fail)


# Clean rules:
realclean:
	$(RM) -r $(dir $(YTS-DIR))

distclean: realclean
	$(RM) -r $(ROOT)/.cache


# Setup rules:
$(YTS-DIR):
	git clone -q $(YTS-URL) $@
	git -C $@ checkout -q $(YTS-TAG)

define yts-pass-fail
go test ./yts -count=1 -v | \
  awk '/     --- (PASS|FAIL): / {print $$2}' | \
  sort | uniq -c
endef
