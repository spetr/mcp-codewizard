# Analysis Prompts

Examples of code analysis using mcp-codewizard's analysis tools.

## Complexity Analysis

### File Complexity

```
User: What's the complexity of internal/processor/engine.go?

AI: [Uses get_complexity tool]

    Complexity Analysis: internal/processor/engine.go

    File Metrics:
    ├── Lines of Code: 456
    ├── Functions: 12
    ├── Average Complexity: 8.3
    └── Max Complexity: 24 (ProcessBatch)

    Function Breakdown:
    ┌────────────────────┬────────────┬───────────┬─────────┐
    │ Function           │ Cyclomatic │ Cognitive │ Nesting │
    ├────────────────────┼────────────┼───────────┼─────────┤
    │ ProcessBatch       │ 24         │ 31        │ 5       │ ⚠️ High
    │ handleError        │ 12         │ 15        │ 3       │ ⚠️ Medium
    │ validateInput      │ 8          │ 10        │ 2       │
    │ NewEngine          │ 2          │ 2         │ 1       │
    │ Start              │ 3          │ 4         │ 1       │
    └────────────────────┴────────────┴───────────┴─────────┘

    Recommendations:
    - ProcessBatch: Consider breaking into smaller functions
    - handleError: Switch statement could use early returns
```

### Find Most Complex Code

```
User: What are the most complex functions in this project?

AI: [Uses get_complexity for multiple files]

    Top 10 Most Complex Functions:

    1. internal/processor/engine.go:ProcessBatch
       Cyclomatic: 24, Cognitive: 31
       Issue: Deep nesting, multiple exit points

    2. internal/parser/sql.go:ParseQuery
       Cyclomatic: 19, Cognitive: 28
       Issue: Complex switch with nested conditions

    3. internal/api/handlers.go:HandleWebhook
       Cyclomatic: 16, Cognitive: 22
       Issue: Multiple error handling branches

    4. cmd/server/main.go:setupRoutes
       Cyclomatic: 14, Cognitive: 18
       Issue: Long function, could be split

    ...

    Summary:
    - 4 functions exceed complexity threshold (15)
    - Average project complexity: 5.2
    - Recommended: Refactor top 3 functions
```

### Function-Specific Analysis

```
User: Analyze the complexity of the HandlePayment function

AI: [Uses get_complexity tool]

    Function: HandlePayment
    Location: internal/handlers/payment.go:45-123

    Metrics:
    ├── Cyclomatic Complexity: 14
    ├── Cognitive Complexity: 18
    ├── Max Nesting Depth: 4
    ├── Lines of Code: 78
    └── Parameters: 3

    Complexity Breakdown:
    - 6 if statements
    - 2 switch cases
    - 3 error handling blocks
    - 1 loop with break

    Hotspots (high local complexity):
    - Lines 67-89: Nested error handling
    - Lines 95-110: Multiple conditions in loop

    Suggested Refactoring:
    1. Extract validation logic (lines 50-65) to validatePayment()
    2. Use early returns instead of nested ifs
    3. Extract retry logic (lines 95-110) to retryWithBackoff()
```

## Dead Code Detection

### Find Unused Functions

```
User: Are there any unused functions in the codebase?

AI: [Uses get_dead_code tool]

    Dead Code Analysis Results:

    Unused Functions (no callers found):
    ┌─────────────────────────────────────┬──────────────────┐
    │ Function                            │ Location         │
    ├─────────────────────────────────────┼──────────────────┤
    │ formatLegacyResponse                │ api/legacy.go:34 │
    │ oldHashPassword                     │ auth/hash.go:89  │
    │ debugPrintRequest                   │ utils/debug.go:12│
    │ TestHelper_unused                   │ test/helpers.go  │
    └─────────────────────────────────────┴──────────────────┘

    Potentially Unused (only test callers):
    - internal/cache/stats.go:GetDetailedStats
      Only called from: cache/stats_test.go

    Notes:
    - formatLegacyResponse: Might be kept for backwards compatibility
    - oldHashPassword: Consider removing after migration
    - debugPrintRequest: Debug code, safe to remove
```

### Find Dead Code in Specific Package

```
User: Check for dead code in the internal/utils package

AI: [Uses get_dead_code tool]

    Analyzing internal/utils/...

    Unused Exports:
    1. FormatBytes - no external callers
    2. TruncateString - only used in tests
    3. ParseDuration - deprecated, replaced by time.ParseDuration

    Unused Internal Functions:
    1. helper_sanitize (utils/strings.go:145)
    2. old_validate (utils/validation.go:78)

    Recommendation:
    - Remove 234 lines of dead code
    - Consider deprecation warnings before removal
```

### Analyze Test Coverage Gaps

```
User: What code is never tested?

AI: [Uses get_dead_code and get_callers tools]

    Functions with no test coverage:

    Critical (business logic):
    1. internal/billing/calculate.go:CalculateDiscount
    2. internal/orders/refund.go:ProcessRefund
    3. internal/auth/mfa.go:ValidateTOTP

    Non-critical (utilities):
    1. internal/utils/format.go:FormatCurrency
    2. pkg/logger/rotate.go:RotateLogs

    Test file exists but function untested:
    1. internal/cache/evict.go:EvictExpired
       Test file: cache/evict_test.go (tests other functions)

    Recommendation: Prioritize tests for billing and auth functions.
```

