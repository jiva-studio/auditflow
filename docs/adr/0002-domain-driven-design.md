# ADR 0002: Domain-Driven Design

## Status
Accepted

## Context
Workstation state tracks multiple data sources: display geometry, active windows, clipboard contents, and OCR text blocks across multiple monitors. Managing this state in procedural code leads to a bloated state structure and fragile state synchronization.

## Decision
We adopted **Domain-Driven Design** for the core business library (`modules/libs/domain`):

1. **Aggregate Root (`desktop.State`)**: Manages the consistency boundary for desktop state and applies domain events via `Apply(ev)`.
2. **Facet Decomposition**: State is partitioned into focused facets:
   - `SpatialFacet`: Display bounds, coordinates, OCR text blocks, and geometric hit-testing.
   - `WindowFacet`: Active application processes, window titles, and focus history.
   - `ClipboardFacet`: Clipboard text mutations and timestamps.
3. **Immutable Value Objects**: Primitives (`Point`, `Rect`, `Display`, `Popup`, `Filter`) validate invariants at construction.

## Consequences

### Positive
- Clear aggregate boundaries with low complexity per facet.
- State mutations are restricted to valid domain methods.
- Domain layer remains free of external dependencies.

### Negative
- Requires building explicit evaluation contexts for external callers.

## Agentic Engineering Implications
DDD provides strict semantic boundaries that guide autonomous agent code generation:
- **Constraint Boundaries**: Terminology and aggregate roots constrain agent modifications to cohesive units, preventing state corruption.
- **Self-Validating Primitives**: Value Objects enforce invariant validation at instantiation, preventing agents from propagating invalid intermediate state.
- **Complexity Guardrails**: Small facets keep cyclomatic complexity low, enabling autonomous agents to reason about state transitions exhaustively.

## Academic & Industry Research References
- **Eric Evans (2003)**: [Domain-Driven Design: Tackling Complexity in the Heart of Software](https://www.domainlanguage.com/ddd/) — ubiquitous language, aggregates, and domain isolation.
- **Vaughn Vernon (2013)**: [Implementing Domain-Driven Design](https://kalele.io/books/) — modeling state invariants and aggregate boundaries.
