# Repository Guidelines

## Project Structure & Module Organization

This repository is a monorepo for several gRPC services. Main code lives under `services/`: Go services in `services/auth`, `services/users`, `services/metadata`, and `services/storage`; the Python ReBAC service in `services/authz`; and the Rust quota service in `services/quota`. Shared protobuf sources are in `shared/api`, generated clients/stubs live in `shared/pkg/go` and `shared/pkg/py`, infrastructure lives in `infra/`, and cross-service docs are in `docs/`.

Go services follow the same layout: `cmd/server` for server entrypoints and `internal/` for app, config, handlers, services, and repositories. Additional maintenance binaries may live under `cmd/<name>`. AuthZ keeps runtime code in `entrypoints/` and `internal/`, with tests in `services/authz/tests/{unit,integration}`. Quota keeps Rust code in `services/quota/src`.

## Build, Test, and Development Commands

- `make up-services`: start local infrastructure and enabled services with Docker Compose.
- `make up-e2e`: start the dedicated local PostgreSQL container for Go integration tests.
- `make up-observability`: start Grafana, Prometheus, Jaeger, and related tooling.
- `make down` / `make down-volumes`: stop containers; the latter also removes persisted data.
- `make generate`: regenerate shared Go and Python protobuf bindings from `shared/api`.
- `go test ./...` from `services/auth`, `services/users`, `services/metadata`, or `services/storage`: run tests for one Go service module.
- `make test-metadata-integration`: run Metadata integration tests against the dedicated e2e PostgreSQL profile.
- `make test-metadata-integration-local`: start the dedicated e2e PostgreSQL container and then run Metadata integration tests.
- `python3 -m pytest tests/unit -v` from `services/authz`: run AuthZ unit tests.
- `python3 -m pytest tests/integration -v -m integration` from `services/authz`: run Neo4j-backed integration tests.
- `cargo test -p quota-service` from the repository root: run Quota tests.

## Coding Style & Naming Conventions

Use standard Go formatting (`gofmt`) and keep package names lowercase. Prefer the existing service layering (`handler`, `service`, `repository`) and place binaries only under `cmd/`. In Python, use 4-space indentation, `snake_case` for functions/modules, and keep gRPC service classes descriptive, for example `PermissionServiceServicer`.

Do not hand-edit generated protobuf files under `shared/pkg/`; regenerate them instead.

## Testing Guidelines

Keep Go tests close to the package they verify, using `*_test.go`. Current examples include handler and service tests under `services/users/internal/.../tests`. For AuthZ, use `tests/unit/test_*.py` for isolated logic and `tests/integration/test_*.py` for Neo4j-dependent flows. Mark integration tests with `@pytest.mark.integration`.

### Documentation Discipline

Documentation must be updated in the same change whenever the code changes the developer workflow or runtime expectations.

Update documentation when the change affects:

- project startup or local run flow
- Docker Compose usage, build contexts, or service dependencies
- ports, env vars, credentials, or required local tools
- `Makefile` commands, test commands, or generation commands
- repository structure that is referenced from onboarding or README
- first-run experience for a new contributor

At minimum, check whether the change also requires updates to:

- `README.md`
- `GETTING_STARTED.md`
- `AGENTS.md`
- service-level README files under `services/*/README.md`

If a document is now partially outdated but a full rewrite is unnecessary, prefer a small accurate update over leaving stale instructions in place.

### Go Test Style

The rules below apply specifically to Go unit tests:

- prefer separate test files per service method or handler method, for example `get_test.go`, `delete_test.go`, `update_password_test.go`
- prefer table-driven tests with explicit `args`, expected result, expected error, and mock builder fields
- create a fresh `minimock.Controller` inside each `t.Run`; do not share one controller across table cases
- use repository and infrastructure mocks from `services/*/pkg/mocks`; do not replace them with ad hoc fakes unless there is a clear reason
- do not branch test setup by `tt.name`; each case should fully describe its own mock expectations
- assert behaviour, arguments, and error propagation; do not test internals of `bcrypt`, logging, tracing, or generated code
- when validation fails, explicitly verify that repository methods are not called
- keep happy-path and error-path cases separate and readable rather than collapsing everything into one oversized test
- for service-layer unit tests, avoid `t.Parallel()` by default unless mocks and shared state are clearly isolated and the test remains deterministic
- when a constructor requires unrelated dependencies, pass mocks/stubs but keep assertions focused only on the method under test

## Commit & Pull Request Guidelines

Recent history uses short conventional-style messages such as `feat: ...` and `fix(proto): ...`. Follow `type(scope): summary` where possible. Keep PRs focused, describe service-level impact, list any new env vars or Docker requirements, and link the issue or task. For API changes, mention regenerated protobuf outputs and affected services explicitly.

### Service README Structure and Preservation

Every implemented service must have `services/<name>/README.md`. Use the same
level-two sections, in this order: purpose and capabilities, architecture and
dependencies, project structure, API, data and main flows, configuration, startup,
usage examples, observability and troubleshooting, development and tests,
limitations and future work, related documents. The current service READMEs use
Russian headings; preserve those headings for consistency. Root, documentation
index, and test-kit READMEs serve different purposes and need not mimic a service.

When correcting facts, preserve useful explanations, runnable examples, design
alternatives, and test guidance. Fix the inaccurate passage in place; do not
replace a detailed document with only a list of limitations. Mark proposed flows
and historical snapshots visibly. A general warning at the top does not fix an
incorrect API example or a contradictory diagram inside the document.

Keep shared development rules in this file. CLAUDE.md files add architecture,
service-specific navigation, and testing context; avoid maintaining conflicting
copies of commands/configuration. Link to service READMEs for full runbooks.
