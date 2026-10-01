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

All local development targets are managed via the root [Makefile](Makefile):

```bash
# Run all End-to-End integration tests (Streamer, Agent, and Central Server pipeline)
make e2e_test

# Validate architectural invariants (AST Guards: zero JSON and zero I/O in domain)
make domain_guard

# Run unit tests and linters by module:
make domain_test domain_lint       # Domain library
make protocol_test protocol_lint   # Protobuf v1 contracts
make rules_test rules_lint         # Rule engine and JSON loader
make telemetry_test telemetry_lint # Telemetry, slog, and tracing library
make agent_test agent_lint         # Employee Agent microservice
make server_test server_lint       # Central Audit Server microservice
make streamer_test streamer_lint   # Replay Streamer microservice

# Build static production binaries into bin/
make build

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
├── .github/
│   └── workflows/
│       └── ci.yml                   # Automated CI/CD pipeline (Lint, Test, E2E, Build, Artifacts)
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
│       ├── rules/                   # Rule parser, DTOs, and flexible JSON deserializer
│       └── telemetry/               # Structured slog logger, context trace ID propagation & middleware
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

## 4. Architectural Decisions

1. **[ADR-0001: Clean & Hexagonal Architecture](docs/adr/0001-clean-and-hexagonal-architecture.md)** — Decouples pure domain logic from transports (HTTP/Protobuf) and storage, allowing adapters and databases to be swapped or mocked without touching business logic.
2. **[ADR-0002: Domain-Driven Design & Zero I/O in Domain](docs/adr/0002-domain-driven-design.md)** — Enforces strict invariants and immutable value objects (`Point`, `Rectangle`, `Popup`) with zero external dependencies, zero JSON tags, and zero I/O in the domain.
3. **[ADR-0003: Protobuf for Inter-Service Communication](docs/adr/0003-protobuf-for-inter-service-communication.md)** — Provides strongly-typed inter-service contracts and cuts network serialization overhead and allocations by >70% compared to JSON during high-frequency telemetry streaming.
4. **[ADR-0004: Comprehensive Testing Strategy & Gremlins Mutation Testing](docs/adr/0004-testing-strategy-and-mutation-testing.md)** — Combines unit tests for domain invariants, black-box E2E tests over real recordings with bounded memory (`GOMEMLIMIT`), and Gremlins mutation testing to verify test effectiveness.
5. **[ADR-0005: Automated Architectural Fitness Functions](docs/adr/0005-automated-architectural-fitness-functions.md)** — Automatically prevents architectural erosion via AST guards in pre-commit hooks and CI, prohibiting forbidden imports (`os`, `io`, `encoding/json`) in pure packages.

---

## 5. Continuous Integration & Delivery (CI/CD)

The automated [GitHub Actions CI/CD Pipeline](.github/workflows/ci.yml) enforces quality gates on every Pull Request and commit to `main`:

* **Stage 1: Quality Gates & Fitness Functions**: Verifies code formatting with `gofmt`, executes Python AST domain purity guards (`no_json.py`, `no_io.py`), and runs `golangci-lint` across all Go modules.
* **Stage 2: Unit & Integration Tests**: Runs test suites with Go race detector (`-race`) and generates aggregated coverage profiles.
* **Stage 3: End-to-End System Tests**: Executes the full integration test pipeline across Streamer, Agent, and Server.
* **Stage 4: Binary Compilation & Docker Packaging**: Statically compiles Go binaries (`bin/server`, `bin/agent`, `bin/streamer`), uploads them as downloadable workflow artifacts, and verifies multi-stage Docker image builds.


