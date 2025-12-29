# Code Navigation Prompts

Examples of navigating code using symbols and call graphs.

## Symbol Navigation

### List Functions

```
User: Show me all functions in the auth package

AI: [Uses get_symbols tool]

    Functions in internal/auth/:

    ┌────────────────────────┬─────────────────┬────────────┬─────────────────────────────┐
    │ Name                   │ File            │ Lines      │ Signature                   │
    ├────────────────────────┼─────────────────┼────────────┼─────────────────────────────┤
    │ NewAuthService         │ service.go      │ 23-35      │ func(cfg Config) *Service   │
    │ Authenticate           │ service.go      │ 37-65      │ func(user, pass string) (*Token, error) │
    │ ValidateToken          │ jwt.go          │ 12-45      │ func(token string) (*Claims, error) │
    │ RefreshToken           │ jwt.go          │ 47-78      │ func(token string) (string, error) │
    │ HashPassword           │ password.go     │ 10-25      │ func(pass string) (string, error) │
    │ VerifyPassword         │ password.go     │ 27-42      │ func(hash, pass string) bool │
    │ AuthMiddleware         │ middleware.go   │ 15-45      │ func(next http.Handler) http.Handler │
    └────────────────────────┴─────────────────┴────────────┴─────────────────────────────┘

    Types:
    - Service (service.go:10)
    - Config (config.go:5)
    - Claims (jwt.go:8)
    - Token (types.go:12)
```

### Find by Type

```
User: List all interfaces in the project

AI: [Uses get_symbols with kind=interface]

    Interfaces found:

    pkg/provider/
    ├── EmbeddingProvider (embedding.go:12)
    │   Methods: Name(), Embed(), Dimensions(), MaxBatchSize()
    │
    ├── ChunkingStrategy (chunking.go:8)
    │   Methods: Name(), Chunk(), SupportedLanguages()
    │
    ├── VectorStore (vectorstore.go:15)
    │   Methods: Init(), Close(), StoreChunks(), Search()...
    │
    └── Reranker (reranker.go:10)
        Methods: Name(), Rerank(), MaxDocuments()

    internal/
    ├── OrderProcessor (orders/processor.go:20)
    ├── PaymentGateway (payments/gateway.go:15)
    └── CacheBackend (cache/backend.go:8)
```

### Find Symbol Definition

```
User: Where is the User struct defined?

AI: [Uses get_symbols tool]

    Found "User" definitions:

    1. Primary Definition:
       internal/models/user.go:15-35

       type User struct {
           ID        int64     `json:"id"`
           Email     string    `json:"email"`
           Name      string    `json:"name"`
           CreatedAt time.Time `json:"created_at"`
           UpdatedAt time.Time `json:"updated_at"`
       }

    2. API Response Type:
       internal/api/types.go:45-52

       type User struct {
           ID    int64  `json:"id"`
           Email string `json:"email"`
           Name  string `json:"name"`
       }
       // Note: Subset of internal User for API responses

    3. Proto Definition:
       proto/user.proto:12-18
       message User { ... }
```

## Call Graph Navigation

### Find Callers

```
User: What calls the ProcessPayment function?

AI: [Uses get_callers tool]

    Callers of ProcessPayment (internal/payments/processor.go:45):

    Direct Callers (3):
    ┌─────────────────────────────────────┬────────────────────────────┬───────┐
    │ Caller                              │ Location                   │ Line  │
    ├─────────────────────────────────────┼────────────────────────────┼───────┤
    │ HandleCheckout                      │ internal/handlers/cart.go  │ 89    │
    │ ProcessSubscriptionRenewal          │ internal/billing/renew.go  │ 56    │
    │ RetryFailedPayments                 │ internal/jobs/payments.go  │ 34    │
    └─────────────────────────────────────┴────────────────────────────┴───────┘

    Call Chain (up to 3 levels):
    ├── HandleCheckout
    │   └── Called by: HTTP POST /api/checkout
    │
    ├── ProcessSubscriptionRenewal
    │   └── Called by: SubscriptionCronJob
    │       └── Called by: main.setupCronJobs
    │
    └── RetryFailedPayments
        └── Called by: PaymentRetryWorker
            └── Called by: main.startWorkers
```

### Find Callees

```
User: What does HandleLogin call?

AI: [Uses get_callees tool]

    Functions called by HandleLogin (internal/handlers/auth.go:34):

    Direct Calls:
    ┌─────────────────────────┬────────────────────────────────┬──────────┐
    │ Function                │ Package                        │ Line     │
    ├─────────────────────────┼────────────────────────────────┼──────────┤
    │ ParseLoginRequest       │ internal/handlers/auth         │ 36       │
    │ Authenticate            │ internal/auth/service          │ 42       │
    │ GenerateToken           │ internal/auth/jwt              │ 48       │
    │ SetCookie               │ net/http                       │ 52       │
    │ JSON                    │ github.com/labstack/echo       │ 55       │
    │ Error                   │ github.com/labstack/echo       │ 40, 45   │
    └─────────────────────────┴────────────────────────────────┴──────────┘

    Call Tree:
    HandleLogin
    ├── ParseLoginRequest
    │   └── json.Unmarshal (stdlib)
    ├── Authenticate
    │   ├── FindUserByEmail (db query)
    │   ├── VerifyPassword
    │   │   └── bcrypt.CompareHashAndPassword (stdlib)
    │   └── UpdateLastLogin (db update)
    ├── GenerateToken
    │   └── jwt.NewWithClaims (external lib)
    └── SetCookie / JSON (response)
```

### Trace Full Call Path

