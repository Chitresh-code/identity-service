# Agent Guidelines

These are the coding conventions for this service. Follow them at all times — for every
change, not just the first one. Collaboration workflow (branches, commits, work items,
wiki updates, PRs) lives in `CONTRIBUTING.md`; follow that at all times too.

Deliver small, idiomatic, production-ready changes. Trace the affected call path before
editing; fix shared root causes instead of patching individual callers.

## Commands

- Format changed Go files: `gofmt -w <files>`
- Run tests: `go test ./...`
- Check common mistakes: `go vet ./...`
- Build all packages: `go build ./...`
- Check race-sensitive changes: `go test -race ./...`
- Update module metadata only after dependency changes: `go mod tidy`

Run the narrowest relevant test first. Run `go test ./...` before handing off a non-trivial change when practical.

## Project knowledge

- **Language:** Go; use the version declared by `go.mod`.
- **Dependencies:** Standard library first. Reuse existing modules before adding one.
- **Stack:** Echo (router), GORM (ORM), golang-migrate (schema), Postgres, Auth0 (human/admin login).
- **Layout:** Read `go.mod`, `README.md`, and the nearest package tests before changing a package. Follow the repository's existing `cmd/`, `internal/`, `pkg/`, and test layout where present.

For a new service, keep dependencies flowing inward and organize packages by domain:

```
cmd/identity-service/main.go   # wiring and process startup only
internal/<domain>/             # domain behavior and its tests
internal/http/                 # transport handlers and middleware
internal/store/                # persistence implementations
internal/config/               # validated configuration loading
```

Use `internal/` by default. Create `pkg/` only for a deliberately supported, reusable public package. Keep `main` limited to composition; do not put business logic in handlers or startup wiring.

## Go standards

- Use `gofmt`; do not manually align formatting.
- Prefer small functions, concrete types, and package-level functions over abstractions with one implementation.
- Accept interfaces at the consumer boundary; return concrete types unless an interface is required.
- Pass `context.Context` as the first parameter for request-scoped or cancellable work. Do not store contexts in structs.
- Return errors with useful operation context: `fmt.Errorf("load user %q: %w", id, err)`. Check errors; do not discard them.
- Make zero values useful when it is natural. Avoid `init()` except when unavoidable.
- Keep exported identifiers documented and names concise: `Client`, `NewClient`, `Fetch`, not `UserDataFetcherService`.
- Validate untrusted input at API, CLI, and persistence boundaries. Use timeouts and cancellation for network calls.
- Use table-driven tests for multiple cases; test observable behavior, including error paths.

## Concurrency and resources

- Start a goroutine only when its owner, cancellation path, and completion are clear. Never start unbounded goroutines per request or loop iteration.
- Propagate cancellation with `ctx`; use a `sync.WaitGroup` or an existing project pattern to wait for owned work during shutdown.
- Protect shared mutable state with a mutex or a channel. Do not copy structs containing a mutex.
- Close resources at the layer that opens them: response bodies, files, rows, and listeners. Check `rows.Err()` after iteration.
- Run `go test -race ./...` after changing goroutines, channels, shared state, or lifecycle code.

## HTTP and persistence

- Keep handlers thin: decode and validate input, call a domain service, then encode a response. Put reusable error mapping in one place.
- Set explicit response status codes and do not leak internal error details to clients. Log errors with request-relevant context, never secrets.
- Bound request bodies and apply server/client timeouts. Outbound requests must use the caller's context.
- Make multi-write operations transactional. Do not commit or roll back transactions in helpers that did not open them.
- Treat schema migrations as reviewed, forward-only operational changes unless the project documents another policy.

```go
// Good: handler owns HTTP concerns; the service owns behavior.
func (h *Handler) getUser(c echo.Context) error {
    user, err := h.users.User(c.Request().Context(), c.Param("id"))
    if err != nil {
        return h.writeError(c, err)
    }
    return c.JSON(http.StatusOK, user)
}
```

```go
// Good: validates input, propagates cancellation, and preserves the cause.
func (s *Service) User(ctx context.Context, id string) (User, error) {
    if id == "" {
        return User{}, errors.New("user id is required")
    }

    user, err := s.store.User(ctx, id)
    if err != nil {
        return User{}, fmt.Errorf("get user %q: %w", id, err)
    }
    return user, nil
}
```

```go
func TestParsePort(t *testing.T) {
    for _, tt := range []struct {
        name  string
        input string
        want  int
        ok    bool
    }{
        {"valid", "8080", 8080, true},
        {"not a number", "http", 0, false},
    } {
        t.Run(tt.name, func(t *testing.T) {
            got, err := parsePort(tt.input)
            if (err == nil) != tt.ok || got != tt.want {
                t.Fatalf("parsePort(%q) = %d, %v", tt.input, got, err)
            }
        })
    }
}
```

Use `t.TempDir`, `t.Cleanup`, and `httptest` for isolated filesystem, cleanup, and HTTP tests. Avoid timing-based tests; use channels or controllable clocks. Call `t.Parallel` only when the test and its dependencies are isolated.

## Boundaries

- **Always:** preserve backwards compatibility unless the task explicitly changes it; add or update focused tests for behavior changes; format touched Go files.
- **Ask first:** adding or upgrading dependencies, changing public APIs, database migrations, authentication/authorization, CI/CD, deployment, or production configuration.
- **Never:** edit vendored code, generated files without running their documented generator, secrets, or credentials; use `panic` for expected runtime errors; remove or weaken failing tests to make checks pass.
