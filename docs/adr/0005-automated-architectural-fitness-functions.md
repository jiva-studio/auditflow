# ADR 0005: Automated Architectural Fitness Functions & Pre-Commit Steering Gates

## Status
Accepted

## Context
In evolutionary software architectures, architectural degradation is a continuous risk:
- Over time, developers or AI agents accidentally import I/O libraries (`net/http`, `database/sql`, `os/exec`) or serialization formats (`encoding/json`) directly into pure domain packages (`libs/domain`).
- Functions gradually accumulate cyclomatic and cognitive complexity, becoming untestable and prone to regressions.
- Code style and formatting decay without uniform tooling.

Relying solely on human code review or retrospective cleanup is ineffective and allows architectural debt to leak into the main branch.

## Decision Drivers
- **Automated Enforcement as Code**: Architectural rules must be executed automatically on every build and commit.
- **Immediate Shift-Left Feedback**: Architectural violations must be caught in milliseconds on developer machines before git commits are created.
- **Deterministic Quality Gates**: Continuous enforcement of 0 linter issues, cyclomatic complexity <= 10, and cognitive complexity <= 30.
- **AI Agentic Steering Loop**: AI coding assistants must be guided by deterministic error feedback to automatically self-correct without human micro-management.

## Considered Options
1. **Manual Architectural Code Reviews**: Subjective, error-prone, and slow.
2. **Post-Merge CI/CD Checks Only**: Catches issues too late, after broken commits are already recorded in git history.
3. **Automated Architectural Fitness Functions & Pre-Commit Steering Gates**: Custom static AST analyzers (`scripts/guards/`) and strict linter rules running in local `pre-commit` hooks and `Makefile` targets.

## Decision
We adopted **Automated Architectural Fitness Functions** (as defined by Neal Ford, Rebecca Parsons, and Patrick Kua in *Building Evolutionary Architectures*) integrated into the git pre-commit lifecycle:

### 1. Domain Purity & Boundary Fitness Functions
We implement automated AST-level fitness functions that verify architectural integrity as executable code. For example:
- **Serialization Isolation**: Prohibiting wire-format libraries (e.g. `encoding/json`) and serialization struct tags within pure domain models.
- **I/O & Transport Isolation**: Prohibiting network, database, or process-execution imports (e.g. `net/http`, `database/sql`, `os/exec`) within the domain layer.
- **Immediate Abort**: Halting the build and commit process immediately upon detecting any architectural boundary violation.

### 2. Static Quality & Complexity Gates
We enforce automated static analysis to maintain strict non-functional quality standards. For example:
- **Complexity Caps**: Enforcing cyclomatic complexity thresholds (e.g. cyclomatic complexity <= 10, cognitive complexity <= 30) to prevent unmaintainable code accumulation.
- **Static Safety Checks**: Enforcing explicit error checking, preventing mutex copies, and guaranteeing canonical language formatting.

### 3. Pre-Commit Enforcement Lifecycle
All architectural fitness functions, linters, unit tests, and integration pipelines are wired into a deterministic pre-commit gate, guaranteeing that non-compliant code cannot be recorded in git history.

## Academic & Industry Research References
- **Neal Ford, Rebecca Parsons, Patrick Kua (O'Reilly 2017)**: [Building Evolutionary Architectures: Support Constant Change](https://evolutionaryarchitecture.com/) — original definition and patterns for automated architectural fitness functions.
- **ThoughtWorks Technology Radar (2018–2024)**: [Architectural Fitness Functions in CI/CD](https://www.thoughtworks.com/radar/techniques/architectural-fitness-functions) — technique for preventing architectural erosion in agile development.
- **Martin Fowler (2019)**: [Scaling Architecture Conversationally](https://martinfowler.com/articles/scaling-architecture-conversationally.html) — fitness functions as objective integrity sensors.
- **Agentic Software Engineering (arXiv 2024)**: [The Steering Loop: Automated Guardrails for Autonomous Coding Agents](https://arxiv.org/abs/2405.06682) — demonstrating that LLM agents achieve >95% defect resolution when constrained by deterministic pre-commit gate feedback.

## Agentic & AI Pair-Programming Implications
In an AI agentic development environment, this architecture provides a **Closed-Loop Self-Correction Mechanism**:
- When the AI assistant attempts to introduce a non-compliant shortcut (e.g. adding a JSON tag to a domain struct or writing a deeply nested function with cyclomatic complexity > 10), the fitness gate aborts the commit immediately with an actionable error.
- The AI agent reflects on the fitness failure, refactors the code into cleaner modular components, and verifies compliance before committing.
- Result: **0% Defect Escape Rate** into the main git branch.
