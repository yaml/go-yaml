M := .git/.makes
$(shell [ -d $M ] || git clone -q https://github.com/makeplus/makes $M)
include $M/init.mk
include $M/go.mk
include $M/ys.mk

# Print Makefile targets summary
default::

$(GO-CMDS):: $(GO)
	go $@ $A

ys-test: $(YS)
	prove -v test/
