# Goal
Implement a complete, testable software system from specifications in
`specs.md` or the `specs/` directory.
Output code in `/src`, tests in `/tests`.

---

# Execution Model
Operate in strict phases:

1. SPEC VALIDATION
2. DESIGN
3. IMPLEMENTATION
4. TESTING
5. FINALIZATION

Do not proceed unless the current phase passes all checks.
Process is self-healing: issues discovered later must trigger controlled
return to SPEC VALIDATION.

To keep track of progress maintain the file `progress.md`.
format:
  <module> | <status> | <src> | <tests>
  statuses: pending | implemented | tested | integrated



---

# Phase 1 — Spec Validation

A specification is valid iff:

- `specs.md` or `specs/` exists
- All referenced files exist
- No placeholders (e.g., “TBD”, “etc.”)
- No undefined types or functions
- No conflicting requirements

If any check fails → STOP and report:
- missing/invalid item
- file + section
- minimal fix

---

# Phase 2 — DESIGN (MANDATORY)

Always produce a full design in `/design`.

## Inputs
- `specs.md` OR `specs/`

## Outputs

- `design/architecture.md`
- `design/api.md`
- `design/global.md`
- `design/use-cases.md`
- `design/modules/*.md` (one per module)
- `design/schema.sql` commented DDL if persistence is required

## Architecture selection (deterministic)

- persistence → layered
- request/response → MVC/REST
- streaming → pipeline
- else → single-module

## API requirements

Each API entry in `design/api.md` MUST include:
- name
- module path
- full typed signature
- behavior (deterministic)
- error cases
- side effects
- idempotency

## Module specs

Each file in `design/modules/*.md` must include:
- target source file path
- owned APIs
- internal logic
- dependencies ONLY via `design/api.md`

## Global spec

The file `design/global.md` must include:
- language + version
- dependency policy
- error handling strategy
- any other specified global or non-functional requirements

## Validation

Design must be sufficient to implement the system WITHOUT consulting original specs.

If incomplete → FIX design before proceeding.

---

# Phase 3 — IMPLEMENTATION

## Rules

- Implement each module independently
- Allowed inputs:
  - its module spec
  - `design/api.md`
  - `design/global.md`
  - `design/architecture.md`
  - `design/schema.sql`

- Ignore all other files

## Determinism

- Do not infer beyond design
- Do not add features
- Minimal dependencies only if specified
- Initialize database using `schema.sql`.

## Self-healing

If during implementation:
- missing API
- ambiguous behavior
- incorrect abstraction

Then:
1. STOP implementation
2. Update affected design files
3. Resume implementation for all changed files under `design/`

---

# Phase 4 — TESTING

Requirements:

- Unit tests for all non-trivial APIs
- Integration tests for all use cases

Tests must:
- be deterministic
- require no manual input
- run via a single command

---

# Phase 5 — FINALIZATION

Produce:

- `implementation-notes.md` documenting all implementation-defined choices.
- `README.md` with instructions on compiling, testing, deploying, running,
    using, and monitoring (if needed) the created  software.

---

# Constraints

- No cross-module access except via API
- No undefined behavior
- No external assumptions
- No manual fixes required

---

# Failure Conditions (STOP)

Any of the following that cannot be fixed by going to the DESIGN
phase over three iterations.

- missing file
- incomplete API
- ambiguous behavior
- undefined types
- conflicting specs

Report minimally and exit.

---

# Ignore
Editor files (e.g. `.*.swp`, `Session.vim`, files in `.gitignore`)
