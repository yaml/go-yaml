# Agent Configuration Notes

## Question: Does Copilot read .agents.md files?

**Short Answer**: The behavior depends on which "Copilot" you're referring to.

### GitHub Copilot (Code Completion)

GitHub Copilot, the code completion tool, does **not** read `.agents.md` files. It focuses on:
- Immediate code context
- Comments in the current file
- File and symbol names
- Common patterns in the codebase

### GitHub Copilot Workspace / SWE Agents

Some GitHub Copilot-powered agents (like SWE/coding agents) may have access to custom configuration, but the `.agents.md` convention is not a standard GitHub Copilot feature.

### Claude-based Agents

The `.agents.md` file convention is more commonly associated with **Claude/Anthropic-based** coding agents. These agents can be configured to:
- Read custom instruction files
- Follow repository-specific guidelines
- Understand project-specific context

## Recommendation

For maximum compatibility across different AI coding assistants:

1. **Use standard documentation**: README.md, CONTRIBUTING.md
2. **Write clear comments**: In-code documentation is universally understood
3. **Follow conventions**: Consistent naming and structure helps all tools
4. **Test with your specific tool**: Different AI assistants have different capabilities

## Conclusion

You are correct that `.agents.md` reading is "a Claude thing" - it's not a standard GitHub Copilot feature but is more commonly used with Claude-based agents.
