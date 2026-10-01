# AuditFlow — Real-Time Telemetry & Contextual Notification Platform

AuditFlow is a distributed, real-time desktop telemetry and contextual notification platform. It ingests desktop activity streams (active windows, OCR text, mouse clicks, clipboard contents, and process events), evaluates triggering rules locally on employee endpoints with sub-millisecond latency, and aggregates recorded popup notifications into a central repository.

---

## 1. How to Run

### Quick Start with Docker Compose (Recommended)

To run the complete system (Central Server and Replay Runner processing all `./data/*.tar.gz` recordings against `./rules.json`) from scratch:

```bash
docker compose up --build
```

#### Fast-Forward Execution (Zero-Delay Replay)
By default, the replay streamer respects the simulated workstation tick cadence (`TICK_MS=10`). To execute the full replay instantly at maximum throughput:

```bash
TICK_MS=0 docker compose up --abort-on-container-exit --exit-code-from runner
```

### Querying Popup Audit Results

Once the containers are running, the Central Audit Server exposes a REST API on port `8080`:

```bash
# Retrieve all recorded popup events
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
│   │   ├── server/                  # Central popup store & query API
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
│       │   ├── audit/               # Popup aggregate and query filters
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

1. **[ADR-0001: Clean and Hexagonal Architecture](docs/adr/0001-clean-and-hexagonal-architecture.md)** — Combines Clean Architecture dependency inversion and Hexagonal Ports & Adapters to decouple domain logic from transports and persistence.
2. **[ADR-0002: Domain-Driven Design](docs/adr/0002-domain-driven-design.md)** — Models workstation state as an aggregate root (`desktop.State`) partitioned into cohesive facets (`SpatialFacet`, `WindowFacet`, `ClipboardFacet`).
3. **[ADR-0003: Specification Pattern](docs/adr/0003-specification-pattern.md)** — Expresses dynamic notification rules as composable predicates (`TextSpec`, `WindowSpec`, `ClipboardSpec`, `ClickSpec`).
4. **[ADR-0004: Protocol Buffers](docs/adr/0004-protocol-buffers.md)** — Schema-first binary contracts (Protobuf v3) for high-frequency inter-service streaming over HTTP.
5. **[ADR-0005: Testing Strategy](docs/adr/0005-testing-strategy.md)** — Multi-tier testing pyramid combining pure domain unit tests, adapter tests with race detection, and blackbox E2E replay pipelines.
6. **[ADR-0006: Mutation Testing](docs/adr/0006-mutation-testing.md)** — AST-level mutation testing via Gremlins to eliminate phantom code coverage and verify assertion quality.
7. **[ADR-0007: Architectural Fitness Functions](docs/adr/0007-architectural-fitness-functions.md)** — Automated AST guards (`no_io.py`, `no_json.py`) and static complexity lint gates in CI.

---

## 5. Continuous Integration (CI)

The automated [GitHub Actions CI](.github/workflows/ci.yml) workflow enforces quality gates on every Pull Request and commit to `main`:

* **Stage 1: Quality Gates & Fitness Functions**: Verifies code formatting with `gofmt`, executes Python AST domain purity guards (`no_json.py`, `no_io.py`), and runs `golangci-lint` across all Go modules.
* **Stage 2: Unit & Integration Tests**: Runs test suites with Go race detector (`-race`) and generates aggregated coverage profiles.
* **Stage 3: End-to-End System Tests**: Executes the full integration test pipeline across Streamer, Agent, and Server.
* **Stage 4: Binary Compilation & Docker Packaging**: Statically compiles Go binaries (`bin/server`, `bin/agent`, `bin/streamer`), uploads them as downloadable workflow artifacts, and verifies multi-stage Docker image builds.

---

## 6. Part 2: Production Scaling & Team Rollout Plan

The complete plan for shipping AuditFlow to thousands of employees across enterprise clients is documented in detail in **[TODO.md](TODO.md)**. Key highlights include:

* **Target Production Architecture**: Stateless Go ingestion gateways, Protobuf contracts, Kafka/Redpanda broker partitioning by `employee_id`, and dual storage (ClickHouse for audit logs + PostgreSQL for configs).
* **Workload & Capacity Math**: Detailed throughput and bandwidth modeling proving edge rule evaluation comfortably scales to 10,000+ endpoints with minimal backend infrastructure.
* **Endpoint Hardening & Privacy**: Native OS services (systemd, Windows Service, LaunchDaemon), offline SQLite (WAL) buffering, and client-side PII regex/Luhn sanitization.
* **Canary Deployment Rings**: 4-stage rollout strategy (Internal -> 5% -> 25% -> 100%) with automated crash-rate rollbacks and signed enterprise packages (MSI/PKG/DEB).
* **Team Workstreams & Ownership**: Cross-functional team structure spanning 5 dedicated tracks (*Agent Core*, *Ingestion Platform*, *Identity & Auth*, *Rules & Detection*, and *Product UI*).



