# ADR 0001: Clean and Hexagonal Architecture (Ports and Adapters)

## Status
Accepted

## Context
The system is an enterprise telemetry and compliance audit platform consisting of multiple distributed components:
- **Streamer**: Ingests and replays high-frequency recorded workstation event logs.
- **Agent**: Ingests desktop telemetry, maintains continuous workstation state across multi-display topologies, and evaluates compliance rules.
- **Server**: Ingests, indexes, and serves audit popup records for querying and compliance reporting.

In high-throughput event processing systems, coupling business rules with networking frameworks, HTTP routers, serialization formats, and persistence engines leads to:
1. Rapid architectural erosion and high maintenance costs.
2. Inability to unit-test domain logic deterministically without standing up external servers, sockets, or databases.
3. Tight coupling to third-party frameworks and infrastructure.
4. "Context rot" during AI-assisted development when business rules and transport details are intermingled in large files.

## Decision Drivers
- **Domain Purity**: Business logic must remain completely independent of external frameworks, transport protocols, and I/O.
- **Testability**: Core domain and application services must be 100% testable in memory in milliseconds without network sockets.
- **Interchangeability of Adapters**: In-memory storage, file logs, or SQL/NoSQL databases should be swappable without altering domain services.
- **Agentic AI Modularity**: Clear layer boundaries provide a "Truth Oracle" with minimal viable context (MVC) for AI pair programming.

## Considered Options
1. **Monolithic Script / Flat Service**: Fast to prototype, but combines transport, logic, and state, leading to low testability and high technical debt.
2. **Standard 3-Tier Layered Architecture (Controller-Service-Repository)**: Better separation, but dependencies typically point toward the database rather than domain abstractions.
3. **Clean & Hexagonal Architecture (Ports & Adapters)**: Strict dependency inversion where domain logic defines inbound and outbound ports, while adapters handle protocol, HTTP, and persistence details.

## Decision
We adopted **Clean and Hexagonal Architecture (Ports & Adapters)** across all modules (`streamer`, `agent`, `server`):

```text
+-------------------------------------------------------------+
|                       ADAPTERS LAYER                        |
|   HTTP Handlers ---> [Inbound Ports] ---> Domain Service    |
|                                                  |          |
|   HTTP Clients  <--- [Outbound Ports] <----------+          |
|   Repositories                                              |
+-------------------------------------------------------------+
|                        DOMAIN LAYER                         |
|   Pure Entities, Value Objects, Domain Events, Invariants   |
+-------------------------------------------------------------+
```

1. **Pure Domain Core (`libs/domain`)**: Contains only business entities (`Popup`, `Point`, `Rect`, `State`, `Rule`), domain events, and pure logic. No dependencies on `net/http`, `database/sql`, `encoding/json`, or Protobuf.
2. **Ports (`internal/ports`)**: Go interfaces defining inbound use cases (`AgentService`, `ServerService`) and outbound capabilities (`AuditClient`, `AuditRepository`, `RulesProvider`, `Timer`).
3. **Adapters (`internal/adapters`)**: Concrete implementations of ports:
   - `adapters/handler`: HTTP REST / Protobuf endpoints.
   - `adapters/client`: HTTP clients with timeouts.
   - `adapters/repository`: Thread-safe `MemoryRepository`.
   - `adapters/mapper`: Translators between external DTOs/Protobufs and domain models.
4. **Composition Roots (`cmd/main.go`)**: Wires dependencies at startup, binds configuration from environment variables, and manages graceful shutdown.

## Consequences

### Positive
- **Deterministic Testability**: Application services and domain rules are tested with in-memory mocks without network or disk latency.
- **Pluggable Persistence**: The audit storage adapter can be swapped from `MemoryRepository` to PostgreSQL or ClickHouse with zero domain changes.
- **AI Agent Context Scoping**: Small, single-responsibility packages allow AI coding agents to reason within focused, high-signal context windows without hallucinating cross-layer dependencies.

### Negative / Trade-offs
- Requires boilerplate mapping between transport DTOs/Protobufs and domain entities (`adapters/mapper`).
- Additional interface definitions and package abstractions.

## Agentic & AI Pair-Programming Implications
In AI-assisted software engineering, Hexagonal Architecture functions as a **Deterministic Truth Oracle**:
- The AI agent writes and validates pure domain logic against strict unit tests without being confused by networking frameworks.
- The separation of concerns prevents the AI from leaking transport concerns (e.g. HTTP status codes) into domain invariants.

## Academic & Industry Research References
- **Alistair Cockburn (2005)**: [Hexagonal Architecture (Ports and Adapters)](https://alistair.cockburn.us/hexagonal-architecture/) — foundational formulation of isolating core application logic behind ports.
- **Robert C. Martin (2012)**: [The Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) — dependency inversion principle and ring isolation.
- **Anthropic Research (2024)**: [Building Effective AI Agents](https://www.anthropic.com/research/building-effective-agents) — analyzing structured architectural boundaries for reliable autonomous agent execution.
- **Agentic Patterns (2024)**: [Catalog of Architectural Patterns for AI Engineering](https://agentic-patterns.com/) — decoupling deterministic domain kernels from AI orchestration.
