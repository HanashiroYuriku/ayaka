# Testing Guide

## Running tests

```bash
make test        # all unit tests
make cover       # tests + coverage summary + coverage.html
make test-race   # with the race detector (needs CGO; always runs in CI)
make check       # fmt + lint + test — run before pushing
```

Run a single package or test:

```bash
go test ./internal/core/service/...
go test ./internal/core/service/ -run TestAuthServiceSuite/TestLogin_Success -v
```

Tests need **no database or network**: repositories use in-memory SQLite, everything else uses mocks.

## What is tested where

| Layer | Location | Style | Dependencies |
| :--- | :--- | :--- | :--- |
| Config | `config/config_test.go` | Load a temp YAML + `t.Setenv` | none |
| Core errors / models | `internal/core/apperror`, `internal/core/model` | Plain table tests | none |
| Services (use cases) | `internal/core/service/*_test.go` | testify `suite` | mocks for every port |
| Repositories | `internal/adapter/repository/*_test.go` | `BaseRepoSuite` | fresh in-memory SQLite per test |
| Token provider | `internal/adapter/token/jwt_test.go` | Plain tests | none |
| Handlers | `internal/delivery/http/*_test.go` | `fiber.App.Test` | mocked services + validator |
| Middleware | `internal/middleware/middleware_test.go` | `fiber.App.Test` | mocked token provider |
| Validator | `pkg/validator/validator_test.go` | Plain tests | in-memory SQLite |

## Test helpers (`internal/testingutils`)

| Helper | Purpose |
| :--- | :--- |
| `NewTestApp()` | Fiber app with the **production** error handler, request ID, request context and recover middleware (silent logger) |
| `FakeAuth(userID, role)` | Authenticates every request; override the user with the `X-Test-User-ID` header |
| `MakeJSONRequest(method, path, body, opts...)` | Builds a JSON request (`body` may be a struct, map or raw `[]byte`); sets `X-Request-Id: TEST-REQUEST-ID` |
| `WithAuth(token)`, `WithRequestID(id)` | Request options |
| `DecodeResponse(t, resp)` | Decodes the standard JSON envelope |

## Mocks (`internal/testingutils/mocks`)

All mocks live in one package and have compile-time interface checks, so they break loudly when an interface changes.

| Mock | Implements |
| :--- | :--- |
| `UserRepo` | `port.UserRepository` |
| `Hasher` | `port.PasswordHasher` |
| `TokenProvider` | `port.TokenProvider` |
| `EmailSender` | `port.EmailSender` |
| `Validator` | `validator.Validator` |
| `AuthService`, `UserService` | service interfaces (for handler tests) |
| `FakeTx` | `port.TxManager` — a **fake**, not a mock: runs the function directly and counts calls (`tx.Count`) |

When you add a port or service, add its mock to `mocks.go` together with a `var _ port.X = (*X)(nil)` assertion.

## Writing tests — conventions

**Service test** — mock the ports, assert behaviour and errors by *kind*:

```go
func (s *AuthServiceSuite) TestLogin_WrongPassword_ReturnsGenericError() {
	s.users.On("FindByEmail", s.ctx, "riku@mail.com").Return(&entity.User{Password: "HASHED"}, nil).Once()
	s.hasher.On("Compare", "HASHED", "wrong").Return(errors.New("mismatch")).Once()

	_, err := s.svc.Login(s.ctx, model.LoginInput{Email: "riku@mail.com", Password: "wrong"})

	s.ErrorIs(err, service.ErrInvalidCredentials)
}
```

**Repository test** — embed `BaseRepoSuite` with the models you need; each test gets a clean database:

```go
func TestPostRepoSuite(t *testing.T) {
	suite.Run(t, &PostRepoSuite{BaseRepoSuite: NewBaseRepoSuite(&entity.Post{})})
}
```

**Handler test** — use `NewTestApp()` so status codes and the JSON envelope match production:

```go
s.app = testingutils.NewTestApp()
s.app.Get("/users/me", testingutils.FakeAuth("USR-1", entity.RoleUser), h.GetMe)

resp, _ := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodGet, "/users/me", nil))
body := testingutils.DecodeResponse(s.T(), resp)
```

Guidelines:
* Name tests `Test<Method>_<Scenario>` (e.g. `TestRegister_Conflict_Returns409`).
* Assert errors with `ErrorIs` against `apperror` sentinels, not with string comparison.
* Cover the success path, each expected error kind, and "unexpected error becomes 500".
* Call `AssertExpectations` in `TearDownTest`, and `AssertNotCalled` for steps that must be skipped.
* Keep tests independent — no shared state between tests, no reliance on execution order.

## Limitations

SQLite does not behave exactly like PostgreSQL (partial indexes, JSONB, locking, `ILIKE`, ...). For queries that rely on Postgres features, add integration tests against a real database (e.g. [testcontainers-go](https://golang.testcontainers.org/)) behind a build tag such as `//go:build integration`, and run them with `go test -tags=integration ./...`.

---

## Manual Test Case Template

Use this table to document manual / QA test scenarios per module.

### **Module: [Module Name]**

| ID | Test Scenario | Test Steps | Expected Result | Status |
| :--- | :--- | :--- | :--- | :--- |
| **Test-ID** | Test Name | Testing procedures | Testing expected result | [OK/X] |

### Example — **Module: Auth**

| ID | Test Scenario | Test Steps | Expected Result | Status |
| :--- | :--- | :--- | :--- | :--- |
| **AUTH-01** | Register with valid data | `POST /api/v1/auth/register` with valid username, email, password | `201`, `status: SUCCESS`, user returned without password | [ ] |
| **AUTH-02** | Register with existing email | Repeat AUTH-01 with the same email | `409`, `errorField.email = "Email already exists"` | [ ] |
| **AUTH-03** | Register with weak password | Password `weak` | `422`, `errorField.password` present | [ ] |
| **AUTH-04** | Login with wrong password | `POST /api/v1/auth/login` with a wrong password | `401`, message `invalid email or password` | [ ] |
| **AUTH-05** | Access profile without token | `GET /api/v1/users/me` without `Authorization` | `401` | [ ] |
| **AUTH-06** | Use refresh token as access token | `GET /api/v1/users/me` with the refresh token | `401` | [ ] |
| **AUTH-07** | Rate limit | Call `/api/v1/auth/login` more than `RATE_LIMIT_AUTH_MAX` times within the window | `429` | [ ] |
