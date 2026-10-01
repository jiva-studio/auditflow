# ADR 0002: Domain-Driven Design (DDD) & Rule Specification Pattern

## Status
Accepted

## Context
The core domain problem involves analyzing continuous desktop events (mouse clicks, keystrokes, clipboard mutations, active window changes, and spatial OCR text blocks across multiple display monitors) to determine when business compliance rules are satisfied.

Business rules have diverse triggering conditions:
- Event-driven click rules on OCR text (e.g. clicking on a forwarded email or Jira "Done" button).
- State-driven passive rules (e.g. copying an invoice number while Outlook is active).
- Multi-monitor coordinate normalization (handling DPI scaling, display bounds, and negative coordinate margins in maximized Windows).
- Deduplication of static frames (`deduplicated_from`) and suppression of duplicate passive alerts.

Without domain modeling, desktop state logic quickly becomes an unmaintainable God-Object filled with brittle `if/else` checks and state leakage.

## Decision Drivers
- **Rich Domain Model**: Desktop state must capture spatial OCR topology, active window history, and clipboard state with clear encapsulation.
- **Specification Pattern**: Business rules must be modeled as composable, testable domain specifications rather than procedural switch-cases.
- **Ubiquitous Language**: Terminology across domain code must match the problem space (`DesktopState`, `SpatialFacet`, `WindowFacet`, `ClipboardFacet`, `Rule`, `Popup`).
- **Zero I/O and Serialization Leakage**: The domain layer must be completely pure.

## Considered Options
1. **Procedural Scripting with Regex Matching**: Fast to write initially, but impossible to scale to complex multi-monitor topologies, frame deduplication, and template substitutions.
2. **Generic Complex Event Processing (CEP) Engine**: External rule engine libraries add massive dependency bloat and reduce debuggability in Go.
3. **Domain-Driven Design with Facet Decomposition & Specification Pattern**: Custom, lightweight domain model composed of focused facets and composable rule specifications.

## Decision
We adopted **Domain-Driven Design (DDD)** for the core business library (`libs/domain`):

### 1. Ubiquitous Language & Core Aggregates
We establish a shared Ubiquitous Language reflecting workstation telemetry and compliance. As examples of this architectural approach:
- **`State` Aggregate Root (e.g., `desktop.State`)**: Encapsulates continuous workstation reality. To maintain low complexity and high cohesion, large domain aggregates are decomposed into focused facets (for example, `SpatialFacet` for display bounds and OCR point-in-rect hit-testing, `WindowFacet` for active application context, and `ClipboardFacet` for clipboard transitions).
- **Immutable Value Objects**: Used for domain primitives with structural equality and invariant validation upon construction (for example, `Point`, `Rect`, `Display`, `Popup`, `Filter`).

### 2. Specification Pattern for Business Rules
Business compliance rules (in `libs/domain/rules`) are modeled as composable domain specifications rather than procedural procedural blocks. For example:
- **Condition Specifications**: Composable predicates such as `TextSpec` (wildcards, case-insensitivity), `WindowSpec` (active process/window title matching), `ClipboardSpec`, and `ClickSpec` (spatial bounding-box intersection).
- **Domain Template Substitution**: Pure domain token replacement (e.g. `{click}`, `{window_title}`, `{clipboard}`) within popup titles and bodies based on matched evaluation context.

## Consequences

### Positive
- **High Cohesion & Low Complexity**: Each facet and specification has cyclomatic complexity <= 10, ensuring exhaustive branch coverage and easy reasoning.
- **Robust Multi-Monitor Support**: Spatial facet correctly maps cursor coordinates across primary and secondary displays.
- **100% Mutation Resistance**: Every invariant and boundary condition is verified by targeted unit tests.

### Negative / Trade-offs
- Requires explicit state update methods (`s.state.Apply(ev)`) and evaluation context extraction (`s.state.BuildEvaluationContext(...)`).

## Agentic & AI Pair-Programming Implications
DDD provides strict semantic boundaries that guide AI code generation:
- Prevents the AI assistant from mixing spatial OCR hit-testing with HTTP payload handling.
- Rich Value Objects enforce runtime validation at instantiation, preventing AI-generated code from propagating invalid state through the system.

## Academic & Industry Research References
- **Eric Evans (2003)**: [Domain-Driven Design: Tackling Complexity in the Heart of Software](https://www.domainlanguage.com/ddd/) — ubiquitous language, aggregates, and domain isolation.
- **Martin Fowler & Eric Evans (2002)**: [Specifications Pattern](https://martinfowler.com/apsupp/spec.pdf) — modeling combinable business rules as first-class domain objects.
- **Vaughn Vernon (2013)**: [Implementing Domain-Driven Design](https://kalele.io/books/) — modeling state invariants and aggregate boundaries.
