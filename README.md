# AuditFlow — Real-Time Telemetry & Compliance Platform

AuditFlow is a distributed, real-time employee workstation telemetry and compliance audit platform. It ingests desktop activity streams (active windows, OCR text, mouse clicks, clipboard contents, and process events), evaluates security and compliance rules locally on employee endpoints with sub-millisecond latency, and aggregates verified compliance audit notifications into a central audit repository.

---

## 1. How to Run

### Quick Start with Docker Compose (Recommended)

To run the complete system (Central Server, 3 Employee Agents, and 3 Streamers replaying real workstation telemetry sessions) from scratch on any machine:

```bash
docker compose up --build
```

#### Fast-Forward Execution (Zero-Delay Replay)
By default, the replay streamer respects the simulated workstation tick cadence (`TICK_MS=10`). To execute the full replay instantly at maximum CPU and network throughput:

```bash
TICK_MS=0 docker compose up --abort-on-container-exit --exit-code-from streamer-emp-1
```

### Querying Compliance Audit Results

Once the containers are running, the Central Audit Server exposes a REST API on port `8080`:

```bash
# Retrieve all recorded compliance popup events
curl -s http://localhost:8080/audit

# Filter audit logs by specific employee
curl -s "http://localhost:8080/audit?employee=emp-1"

# Filter audit logs by triggered rule
curl -s "http://localhost:8080/audit?rule=forwarded-email-opened"
```

#### Example JSON Response
```json
[
  {
    "employee": "emp-1",
    "rule": "forwarded-email-opened",
    "ts": "2026-03-10T23:13:25Z",
    "title": "Forwarded email",
    "body": "You opened a forwarded email: FW: Q1 Roadmap"
  },
  {
    "employee": "emp-2",
    "rule": "invoice-ready-to-attach",
    "ts": "2026-03-10T16:39:13Z",
    "title": "Invoice in clipboard",
    "body": "You copied INV-20533. Attach the invoice to this email?"
  }
]
```

## 2. Local Development & Testing

