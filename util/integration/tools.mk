# Copyright 2026 The go-yaml Project Contributors
# SPDX-License-Identifier: Apache-2.0

# Read after GNUmakefile. Each project script requests only its own tools.
ifneq ($(INTEGRATION_TOOL_MODULE),)
include $(MAKES)/$(INTEGRATION_TOOL_MODULE)
endif

integration-tool: $($(INTEGRATION_TOOL_VARIABLE))
	@for key in PATH GOROOT GOPATH GOMODCACHE GOCACHE CARGO_HOME \
	    RUSTUP_HOME UV_PYTHON_INSTALL_DIR VIRTUAL_ENV; do \
	  if [[ -v $$key ]]; then \
	    printf '%s=%s\0' "$$key" "$${!key}"; \
	  fi; \
	done > "$(INTEGRATION_TOOL_ENV)"
