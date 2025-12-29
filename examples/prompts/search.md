# Search Prompts

Examples of how to search code semantically with mcp-codewizard.

## Basic Search

### Find by Functionality

```
User: Search for code that handles HTTP authentication

AI: [Uses search_code tool]
    Found 5 relevant results:

    1. internal/auth/middleware.go:45 (score: 0.92)
       AuthMiddleware - validates JWT tokens on incoming requests

    2. internal/handlers/login.go:23 (score: 0.87)
       HandleLogin - processes login form, creates session

    3. internal/auth/jwt.go:78 (score: 0.81)
       ValidateToken - parses and validates JWT claims
    ...
```

### Find by Concept

```
User: Find code related to database connection pooling

AI: [Uses search_code tool]
    Found 3 results:

    1. internal/db/pool.go:12 (score: 0.94)
       Pool struct - manages connection pool with configurable size

    2. internal/db/conn.go:89 (score: 0.78)
       getConnection - acquires connection from pool with timeout
    ...
```

### Find Error Handling

```
User: How do we handle errors in API responses?

AI: [Uses search_code tool]
    Your codebase uses a consistent error handling pattern:

    1. pkg/errors/api.go - defines APIError struct with code, message
    2. internal/handlers/errors.go - ErrorHandler middleware
    3. Example usage in internal/handlers/users.go:67:

       if err != nil {
           return c.JSON(500, APIError{Code: "DB_ERROR", Message: err.Error()})
       }
```

## Advanced Search

### With Language Filter

```
User: Find authentication code in Go files only

AI: [Uses search_code with languages filter]
    Searching Go files only...

    Found 4 results in .go files:
    1. auth/middleware.go - AuthMiddleware
    2. auth/session.go - SessionManager
    ...
```

### With File Path Filter

```
User: Search for validation logic in the handlers directory

AI: [Uses search_code with file path filter]
    Searching in **/handlers/**...

    Found validation code in:
    1. handlers/users.go:34 - validateUserInput()
    2. handlers/orders.go:56 - validateOrderRequest()
    ...
```

### Find by Pattern

```
User: Find all places where we log errors

AI: [Uses search_code tool]
    Found error logging patterns:

    1. Using slog.Error():
       - internal/api/server.go:123
       - internal/worker/processor.go:67

    2. Using log.Printf() (legacy):
       - cmd/migrate/main.go:45

    Recommendation: Consider migrating log.Printf to slog for consistency.
```

## Search with Context

### Get Surrounding Code

```
User: Find the ProcessOrder function and show me the context

AI: [Uses search_code then get_chunk with context]

    Found in internal/orders/processor.go:

    // Lines 45-89
    // ProcessOrder handles order processing workflow
    // It validates the order, checks inventory, and creates shipment
    func (p *Processor) ProcessOrder(ctx context.Context, order *Order) error {
        // Validate order
        if err := p.validator.Validate(order); err != nil {
            return fmt.Errorf("validation failed: %w", err)
        }

        // Check inventory
        available, err := p.inventory.Check(order.Items)
        if err != nil {
            return fmt.Errorf("inventory check failed: %w", err)
        }
        ...
    }
```

### Find Similar Code

```
User: Find code similar to how we handle payments

AI: [Uses search_code with "payment handling" query]

    Found related payment handling code:

    1. internal/payments/stripe.go - Stripe integration
    2. internal/payments/paypal.go - PayPal integration
    3. internal/orders/checkout.go - Uses both in checkout flow

    All follow the same pattern:
    - Create payment intent
    - Validate with provider
    - Store transaction record
```

## Comparison with Text Search

### Semantic Search Finds Concepts

```
User: Find code that prevents duplicate entries

AI: [Uses search_code - finds conceptually related code]

    Found duplicate prevention mechanisms:

    1. internal/db/users.go:34 - UNIQUE constraint check
    2. internal/cache/dedup.go:12 - Deduplication cache
    3. internal/queue/processor.go:78 - Idempotency key check

    Note: None of these contain the word "duplicate" but all
    prevent duplicate entries in different ways.
```

### Text Search Only Finds Exact Matches

```bash
# Traditional grep only finds literal matches
$ grep -r "duplicate" .
internal/utils/strings.go:// removeDuplicate removes duplicate strings
# Misses: UNIQUE constraints, dedup cache, idempotency checks
```

## Search Modes

### Hybrid (Default, Best Quality)

```
User: Search for user registration with hybrid mode

AI: [Uses search_code mode=hybrid]
    Combining semantic + keyword matching for best results...
```

### Vector Only (Faster)

```
User: Quick search for error handling patterns

AI: [Uses search_code mode=vector]
    Using vector search only (faster, may miss exact matches)...
```

### BM25 Only (Keyword Match)

```
User: Find exact usages of "ValidationError"

AI: [Uses search_code mode=bm25]
    Using keyword search for exact matches...

    Found 12 usages of "ValidationError":
    1. pkg/errors/validation.go:5 - type definition
    2. internal/handlers/users.go:34 - error return
    ...
```

## Tips for Effective Searching

1. **Be specific about what you want**
   - Good: "HTTP middleware that adds request ID to context"
   - Bad: "middleware"

2. **Use domain language**
   - If codebase uses "tenant", search for "tenant" not "customer"

3. **Describe the behavior**
   - "Function that retries failed HTTP requests with exponential backoff"

4. **Combine with follow-up questions**
   - First: "Find authentication code"
   - Then: "Show me who calls the AuthMiddleware"

5. **Ask for context when needed**
   - "Show me the validateOrder function with 10 lines of context"
