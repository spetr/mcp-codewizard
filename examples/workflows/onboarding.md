# Onboarding Workflow

How to use mcp-codewizard to quickly understand a new codebase.

## Scenario

You've just joined a team and need to get up to speed on a large Go/Python/TypeScript codebase. Instead of spending days reading code, use your AI assistant with mcp-codewizard to accelerate your understanding.

## Step 1: Get the Big Picture

### Understand Project Structure

```
User: Give me an overview of this codebase - what are the main packages and their purposes?

AI: [Uses get_symbols to analyze package structure]

    Project Overview: e-commerce-platform

    Main Packages:
    ├── cmd/                    # Application entry points
    │   ├── api/               # REST API server (main entry)
    │   ├── worker/            # Background job processor
    │   └── migrate/           # Database migration tool
    │
    ├── internal/              # Private application code
    │   ├── handlers/          # HTTP request handlers (23 files)
    │   ├── services/          # Business logic layer (15 files)
    │   ├── repository/        # Database access (12 files)
    │   ├── models/            # Domain models (8 files)
    │   └── middleware/        # HTTP middleware (5 files)
    │
    ├── pkg/                   # Public, reusable packages
    │   ├── logger/            # Structured logging
    │   ├── errors/            # Custom error types
    │   └── validation/        # Input validation
    │
    └── proto/                 # gRPC definitions

    Key Stats:
    - 156 Go files
    - 45,000 lines of code
    - 312 exported functions
    - 45 interfaces
```

### Find Entry Points

```
User: What are the main entry points? How does the application start?

AI: [Uses get_symbols and search_code]

    Application Entry Points:

    1. API Server (cmd/api/main.go)
       main() → setupServer() → registerRoutes() → server.Start()

       HTTP Endpoints: 34 routes
       Key handlers:
       - /api/v1/products   → ProductHandler
       - /api/v1/orders     → OrderHandler
       - /api/v1/users      → UserHandler
       - /api/v1/auth       → AuthHandler

    2. Background Worker (cmd/worker/main.go)
       main() → worker.Start()

       Jobs:
       - OrderProcessor     (processes new orders)
       - EmailSender        (sends notifications)
       - InventorySync      (syncs with warehouse)

    3. Startup Flow:
       main.go:15  → Load config from env/files
       main.go:23  → Connect to database
       main.go:31  → Initialize Redis cache
       main.go:38  → Setup HTTP server with middleware
       main.go:45  → Register routes
       main.go:52  → Start server (blocking)
```

## Step 2: Understand Core Domain

### Find Domain Models

```
User: What are the main domain models and how do they relate?

AI: [Uses get_symbols kind=type]

    Core Domain Models:

    1. User (internal/models/user.go)
       Fields: ID, Email, Name, PasswordHash, Role, CreatedAt
       Related: Orders (1:N), Reviews (1:N)

    2. Product (internal/models/product.go)
       Fields: ID, Name, Description, Price, Stock, CategoryID
       Related: Category (N:1), OrderItems (1:N), Reviews (1:N)

    3. Order (internal/models/order.go)
       Fields: ID, UserID, Status, Total, CreatedAt
       Related: User (N:1), OrderItems (1:N), Payment (1:1)

    4. OrderItem (internal/models/order_item.go)
       Fields: ID, OrderID, ProductID, Quantity, Price
       Related: Order (N:1), Product (N:1)

    Entity Relationship:
    ┌────────┐      ┌─────────┐      ┌─────────┐
    │  User  │──1:N─│  Order  │──1:N─│OrderItem│
    └────────┘      └─────────┘      └────┬────┘
         │                                 │
         │              ┌─────────┐       N:1
         └──────1:N─────│ Review  │────────│
                        └────┬────┘        │
                            N:1            │
                             │        ┌────┴────┐
                             └────────│ Product │
                                      └─────────┘
```

### Understand Business Logic

```
User: How does order processing work? Walk me through the flow.

AI: [Uses search_code and get_callees]

    Order Processing Flow:

    1. Create Order (internal/services/order.go:CreateOrder)
       │
       ├── Validate cart items
       │   └── Check stock availability
       │
       ├── Calculate totals
       │   ├── Apply discounts
       │   └── Calculate tax
       │
       ├── Create order record (status: "pending")
       │
       └── Emit OrderCreated event

    2. Process Payment (internal/services/payment.go:ProcessPayment)
       │
       ├── Create payment intent (Stripe)
       │
       ├── On success:
       │   ├── Update order status → "paid"
       │   └── Emit PaymentSucceeded event
       │
       └── On failure:
           ├── Update order status → "payment_failed"
           └── Release reserved stock

    3. Fulfill Order (internal/services/fulfillment.go:FulfillOrder)
       │
       ├── Reduce inventory
       ├── Create shipment record
       ├── Send confirmation email
       └── Update order status → "shipped"

    Key Files:
    - internal/services/order.go (main orchestration)
    - internal/services/payment.go (payment processing)
    - internal/services/inventory.go (stock management)
    - internal/events/handlers.go (event-driven parts)
```

## Step 3: Find Specific Code

### How Do We Handle X?

