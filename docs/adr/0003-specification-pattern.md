# ADR 0003: Specification Pattern

## Status
Accepted

## Context
Notification popups trigger based on dynamic criteria (e.g. clicking specific OCR text, copying invoice identifiers, or matching active window titles). Writing these rules as procedural `if/else` checks leads to rigid code that is difficult to extend and test.

## Decision
We adopted the **Specification Pattern** for rule evaluation (`modules/libs/domain/rules`):

1. **Composable Predicates**: Rule conditions are modeled as modular specifications:
   - `TextSpec`: Text matching with wildcard support.
   - `WindowSpec`: Process name and window title conditions.
   - `ClipboardSpec`: Clipboard content inspection.
   - `ClickSpec`: Spatial bounding-box intersection over OCR text blocks.
2. **Rule Engine**: Evaluates incoming domain events against registered specifications and produces matching results.
3. **Template Substitution**: Replaces tokens (`{click}`, `{window_title}`, `{clipboard}`) in popup titles and bodies from the evaluation context.

## Consequences

### Positive
- New condition types can be introduced without modifying existing evaluation code.
- Each specification predicate is tested in isolation.
- Rules defined in JSON map directly to domain specification trees.

### Negative
- Slight allocation overhead from composing specification objects.

## Agentic Engineering Implications
The Specification Pattern provides a declarative structure for autonomous rule composition:
- **Isolated Modifications**: Autonomous agents can implement, extend, or debug individual rule predicates without touching the core evaluation loop.
- **Zero Nested Branching**: Replaces complex multi-branch `if/else` logic with composable objects, keeping agent generation deterministic.

## Academic & Industry Research References
- **Martin Fowler & Eric Evans (2002)**: [Specifications Pattern](https://martinfowler.com/apsupp/spec.pdf) — modeling combinable business rules as first-class domain objects.
