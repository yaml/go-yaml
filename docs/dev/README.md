# Developer Documentation

This directory contains internal documentation for go-yaml developers and contributors.

## Contents

- **[go-yaml-internals.md](go-yaml-internals.md)** - Comprehensive analysis of the load and dump stack internals, including all stages, transforms, security features, and architectural issues
- **[how-go-yaml-works.md](how-go-yaml-works.md)** - High-level overview of how go-yaml processes YAML
- **[dump-load-api.md](dump-load-api.md)** - API documentation for dump and load operations

### Mermaid Diagrams

- **[pipeline-overview.mmd](pipeline-overview.mmd)** - Complete processing pipeline from bytes to Go values and back
- **[call-hierarchy.mmd](call-hierarchy.mmd)** - Function call hierarchy showing pull-based (load) vs push-based (dump) architecture
- **[comment-flow.mmd](comment-flow.mmd)** - Comment handling flow through Scanner → Parser → Composer

## For Users

User-facing documentation is in the parent [docs/](../) directory:
- [README.md](../README.md) - Main documentation index
- [options.md](../options.md) - Configuration options reference
- [v3-to-v4-migration.md](../v3-to-v4-migration.md) - Migration guide for upgrading from v3 to v4