```
User: How does authentication work in this codebase?

AI: [Uses search_code "authentication"]

    Authentication System:

    1. Login Flow (internal/handlers/auth.go:HandleLogin)
       - Accepts email/password
       - Validates credentials against DB
       - Generates JWT token (24h expiry)
       - Sets HTTP-only cookie

    2. JWT Validation (internal/middleware/auth.go:AuthMiddleware)
       - Extracts token from cookie or Authorization header
       - Validates signature and expiry
       - Loads user from cache/DB
       - Sets user in request context

    3. Authorization (internal/middleware/rbac.go)
       - Role-based: admin, manager, customer
       - Checked via RequireRole("admin") middleware

    Key Code:
    ```go
    // internal/middleware/auth.go:34
    func AuthMiddleware(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractToken(r)
            claims, err := validateToken(token)
            if err != nil {
                http.Error(w, "Unauthorized", 401)
                return
            }
            ctx := context.WithValue(r.Context(), userKey, claims.UserID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
    ```

    Configuration:
    - JWT_SECRET env var (required)
    - TOKEN_EXPIRY (default: 24h)
    - REFRESH_TOKEN_EXPIRY (default: 7d)
```

### Where Is This Pattern Used?

```
User: Show me how we handle database transactions in this project

AI: [Uses search_code "transaction"]

    Database Transaction Patterns:

    Pattern 1: Service-Level Transactions
    Location: internal/services/order.go:89

    ```go
    func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
        tx, err := s.db.BeginTx(ctx, nil)
        if err != nil {
            return nil, err
        }
        defer tx.Rollback() // rollback if not committed

        order, err := s.orderRepo.CreateTx(ctx, tx, req)
        if err != nil {
            return nil, err
        }

        for _, item := range req.Items {
            if err := s.inventoryRepo.ReserveTx(ctx, tx, item); err != nil {
                return nil, err
            }
        }

        if err := tx.Commit(); err != nil {
            return nil, err
        }
        return order, nil
    }
    ```

    Pattern 2: Repository with Transaction Support
    All repositories accept optional *sql.Tx parameter:
    - orderRepo.CreateTx(ctx, tx, ...)
    - inventoryRepo.UpdateTx(ctx, tx, ...)
    - paymentRepo.CreateTx(ctx, tx, ...)

    Usage Locations:
    - internal/services/order.go (order creation)
    - internal/services/refund.go (refund processing)
    - internal/jobs/inventory_sync.go (bulk updates)
```

## Step 4: Understand Conventions

### Code Style and Patterns

```
User: What patterns and conventions does this codebase follow?

AI: [Analyzes code structure]

    Code Conventions:

    1. Layer Architecture
       handlers → services → repositories → database
       - Handlers: HTTP concerns only
       - Services: Business logic
       - Repositories: Data access

    2. Error Handling
       - Custom error types in pkg/errors/
       - Wrap errors with context: fmt.Errorf("create order: %w", err)
       - Return domain errors: ErrNotFound, ErrValidation

    3. Dependency Injection
       - Constructor injection via New*() functions
       - No global state
       - Interfaces for testability

    4. Naming Conventions
       - Handlers: Handle{Action} (HandleCreateUser)
       - Services: {Action} (CreateUser, GetByID)
       - Repos: {Action} (Create, FindByID, Update)

    5. File Organization
       - One domain per file (user.go, order.go)
       - Tests alongside code (*_test.go)
       - Interfaces in separate file (interfaces.go)

    6. Configuration
       - Environment variables with defaults
       - Config struct in internal/config/
       - Validation at startup

    Example Flow:
    POST /users → HandleCreateUser → userService.Create → userRepo.Create
```

## Step 5: Find Your First Task

### Where to Start Contributing

```
User: I need to add email verification to user registration. Where should I start?

AI: [Uses search_code and get_callers]

    Email Verification Implementation Guide:

    Current Registration Flow:
    1. POST /api/v1/users → HandleCreateUser (handlers/user.go:45)
    2. → userService.Create (services/user.go:23)
    3. → userRepo.Create (repository/user.go:34)
    4. → Send welcome email (async, events/user.go:12)

    Suggested Changes:

    1. Add to User model (models/user.go):
       - EmailVerified bool
       - VerificationToken string
       - VerificationExpiry time.Time

    2. Modify userService.Create (services/user.go:23):
       - Generate verification token
       - Set EmailVerified = false
       - Include token in welcome email

    3. Add new endpoint (handlers/user.go):
       - GET /api/v1/users/verify?token=xxx
       - → HandleVerifyEmail

    4. Add new service method (services/user.go):
       - VerifyEmail(token string) error

    5. Update database migration:
       - Add columns to users table

    Related Files to Study:
    - internal/services/user.go (current implementation)
    - internal/handlers/user.go (HTTP layer)
    - internal/events/handlers.go (email sending)
    - migrations/000X_add_user_fields.sql (migration example)

    Tests to Add:
    - services/user_test.go (unit tests)
    - handlers/user_test.go (integration tests)
```

## Quick Reference Card

| Question | Prompt |
|----------|--------|
| Project structure | "Give me an overview of this codebase" |
| Entry points | "What are the main entry points?" |
| Domain models | "What are the main domain models?" |
| How X works | "How does [feature] work? Walk me through the flow" |
| Find code | "How do we handle [pattern]?" |
| Conventions | "What patterns does this codebase follow?" |
| Start task | "I need to implement X. Where should I start?" |

## Tips for Effective Onboarding

1. **Start with entry points** - Understand how the app starts and routes requests

2. **Follow the data** - Trace how data flows from API to database

3. **Find the domain model** - Understanding the core types unlocks everything

4. **Look for patterns** - Once you see one handler/service/repo, you've seen them all

5. **Read tests** - Tests often show intended usage better than docs

6. **Ask "who calls this?"** - Understanding callers gives context for any function

7. **Take notes** - Ask the AI to summarize and save for reference
