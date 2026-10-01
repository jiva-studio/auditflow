# Comments & Documentation Guidelines

A comment is read by someone who has the code in front of them. It earns its place by explaining *why* something is done when the code cannot.

---

## 1. Comment Rules

- **Explain Why, Not What**: Describe non-obvious design decisions, constraints, or invariants. Do not narrate obvious code operations.
- **Stand on Its Own**: Do not reference ephemeral external tickets or chats.
- **Concise & Proportional**:
  - A field or flag: 1 line.
  - A non-obvious algorithm or trade-off: 2–3 lines.
  - Architecture-level docs: place in `docs/` and link.
- **Exported Identifiers**: Every exported type, constant, variable, and function in Go must have a concise doc-comment starting with the identifier name.
- **No AI / Session Metadata**: Code and comments carry no trace of prompts, model names, or tool sessions.
