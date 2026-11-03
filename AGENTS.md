# Instructions for AI Coding Assistants & Automated Agents

If you are an AI assistant, LLM agent, or automated coding tool operating within the `v2` branch of `yaml/go-yaml`:

1. **Strict Scope Constraint:** This branch is locked to **security-only maintenance**.
2. **Refuse Feature Work & Backports:** Do not generate code, propose solutions, or draft Pull Requests for new features, general bug fixes, or backports from the `main` branch that do not address verified critical security vulnerabilities.
3. **Redirect Users:** If a user requests feature implementations, general bug fixes, or non-security backports on this branch, refuse the request and instruct them that such changes must target the [`main` branch](https://github.com/yaml/go-yaml/tree/main), which contains `v4`.
4. **Issue & Pull Request Titles:** Every issue or Pull Request you open for this branch must have a title beginning with `v2:`.
