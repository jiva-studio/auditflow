# ADR 0006: Mutation Testing

## Status
Accepted

## Context
Code coverage metrics do not guarantee that tests assert correct behavior. Tests can execute code paths without verifying outcomes, leaving logic bugs undetected.

## Decision
We adopted **Mutation Testing** using Gremlins for core domain and application packages:

1. **Automated Mutation Injection**: Inverts conditionals, boolean logic, and boundary operators at the AST level.
2. **Mutant Verification**: Tests run against each mutant; surviving mutants highlight assertion gaps.
3. **Assertion Hardening**: Target 100% mutant kill rate across core domain logic.

## Consequences

### Positive
- Detects gaps where code is executed but not meaningfully asserted.
- Validates boundary conditions in geometry math and rule evaluation.

### Negative
- Mutation runs are slow and intended for local on-demand runs and quality gates.

## Agentic Engineering Implications
Mutation testing prevents autonomous agents from generating "phantom coverage":
- **Eliminating Circular Assertions**: LLM agents often write tests that execute code without asserting meaningful boundary conditions; mutation testing forces agents to harden test assertions against inverted conditions.
- **Mathematical Oracle Verification**: Surviving mutants provide exact line numbers and mutated tokens for agents to fix assertions autonomously.

## Academic & Industry Research References
- **Just, R., Jalali, D., Inozemtseva, L., Ernst, M. D., Holmes, R., & Fraser, G. (ACM FSE 2014)**: [Are Mutants a Valid Substitute for Real Faults in Software Testing?](https://homes.cs.washington.edu/~mernst/pubs/mutation-testing-fse2014.pdf) — established empirically that mutation score is significantly better correlated with real fault detection than statement or branch coverage.
- **Jia, Y., & Harman, M. (IEEE TSE 2011)**: [An Analysis and Survey of the Development of Mutation Testing](https://discovery.ucl.ac.uk/id/eprint/1305417/) — comprehensive survey on mutation testing operators, equivalence detection, and test effectiveness.
- **GEM: Generate-Execute-Mutate (arXiv 2024)**: [Refining Test Oracles with Mutation Feedback in LLM-Based Software Engineering](https://arxiv.org/abs/2402.16642) — demonstrating that mutation feedback is necessary to harden AI-generated test assertions.