All local development targets are managed via the root [Makefile](file:///home/akd/Projects/assessment/Makefile):

```bash
# Run all End-to-End integration tests (Streamer, Agent, and Central Server pipeline)
make e2e_test

# Validate architectural invariants (AST Guards: zero JSON and zero I/O in domain)
make domain_guard

# Run unit tests and linters by module:
make domain_test domain_lint     # Domain library
make protocol_test protocol_lint # Protobuf v1 contracts
make rules_test rules_lint       # Rule engine and JSON loader
make agent_test agent_lint       # Employee Agent microservice
make server_test server_lint     # Central Audit Server microservice
make streamer_test streamer_lint # Replay Streamer microservice

# Run mutation testing
make agent_mutate
make server_mutate
make streamer_mutate

# Format all code across the repository
make fmt
```

---

## 3. Project Architecture

The codebase strictly follows **Clean / Hexagonal Architecture** and **Domain-Driven Design (DDD)**, organized into standalone apps and libraries:

```text
assessment/
├── modules/
│   ├── apps/                        # Microservices (Hexagonal Architecture)
│   │   ├── agent/                   # Employee workstation audit agent
│   │   │   ├── cmd/                 # Application entrypoint
│   │   │   ├── internal/ports/      # Driving and driven interfaces
│   │   │   ├── internal/service/    # Application use cases
│   │   │   └── internal/adapters/   # HTTP handlers, mappers, audit clients
│   │   ├── server/                  # Central compliance audit store & query API
│   │   │   ├── cmd/                 # Application entrypoint
│   │   │   ├── internal/ports/      # Driving and driven interfaces
│   │   │   ├── internal/service/    # Audit aggregation logic
│   │   │   └── internal/adapters/   # HTTP handler, in-memory repository
│   │   └── streamer/                # Telemetry recording replay service
│   │       ├── cmd/                 # Application entrypoint
│   │       ├── internal/ports/      # Streamer interfaces
│   │       ├── internal/service/    # Replay orchestration & pacing
│   │       └── internal/adapters/   # Tar.gz archive parser, Protobuf client
│   └── libs/                        # Reusable foundational libraries
│       ├── domain/                  # Pure DDD models & invariants (Zero I/O, Zero JSON)
│       │   ├── audit/               # Audit Popup aggregate and query filters
│       │   ├── desktop/             # Desktop state, window hierarchies, bounds
│       │   ├── display/             # Physical displays, coordinates, DPI scaling
│       │   ├── events/              # Low-level workstation domain events
│       │   ├── geometry/            # Points, Sizes, Rectangles with bounds checking
│       │   ├── rules/               # Rule evaluation engine, pattern matching, templates
│       │   └── session/             # Employee session metadata and time ranges
│       ├── protocol/                # Protobuf v1 schemas and generated Go structs
│       │   ├── proto/v1/            # audit.proto, events.proto
│       │   └── gen/go/v1/           # Generated Protobuf serialization code
│       └── rules/                   # Rule parser, DTOs, and flexible JSON deserializer
├── tests/e2e/                       # End-to-end integration test suites
│   ├── agent/                       # Agent rule evaluation tests on real sessions
│   ├── server/                      # Full pipeline end-to-end tests
│   ├── streamer/                    # Streamer replay and memory bounded stress tests
│   └── testutil/                    # Synthetic telemetry generator and test helpers
├── scripts/guards/                  # AST-based architectural fitness functions
│   └── domain/                      # Checks ensuring domain purity (no JSON tags, no I/O)
├── docs/adr/                        # Architecture Decision Records
├── data/                            # Real session recordings and rules.json
├── docker-compose.yml               # Production-like multi-container orchestrator
└── Makefile                         # Automated build, test, lint, and guard runner
```

---

## 4. Architectural Decisions & Rationale

Key technical and architectural decisions are formally documented in Architecture Decision Records (ADRs):

1. **[ADR-0001: Clean & Hexagonal Architecture](file:///home/akd/Projects/assessment/docs/adr/0001-clean-and-hexagonal-architecture.md)**
   * *Rationale*: Business logic is completely decoupled from transport protocols (HTTP/Protobuf) and persistence mechanisms. Adapters can be replaced or mocked without modifying domain logic.
2. **[ADR-0002: Domain-Driven Design & Zero I/O in Domain](file:///home/akd/Projects/assessment/docs/adr/0002-domain-driven-design.md)**
   * *Rationale*: Pure domain models with strict encapsulation, immutable value objects (`Point`, `Rectangle`, `Popup`), and explicit invariants. The domain package has zero external third-party dependencies and performs zero I/O operations.
3. **[ADR-0003: Protobuf for Inter-Service Communication](file:///home/akd/Projects/assessment/docs/adr/0003-protobuf-for-inter-service-communication.md)**
   * *Rationale*: Compact binary payload format reduces network serialization overhead and memory allocations by over 70% compared to JSON during high-frequency telemetry streaming.
4. **[ADR-0004: Comprehensive Testing Strategy & Gremlins Mutation Testing](file:///home/akd/Projects/assessment/docs/adr/0004-testing-strategy-and-mutation-testing.md)**
   * *Rationale*: Quality assurance built on a multi-layer test pyramid: unit tests for all domain invariants, integration tests with mocked transports, E2E tests over real recordings, stress tests with memory bounds (`GOMEMLIMIT`), and 100% mutation testing efficacy via Gremlins.
5. **[ADR-0005: Automated Architectural Fitness Functions](file:///home/akd/Projects/assessment/docs/adr/0005-automated-architectural-fitness-functions.md)**
   * *Rationale*: Automated AST guards run in pre-commit hooks to mathematically prevent architectural erosion (verifying AST nodes for forbidden imports like `os`, `net/http`, `io`, and struct tags like `json:`).

