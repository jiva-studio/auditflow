---
name: coder
description: Implementation workflow for backend features, bug fixes, and domain logic. Ensures TDD, pure domain boundaries, zero regressions, and passes all linters.
---

# Coder Workflow

Standard implementation procedure for features and modules.

## Phase 1: Context & Constraints
1. Read the domain models in `libs/domain/` to understand existing entities, value objects, and invariants.
2. Confirm architectural boundaries:
   - `libs/domain/`: Pure business logic, 0 JSON, 0 I/O.
   - `modules/`: Service logic, transports, stateful accumulators, HTTP handlers.
3. Review `.agents/rules/coding-style-backend.md` and `.agents/rules/comments.md`.

## Phase 2: Design & TDD
1. **Red**: Write a unit test asserting the expected behavior or edge case.
2. **Green**: Implement the minimal, clean, robust logic satisfying the test.
3. **Refactor**: Keep methods small, flat (guard clauses, no deep nesting), and cyclomatic complexity low.

## Phase 3: Verification Loop
1. Run local targeted module tests:
   ```bash
   make <module>_test
   make <module>_lint
   ```
2. Verify race safety:
   ```bash
   go test -race ./...
   ```
3. Ensure `gofmt` compliance and clean comments.
