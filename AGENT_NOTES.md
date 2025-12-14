# AI Agent Configuration Notes

## Overview of AI Agent Instruction Files

Different AI coding assistants read different instruction files. Here's a comprehensive breakdown:

### GitHub Copilot Workspace

GitHub Copilot Workspace and some GitHub Copilot-powered agents can read instructions from:
- **`.github/copilot-instructions.md`** (primary convention)
- **`.github/[workspace-name]/instructions.md`** (workspace-specific, e.g., `.github/docs/instructions.md`)

These files are **GitHub-specific** and primarily work with GitHub Copilot Workspace and GitHub Copilot Chat.

### Claude-based Agents

Claude and Anthropic-based coding agents may read:
- **`.cursorrules`** (Cursor editor convention)
- **`.claude-instructions.md`** or **`.agents.md`** (custom conventions)
- Custom instruction files specified in agent configuration

### Aider

Aider (AI pair programming tool) reads:
- **`.aider.conf.yml`** (configuration file)
- Can be configured to read custom instruction files

### Cline, Roo-Coder, and Other VSCode Extensions

Many VSCode AI extensions read:
- **`.clinerules`** or **`.clineignore`** (Cline-specific)
- **`.cursorrules`** (Cursor editor convention, adopted by some extensions)
- Project-specific configuration files

## Universal Best Practices

For maximum compatibility across **all AI coding assistants** (GitHub Copilot, Claude, Cursor, Aider, etc.):

1. **README.md** - All agents read this for project context
2. **CONTRIBUTING.md** - Guidelines for contributing (widely supported)
3. **Inline comments** - Code comments are universally understood
4. **Standard documentation** - docs/, API documentation, etc.

## File Naming Recommendations

If you want instructions for **multiple agents**:

- **`.github/copilot-instructions.md`** - For GitHub Copilot Workspace
- **`.cursorrules`** - For Cursor editor and some compatible VSCode extensions
- **README.md** - Universal fallback for all agents

## Specific Answers

**Q: Does GitHub Copilot read `.agents.md`?**
A: No. GitHub Copilot uses `.github/copilot-instructions.md` or `.github/[workspace-name]/instructions.md`.

**Q: Does Claude read `.github/[workspace-name]/instructions.md` files?**

A: No, this is a GitHub Copilot convention. Claude-based agents use custom instruction files that vary by implementation. Examples include `.cursorrules` in Cursor editor, or custom configuration in other tools.

**Q: What file should I use for maximum compatibility?**

A: Use **README.md** and **CONTRIBUTING.md** for universal support. Add `.github/copilot-instructions.md` for GitHub Copilot and tool-specific files (like `.cursorrules` for Cursor) as needed.
