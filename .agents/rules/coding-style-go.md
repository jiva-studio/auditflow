# Go Coding Style

Standards and conventions for Go code across all modules (`libs/domain`, `modules/server`, `modules/agent`, `modules/streamer`, `tests/e2e`).

---

## 1. Project & Module Structure

```text
assessment/
├── libs/
│   └── domain/         # Pure domain entities, value objects, invariants (0 JSON, 0 I/O)
├── modules/
│   ├── server/         # Audit REST API server (:8080)
│   ├── agent/          # Employee workstation state tracker & rule evaluator
│   └── streamer/       # Replay generator & tick bucketing
└── tests/
    └── e2e/            # End-to-end integration tests
```

- `libs/domain` remains completely pure and stateless: no network, no database, no JSON tags.
- Stateful accumulation, tick dispatching, and HTTP transports live inside their respective `modules/`.
- Local cross-module dependencies use Go `replace` directives in `go.mod`.

---

## 2. Idiomatic Go Standards

### Context
- `ctx context.Context` is the first parameter of any function doing I/O, network calls, or blocking loops. Never store context in a struct.
- Cancellation must be observed inside long-running loops (`select { case <-ctx.Done(): ... }`).
- Tests use `t.Context()`.

### Errors
- Errors are never ignored (`_ = fn()` is forbidden).
- Wrap errors with `%w` providing contextual info: `fmt.Errorf("parse tick %d: %w", tick, err)`.
- Errors are either handled and logged, or returned up the stack — never both.
- Compare sentinels with `errors.Is` and custom error types with `errors.As`.
- `panic` is strictly reserved for unrecoverable programmer bugs, never for external input.

### Interfaces
- Defined where consumed (consumer side), not producer side.
- Keep interfaces small (1–3 methods). Accept interfaces, return concrete structs.

### Constructors & State
- `New*` constructors enforce and validate all required dependencies and business invariants.
- No mutable global state; no `init()` functions performing I/O.

### Concurrency & Synchronization
- Every goroutine has an owner and lifecycle tied to `context.Context`.
- Shared state is protected with `sync.RWMutex` or channels.
- Never use `time.Sleep` as synchronization in production or test code.
- Always run tests with race detection: `go test -race ./...`.

### Resource Management
- Always use `defer` for closing streams, readers, HTTP bodies, and mutex unlocks immediately after acquisition.

### Naming
- Avoid stuttering: `session.Metadata` (not `session.SessionMetadata`).
- Functions use imperative verb phrases (`EvaluateRules`, `HitTest`, `ResolveClick`).

---

## 3. Testing Standards
- Unit tests live next to the code being tested (`*_test.go`).
- Tests must be deterministic, isolated, and take milliseconds.
- Test failure messages clearly state expectation vs reality: `got %v, want %v`.
