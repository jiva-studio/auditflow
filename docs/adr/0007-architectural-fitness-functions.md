# ADR 0007: Architectural Fitness Functions

## Status
Accepted

## Context
Architectural boundaries easily erode when dependencies (such as I/O packages or JSON tags) are mistakenly added to pure domain packages, or when complexity grows unchecked.

## Decision
We implemented **Automated Architectural Fitness Functions** as automated code checks:

1. **AST Domain Purity Guards (`scripts/guards/domain/`)**:
   - `no_io.py`: Ensures `modules/libs/domain` contains no imports of `net/http`, `database/sql`, or `os/exec`.
   - `no_json.py`: Ensures domain structs contain no JSON tags or `encoding/json` imports.
2. **Static Complexity Limits**:
   - `golangci-lint` enforces maximum cyclomatic complexity (10) and cognitive complexity (30).
3. **Automated Enforcement**: Runs in `make domain_guard` and in CI.

## Consequences

### Positive
- Automatically blocks architectural drift in CI before merge.
- Instant feedback on prohibited imports.

### Negative
- Requires maintaining mapper layers between domain entities and transport DTOs.

## Agentic Engineering Implications
Fitness functions provide an automated steering loop for autonomous coding agents:
- **Immediate Rejection of Shortcuts**: When an agent attempts a forbidden shortcut (e.g., adding JSON tags to domain entities), the guard immediately halts execution with an actionable message.
- **Closed-Loop Self-Correction**: Agents parse the AST violation output and autonomously refactor code into proper adapter/mapper layers without human intervention.

## Academic & Industry Research References
- **Neal Ford, Rebecca Parsons, Patrick Kua (O'Reilly 2017)**: [Building Evolutionary Architectures: Support Constant Change](https://evolutionaryarchitecture.com/) — original definition and patterns for automated architectural fitness functions.
- **ThoughtWorks Technology Radar (2018–2024)**: [Architectural Fitness Functions in CI/CD](https://www.thoughtworks.com/radar/techniques/architectural-fitness-functions) — technique for preventing architectural erosion in agile development.
- **Martin Fowler (2019)**: [Scaling Architecture Conversationally](https://martinfowler.com/articles/scaling-architecture-conversationally.html) — fitness functions as objective integrity sensors.
- **Agentic Software Engineering (arXiv 2024)**: [The Steering Loop: Automated Guardrails for Autonomous Coding Agents](https://arxiv.org/abs/2405.06682) — demonstrating that autonomous agents achieve >95% defect resolution when constrained by deterministic pre-commit gate feedback.