## Git Blame Analysis

### Who Wrote This Code?

```
User: Who wrote the authentication middleware?

AI: [Uses get_blame tool]

    Git Blame: internal/auth/middleware.go

    Overall Authorship:
    ├── alice@company.com: 65% (primary author)
    ├── bob@company.com: 25% (security fixes)
    └── carol@company.com: 10% (refactoring)

    Key Functions:
    ┌────────────────────┬─────────────────────┬────────────┐
    │ Function           │ Primary Author      │ Last Modified │
    ├────────────────────┼─────────────────────┼────────────┤
    │ AuthMiddleware     │ alice@company.com   │ 2024-01-15 │
    │ ValidateToken      │ bob@company.com     │ 2024-02-20 │
    │ RefreshToken       │ alice@company.com   │ 2024-01-15 │
    └────────────────────┴─────────────────────┴────────────┘

    Recent Changes:
    - 2024-02-20: bob fixed JWT validation vulnerability
    - 2024-01-15: alice initial implementation
```

### Blame for Specific Lines

```
User: Who added the rate limiting logic in line 45-60 of api/server.go?

AI: [Uses get_blame tool with line range]

    Blame for api/server.go:45-60

    45 │ david  │ 2024-03-01 │ // Rate limiting middleware
    46 │ david  │ 2024-03-01 │ func RateLimitMiddleware(limit int) func(next http.Handler) http.Handler {
    47 │ david  │ 2024-03-01 │     limiter := rate.NewLimiter(rate.Limit(limit), limit*2)
    48 │ david  │ 2024-03-01 │     return func(next http.Handler) http.Handler {
    49 │ emily  │ 2024-03-15 │         return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    50 │ emily  │ 2024-03-15 │             if !limiter.Allow() {
    51 │ emily  │ 2024-03-15 │                 http.Error(w, "Too Many Requests", 429)
    52 │ emily  │ 2024-03-15 │                 return
    53 │ emily  │ 2024-03-15 │             }
    ...

    Summary:
    - david@company.com added initial rate limiting (2024-03-01)
    - emily@company.com fixed response handling (2024-03-15)

    Related commits:
    - abc123: "Add rate limiting to API endpoints"
    - def456: "Fix rate limit response code"
```

### Find Code Age

```
User: What's the oldest code in the project?

AI: [Uses get_blame across files]

    Oldest Code Analysis:

    Oldest Files (by first commit):
    1. cmd/server/main.go - 2022-06-15 (project start)
    2. internal/config/config.go - 2022-06-15
    3. pkg/logger/logger.go - 2022-06-20

    Oldest Unchanged Code:
    1. internal/constants/errors.go:1-50
       Last modified: 2022-07-01 (2+ years ago)

    2. pkg/utils/hash.go:computeHash
       Last modified: 2022-08-15

    Stale Code (no changes in 1+ year):
    - 15 files have not been modified in over a year
    - Consider reviewing for deprecation or updates
```

## Combined Analysis

### Refactoring Candidates

```
User: Find code that needs refactoring

AI: [Combines complexity + dead code + blame analysis]

    Refactoring Candidates (Priority Order):

    1. internal/processor/engine.go:ProcessBatch
       Issues:
       ├── Complexity: 24 (threshold: 15)
       ├── Cognitive: 31 (hard to understand)
       ├── Last touched: 8 months ago
       └── Author: alice (still on team)

       Recommendation: Break into smaller functions

    2. internal/api/legacy.go
       Issues:
       ├── 3 unused functions (dead code)
       ├── No tests
       ├── Last touched: 14 months ago
       └── Author: bob (left company)

       Recommendation: Remove or archive

    3. internal/parser/sql.go
       Issues:
       ├── Complexity: 19
       ├── Deep nesting (5 levels)
       ├── Actively modified
       └── Multiple authors

       Recommendation: Refactor with careful testing

    Quick Wins (easy improvements):
    - Remove 4 unused utility functions (-156 lines)
    - Extract 3 helper functions from ProcessBatch
    - Add early returns to handleError
```

### Security Review

```
User: Find security-sensitive code and its authors

AI: [Combines search + blame analysis]

    Security-Sensitive Code:

    1. Authentication
       Location: internal/auth/
       Primary Author: security-team@company.com
       Last Audit: 2024-01-15
       Status: Recently reviewed

    2. Password Handling
       Location: internal/users/password.go
       Primary Author: alice@company.com
       Last Change: 2023-06-10
       Status: ⚠️ Not reviewed in 18 months

    3. SQL Queries
       Locations with raw SQL:
       - internal/db/queries.go (parameterized - OK)
       - internal/reports/custom.go:BuildQuery (⚠️ string concat)

       BuildQuery author: intern@company.com (2023-07-01)
       Recommendation: Review for SQL injection

    4. Crypto Usage
       Location: internal/crypto/
       Uses: AES-256-GCM (OK), MD5 (⚠️ deprecated)
       MD5 usage: internal/legacy/hash.go - for non-security purpose
```