```
User: Show the call path from main to database queries

AI: [Uses get_callees recursively]

    Call Paths from main() to database:

    Path 1: HTTP Request Flow
    main
    └── setupServer
        └── RegisterRoutes
            └── HandleGetUser
                └── userService.GetByID
                    └── userRepo.FindByID
                        └── db.QueryRow ← DATABASE

    Path 2: Background Job Flow
    main
    └── startWorkers
        └── OrderProcessor.Run
            └── processOrder
                └── orderRepo.UpdateStatus
                    └── db.Exec ← DATABASE

    Path 3: Cache Miss Flow
    main
    └── setupServer
        └── cacheMiddleware
            └── cache.Get (miss)
                └── loadFromDB
                    └── db.Query ← DATABASE

    Database Entry Points:
    - 12 functions directly call database
    - All go through internal/db/repository.go
    - Connection pool managed by internal/db/pool.go
```

### Find Entry Points

```
User: What are the main entry points to this application?

AI: [Uses get_symbols and analyzes call graph]

    Application Entry Points:

    1. HTTP Endpoints (internal/api/routes.go)
       ├── GET  /api/users        → HandleListUsers
       ├── GET  /api/users/:id    → HandleGetUser
       ├── POST /api/users        → HandleCreateUser
       ├── POST /api/auth/login   → HandleLogin
       ├── POST /api/auth/logout  → HandleLogout
       └── ... (15 more endpoints)

    2. CLI Commands (cmd/)
       ├── server   → cmd/server/main.go    (starts HTTP server)
       ├── migrate  → cmd/migrate/main.go   (database migrations)
       ├── worker   → cmd/worker/main.go    (background jobs)
       └── seed     → cmd/seed/main.go      (seed test data)

    3. Background Jobs (internal/jobs/)
       ├── PaymentRetryJob     → runs every 5 minutes
       ├── CleanupExpiredJob   → runs daily
       └── ReportGeneratorJob  → runs weekly

    4. Event Handlers (internal/events/)
       ├── OnUserCreated      → sends welcome email
       ├── OnOrderCompleted   → updates inventory
       └── OnPaymentFailed    → notifies support
```

## Impact Analysis

### What Would Break?

```
User: If I change the User struct, what would be affected?

AI: [Uses get_callers and analyzes dependencies]

    Impact Analysis for User (internal/models/user.go)

    Direct Dependencies (18 files):
    ├── internal/handlers/     (5 files) - HTTP handlers
    ├── internal/services/     (3 files) - Business logic
    ├── internal/repository/   (2 files) - Database layer
    ├── internal/api/          (4 files) - API types
    └── internal/events/       (4 files) - Event handlers

    Functions Using User:
    ┌─────────────────────────────┬──────────────────────────────┐
    │ Function                    │ Usage                        │
    ├─────────────────────────────┼──────────────────────────────┤
    │ userRepo.Create             │ INSERT (all fields)          │
    │ userRepo.Update             │ UPDATE (mutable fields)      │
    │ userRepo.FindByID           │ SELECT (all fields)          │
    │ HandleCreateUser            │ Request → User conversion    │
    │ HandleGetUser               │ User → Response conversion   │
    │ userService.Register        │ Creates new User             │
    │ authService.Authenticate    │ Reads Email, Password hash   │
    └─────────────────────────────┴──────────────────────────────┘

    Field-by-Field Impact:
    - ID: 12 usages (primary key, changing type = major)
    - Email: 8 usages (authentication, notifications)
    - Name: 5 usages (display only, low risk)
    - CreatedAt: 3 usages (audit, sorting)
    - UpdatedAt: 2 usages (audit only)

    Tests That Would Need Updates:
    - user_test.go (15 tests)
    - handlers_test.go (8 tests)
    - integration_test.go (3 tests)

    Recommendation:
    - Adding fields: Low risk, update Create/Update
    - Renaming fields: Medium risk, update 18 files
    - Changing types: High risk, full regression test needed
```

### Dependency Graph

```
User: Show the dependency graph for the auth package

AI: [Analyzes imports and call graph]

    Dependency Graph: internal/auth/

    Incoming Dependencies (who uses auth):
    ├── internal/handlers/auth.go
    ├── internal/handlers/users.go
    ├── internal/middleware/auth.go
    ├── internal/api/routes.go
    └── cmd/server/main.go

    Outgoing Dependencies (what auth uses):
    ├── internal/models     → User, Session types
    ├── internal/repository → UserRepository
    ├── internal/config     → Config loading
    ├── pkg/logger          → Logging
    └── External:
        ├── golang.org/x/crypto/bcrypt
        ├── github.com/golang-jwt/jwt/v5
        └── github.com/redis/go-redis/v9

    Dependency Diagram:
    ┌─────────────┐     ┌─────────────┐
    │  handlers   │────▶│    auth     │
    └─────────────┘     └──────┬──────┘
    ┌─────────────┐            │
    │ middleware  │────────────┤
    └─────────────┘            │
                               ▼
                    ┌─────────────────────┐
                    │ models, repository, │
                    │ config, logger      │
                    └─────────────────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │  bcrypt, jwt, redis │
                    └─────────────────────┘

    Circular Dependencies: None detected ✓
```

### Find Unused Exports

```
User: Which exported functions are never used outside their package?

AI: [Analyzes symbols and cross-package calls]

    Potentially Over-Exported Functions:

    internal/utils/
    ├── FormatBytes      - only used internally
    ├── TruncateString   - only used in tests
    └── ParseDuration    - never used

    internal/cache/
    ├── GetStats         - only used internally
    └── DebugDump        - never used (debug code?)

    pkg/logger/
    └── SetOutput        - never used (has default)

    Recommendations:
    1. Make internal-only functions unexported (lowercase)
    2. Remove truly unused functions
    3. Document why some exports exist (public API stability)

    Safe to Unexport:
    - utils.FormatBytes → formatBytes
    - cache.GetStats → getStats
```
