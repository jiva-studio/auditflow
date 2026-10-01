# ADR 0001: Clean and Hexagonal Architecture

## Status
Accepted

## Context
AuditFlow consists of distributed services (`streamer`, `agent`, `server`) processing desktop activity streams. Coupling business logic with transport protocols (HTTP, Protobuf), serialization formats, and persistence engines causes tight coupling to frameworks and prevents testing domain logic in isolation.

## Decision
We adopted **Clean Architecture** (Dependency Inversion) and **Hexagonal Architecture** (Ports and Adapters) across all applications (`modules/apps/`):

1. **Dependency Rule**: Dependencies point strictly inward. Domain logic has zero dependencies on transport, database, or external libraries.
2. **Ports (`internal/ports`)**: Go interfaces defining inbound driving use cases (`AgentService`, `ServerService`) and outbound driven dependencies (`AuditClient`, `AuditRepository`, `RulesProvider`, `Timer`).
3. **Adapters (`internal/adapters`)**: Concrete implementations of ports:
   - `adapters/handler`: HTTP REST and Protobuf endpoints.
   - `adapters/client`: HTTP clients with timeouts and error handling.
   - `adapters/repository`: Thread-safe in-memory repository.
   - `adapters/mapper`: Mappers between wire DTOs and internal domain models.
4. **Composition Root (`cmd/main.go`)**: Wires dependencies at startup and handles graceful shutdown.

## Consequences

### Positive
- Business logic is completely isolated from transport and storage layers.
- Application services and domain rules are tested in memory without network or disk latency.
- Storage adapters can be swapped without touching core domain services.

### Negative
- Requires explicit mapping between wire DTOs and domain entities.
- Increases the initial number of interfaces and package abstractions.

## Agentic Engineering Implications
Hexagonal and Clean Architecture provides clear boundary isolation for autonomous AI coding agents:
- **Minimal Viable Context (MVC)**: Small, single-responsibility packages allow agents to operate within focused context windows without cross-layer pollution.
- **Deterministic Separation**: Prevents autonomous agents from accidentally leaking transport details (HTTP headers, status codes) into domain invariants.
- **Adapter Swappability**: Agents can implement and verify new adapters against port interfaces using localized unit tests.

## Academic & Industry Research References
- **Alistair Cockburn (2005)**: [Hexagonal Architecture (Ports and Adapters)](https://alistair.cockburn.us/hexagonal-architecture/) — foundational formulation of isolating core application logic behind ports.
- **Robert C. Martin (2012)**: [The Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) — dependency inversion principle and ring isolation.
- **Anthropic Research (2024)**: [Building Effective AI Agents](https://www.anthropic.com/research/building-effective-agents) — structured architectural boundaries for reliable autonomous agent execution.
- **Agentic Patterns (2024)**: [Catalog of Architectural Patterns for AI Engineering](https://agentic-patterns.com/) — decoupling deterministic domain kernels from AI orchestration.
