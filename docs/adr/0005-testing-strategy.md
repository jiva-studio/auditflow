# ADR 0005: Testing Strategy

## Status
Accepted

## Context
Testing must verify correctness across pure business algorithms, transport adapters, and multi-process pipelines. Unit tests alone cannot verify network integration, while end-to-end tests alone are too slow for rapid feedback.

## Decision
We adopted a **Multi-Tier Testing Pyramid**:

1. **Domain Unit Tests (`modules/libs/domain/...`)**: Fast in-memory tests verifying state transitions, geometry calculations, and rule evaluation.
2. **Adapter Tests (`internal/adapters/...`)**: Tests for HTTP handlers, clients, error mapping, and concurrency safety under the Go race detector.
3. **Blackbox E2E Integration Tests (`tests/e2e/...`)**: Multi-process integration tests that build binaries, spin up ephemeral ports, replay recorded sessions (`./data/*.tar.gz`), and verify output records.

## Consequences

### Positive
- Fast feedback loop from unit tests running in milliseconds.
- High integration confidence from blackbox replay tests.
- Dynamic port allocation prevents port conflicts during parallel test execution.

### Negative
- End-to-end tests require compiling binaries and managing test processes.

## Agentic Engineering Implications
Tiered verification provides fast, actionable feedback loops for autonomous agents:
- **Instant Local Validation**: Unit tests provide millisecond verification during agent test-driven development (TDD).
- **Flake-Free CI**: Dynamic ephemeral ports and bounded memory execution guarantee that agents receive deterministic test pass/fail signals.

## Academic & Industry Research References
- **Mike Cohn (2009)**: [Succeeding with Agile: Software Development Using Scrum](https://www.mountaingoatsoftware.com/books/succeeding-with-agile) — original definition of the Test Automation Pyramid.
- **Martin Fowler (2012)**: [The Test Pyramid](https://martinfowler.com/bliki/TestPyramid.html) — portfolio of testing layers balancing execution speed and integration fidelity.
