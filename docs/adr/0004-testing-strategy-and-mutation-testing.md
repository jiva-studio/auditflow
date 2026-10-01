# ADR 0004: Multi-Tier Testing Strategy & Mutation Testing Efficacy

## Status
Accepted

## Context
In an audit and compliance system, software correctness is critical:
- A false negative (missing a rule match) allows unauthorized or non-compliant actions to go undetected.
- A false positive (erroneous popup trigger) creates alert fatigue and disrupts employee workflow.

Traditional software metrics rely on **Code Line Coverage** (e.g. 80–90%). However, academic research and industrial experience consistently show that:
1. **Coverage Does Not Guarantee Correctness**: A test can execute every line of a function without asserting meaningful invariants or boundary conditions (*"Phantom Coverage"*).
2. **AI Test Generation Fragility**: In AI-assisted development, LLMs frequently generate tests with weak or circular assertions (tests that mirror the buggy implementation's assumptions).
3. **Flakiness in E2E Tests**: Poorly isolated integration tests using hardcoded ports or static sleeps introduce non-deterministic test failures in CI/CD pipelines.

## Decision Drivers
- **100% Fault Detection**: Objective mathematical proof that tests catch real logic defects, condition boundary errors, and operator inversions.
- **Elimination of Phantom Coverage**: Guarantee that every line of business logic is strictly asserted.
- **Deterministic Blackbox E2E Validation**: Verification of the full binary pipeline (`Streamer -> Agent -> Server`) against real and synthetic edge-case recordings.
- **Zero Flakiness**: Dynamic ephemeral port allocation and healthcheck polling for concurrent test safety.

## Considered Options
1. **Line Coverage Alone (>80%)**: Quick to measure, but provides a false sense of security and ignores assertion quality.
2. **Manual QA / Exploratory Testing**: Unrepeatable, slow, and cannot prevent regressions during rapid iterations.
3. **Multi-Tier Testing Pyramid with Automated Mutation Testing**: Combines pure unit tests, adapter tests, Blackbox E2E pipelines, and AST-level mutation testing (Gremlins) targeting 100% test efficacy.

## Decision
We adopted a comprehensive **Multi-Tier Testing Pyramid** validated by **Mutation Testing**:

```text
+-------------------------------------+
|       Blackbox E2E Pipelines        |
|    (Streamer -> Agent -> Server)    |
|      Real & Synthetic Datasets      |
+-------------------------------------+
|       Hexagonal Adapter Tests       |
|    (HTTP Handlers, Clients, DB)     |
+-------------------------------------+
|       Pure Domain Unit Tests        |
|    (Rules, State, Hit-Testing)      |
+-------------------------------------+
                  ^
                  | Validated by
+-------------------------------------+
|        AST Mutation Testing         |
|      (Gremlins: 100% Efficacy)      |
+-------------------------------------+
```

### 1. Multi-Tier Testing Pyramid
We structure the automated verification suite into distinct architectural tiers:
- **Pure Domain Unit Tests**: Isolated, in-memory unit tests verifying business state transitions, spatial geometry math, and rule specifications without external I/O or framework dependencies.
- **Hexagonal Adapter Tests**: Adapter tests validating network transport, protocol serialization, HTTP error status mapping, cancellation handling, and concurrent access safety under the Go race detector.
- **Blackbox End-to-End Pipeline Tests**: Integration tests executing compiled service binaries over network boundaries, replaying full telemetry sessions (production and synthetic edge-case streams) and asserting compliance notifications against deterministic golden fixtures.

### 2. AST-Level Mutation Testing Quality Gate
We enforce AST mutation testing as a mandatory quality criterion:
- Automatically injects code mutations (boundary mutations, arithmetic and boolean operator inversions, logical negation).
- Requires 100% Mutation Test Efficacy (zero surviving mutants) across domain and core application packages to mathematically verify assertion quality.

## Academic Research & Industry References
- **Just, R., Jalali, D., Inozemtseva, L., Ernst, M. D., Holmes, R., & Fraser, G. (ACM FSE 2014)**: [Are Mutants a Valid Substitute for Real Faults in Software Testing?](https://homes.cs.washington.edu/~mernst/pubs/mutation-testing-fse2014.pdf) — established empirically that mutation score is significantly better correlated with real fault detection than statement or branch coverage.
- **Jia, Y., & Harman, M. (IEEE TSE 2011)**: [An Analysis and Survey of the Development of Mutation Testing](https://discovery.ucl.ac.uk/id/eprint/1305417/) — comprehensive survey on mutation testing operators, equivalence detection, and test effectiveness.
- **Meta ACH (Automated Compliance Hardening, 2024)**: [Mutation-Guided Test Hardening for Compliance](https://engineering.fb.com/) — industrial application of mutation testing to eliminate test oracle flakiness.
- **GEM: Generate-Execute-Mutate (arXiv 2024)**: [Refining Test Oracles with Mutation Feedback in LLM-Based Software Engineering](https://arxiv.org/abs/2402.16642) — demonstrating that mutation feedback is necessary to harden AI-generated test assertions.

## Agentic & AI Pair-Programming Implications
In AI-assisted pair programming, Mutation Testing acts as an **Uncompromising Quality Gate**:
- When the AI generates code and tests, Gremlins immediately highlights any assertion gaps where code was executed but not rigorously verified.
- Forces the AI agent to write exhaustive, boundary-aware assertions before code is marked complete.
