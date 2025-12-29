# Refactoring Workflow

How to use mcp-codewizard to safely refactor code.

## Scenario

You've been asked to refactor the payment processing module to support multiple payment providers. Currently it only supports Stripe, but you need to add PayPal and future providers.

## Step 1: Understand Current Implementation

### Map the Existing Code

```
User: Show me the current payment processing implementation

AI: [Uses search_code and get_symbols]

    Current Payment Implementation:

    Main File: internal/services/payment.go (245 lines)

    Structure:
    ```go
    type PaymentService struct {
        stripeClient *stripe.Client
        orderRepo    OrderRepository
        config       *PaymentConfig
    }

    func (s *PaymentService) ProcessPayment(ctx context.Context, order *Order) (*Payment, error)
    func (s *PaymentService) RefundPayment(ctx context.Context, paymentID string) error
    func (s *PaymentService) GetPaymentStatus(ctx context.Context, paymentID string) (PaymentStatus, error)
    ```

    Key Functions:
    ├── ProcessPayment (line 45) - Creates Stripe payment intent
    ├── RefundPayment (line 112) - Processes Stripe refund
    ├── GetPaymentStatus (line 156) - Checks Stripe payment status
    ├── handleWebhook (line 189) - Processes Stripe webhooks
    └── createStripeCustomer (line 220) - Creates Stripe customer

    Direct Stripe Dependencies:
    - stripe.Client used throughout
    - Stripe-specific types (PaymentIntent, Customer)
    - Stripe webhook signature verification
```

### Analyze Dependencies

```
User: What depends on the payment service? What would break if I change it?

AI: [Uses get_callers]

    Payment Service Callers:

    Direct Callers:
    ┌────────────────────────────────┬───────────────────────────┬───────────┐
    │ Function                       │ File                      │ Usage     │
    ├────────────────────────────────┼───────────────────────────┼───────────┤
    │ HandleCheckout                 │ handlers/checkout.go:45   │ ProcessPayment │
    │ HandleRefund                   │ handlers/refund.go:23     │ RefundPayment │
    │ HandlePaymentWebhook           │ handlers/webhook.go:67    │ handleWebhook │
    │ SubscriptionService.Renew      │ services/subscription.go  │ ProcessPayment │
    │ OrderService.RetryPayment      │ services/order.go:234     │ ProcessPayment │
    └────────────────────────────────┴───────────────────────────┴───────────┘

    Interface Consumers (none found - no interface exists!)
    ⚠️ All callers depend directly on *PaymentService concrete type

    Webhook Endpoints:
    - POST /webhooks/stripe → HandlePaymentWebhook

    Configuration:
    - config.yaml: stripe_api_key, stripe_webhook_secret
    - No abstraction for provider selection

    Test Files:
    - services/payment_test.go (12 tests, all mock Stripe)
    - handlers/checkout_test.go (5 tests)
```

## Step 2: Design the Refactoring

### Check for Existing Patterns

```
User: Do we have any provider patterns in this codebase I should follow?

AI: [Uses search_code for interfaces and providers]

    Existing Provider Patterns:

    1. ✓ Email Provider (internal/email/provider.go)
       ```go
       type EmailProvider interface {
           Send(ctx context.Context, msg *Message) error
           SendBatch(ctx context.Context, msgs []*Message) error
       }

       // Implementations
       type SendGridProvider struct { ... }
       type SESProvider struct { ... }
       type SMTPProvider struct { ... }
       ```
       Selection: Based on config `email.provider: sendgrid`

    2. ✓ Storage Provider (internal/storage/provider.go)
       ```go
       type StorageProvider interface {
           Upload(ctx context.Context, key string, data io.Reader) error
           Download(ctx context.Context, key string) (io.ReadCloser, error)
           Delete(ctx context.Context, key string) error
       }

       // Implementations
       type S3Provider struct { ... }
       type GCSProvider struct { ... }
       type LocalProvider struct { ... }
       ```

    3. ✓ Cache Provider (internal/cache/provider.go)
       Similar pattern with Redis, Memcached, Memory implementations

    Common Pattern Elements:
    - Interface in provider.go
    - Implementations in separate files (stripe.go, paypal.go)
    - Factory function: NewProvider(config) Provider
    - Config-based selection
    - Consistent error wrapping
```

### Design the Interface

```
User: Help me design a PaymentProvider interface based on these patterns

AI: [Analyzes current usage and patterns]

    Proposed PaymentProvider Interface:

    Based on current PaymentService usage and existing patterns:

    ```go
    // internal/payment/provider.go

    // PaymentProvider defines the interface for payment processors
    type PaymentProvider interface {
        // Name returns the provider identifier (e.g., "stripe", "paypal")
        Name() string

        // CreatePayment initiates a payment
        CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*Payment, error)

        // CapturePayment captures an authorized payment
        CapturePayment(ctx context.Context, paymentID string) (*Payment, error)

        // RefundPayment processes a refund
        RefundPayment(ctx context.Context, req *RefundRequest) (*Refund, error)

        // GetPayment retrieves payment status
        GetPayment(ctx context.Context, paymentID string) (*Payment, error)

        // ValidateWebhook validates and parses incoming webhook
        ValidateWebhook(ctx context.Context, req *WebhookRequest) (*WebhookEvent, error)
    }

    // Provider-agnostic types
    type CreatePaymentRequest struct {
        OrderID     string
        Amount      decimal.Decimal
        Currency    string
        CustomerID  string
        Description string
        Metadata    map[string]string
    }

    type Payment struct {
        ID            string
        ProviderID    string  // Provider's payment ID
        Provider      string  // "stripe", "paypal"
        Status        PaymentStatus
        Amount        decimal.Decimal
        Currency      string
        CreatedAt     time.Time
    }
    ```

    Changes from Current Implementation:
    1. Extract interface (currently no interface)
    2. Rename: ProcessPayment → CreatePayment (clearer)
    3. Add: Name() for provider identification
    4. Add: CapturePayment for two-step payments (PayPal needs this)
    5. Generalize: Webhook handling with provider-agnostic event type

    File Structure:
    ```
    internal/payment/
    ├── provider.go      # Interface + types
    ├── stripe.go        # Stripe implementation
    ├── paypal.go        # PayPal implementation (new)
    ├── factory.go       # NewProvider(config) factory
    └── service.go       # PaymentService (uses interface)
    ```
```

## Step 3: Plan the Migration

### Create Migration Steps

```
User: Create a step-by-step plan to safely refactor this

AI: [Creates detailed plan]

    Refactoring Plan: Payment Provider Abstraction

    Phase 1: Prepare (No Breaking Changes)
    ─────────────────────────────────────────
    Step 1.1: Create the interface
    - Create internal/payment/provider.go with interface
    - Create internal/payment/types.go with shared types
    - No changes to existing code yet

    Step 1.2: Wrap existing Stripe code
    - Create internal/payment/stripe.go
    - Implement PaymentProvider interface
    - Internally delegates to existing code

    Step 1.3: Add factory function
    - Create internal/payment/factory.go
    - NewProvider(config) returns *StripeProvider
    - Config reads "payment.provider" (default: "stripe")

    Step 1.4: Update PaymentService
    - Change: stripeClient → provider PaymentProvider
    - Methods delegate to provider
    - No external API changes

    Step 1.5: Verify
    - All existing tests pass
    - Manual testing of checkout flow
    - No behavior changes


    Phase 2: Add PayPal (Additive)
    ─────────────────────────────────────────
    Step 2.1: Implement PayPal provider
    - Create internal/payment/paypal.go
    - Implement PaymentProvider interface
    - Add PayPal SDK dependency

    Step 2.2: Update factory
    - NewProvider returns Stripe or PayPal based on config
    - Add config validation

    Step 2.3: Add PayPal webhook endpoint
    - POST /webhooks/paypal
    - Separate from Stripe webhook

    Step 2.4: Add provider selection to checkout
    - Frontend sends preferred provider
    - Backend validates and uses correct provider

    Step 2.5: Feature flag rollout
    - paypal.enabled: false (default)
    - Enable for beta users first


    Phase 3: Clean Up
    ─────────────────────────────────────────
    Step 3.1: Remove old payment.go
    - Delete internal/services/payment.go
    - All usage now through payment package

    Step 3.2: Update tests
    - Add provider interface mocks
    - Test each provider independently
    - Integration tests for both

    Step 3.3: Documentation
    - Update API docs
    - Add provider configuration docs


    Risk Mitigation:
    - Each phase is independently deployable
    - Feature flags for new providers
    - Existing Stripe flow unchanged until Phase 3
    - Rollback: Just change config back to "stripe"
```

### Identify Test Requirements

```
User: What tests do I need to add or update?

AI: [Analyzes existing tests]

    Test Plan for Payment Refactoring:

    Existing Tests to Update:
    ─────────────────────────────────────────
    1. services/payment_test.go
       Current: Tests *PaymentService directly
       Change: Test through PaymentProvider interface

       ```go
       // Before
       func TestProcessPayment(t *testing.T) {
           svc := NewPaymentService(stripeMock, ...)
           result, err := svc.ProcessPayment(ctx, order)
       }

       // After
       func TestCreatePayment_Stripe(t *testing.T) {
           provider := stripe.NewProvider(config)
           result, err := provider.CreatePayment(ctx, req)
       }
       ```

    2. handlers/checkout_test.go
       Current: Mocks PaymentService
       Change: Mock PaymentProvider interface

    New Tests to Add:
    ─────────────────────────────────────────
    1. payment/provider_test.go (interface tests)
       - TestProvider_Interface_Compliance
       - Run same tests against all providers

       ```go
       func TestAllProviders(t *testing.T) {
           providers := []PaymentProvider{
               stripe.NewProvider(stripeConfig),
               paypal.NewProvider(paypalConfig),
           }
           for _, p := range providers {
               t.Run(p.Name(), func(t *testing.T) {
                   runProviderTests(t, p)
               })
           }
       }
       ```

    2. payment/stripe_test.go
       - TestStripe_CreatePayment
       - TestStripe_RefundPayment
       - TestStripe_WebhookValidation
       - TestStripe_ErrorHandling

    3. payment/paypal_test.go
       - TestPayPal_CreatePayment
       - TestPayPal_CapturePayment (two-step)
       - TestPayPal_RefundPayment
       - TestPayPal_WebhookValidation

    4. payment/factory_test.go
       - TestNewProvider_Stripe
       - TestNewProvider_PayPal
       - TestNewProvider_Unknown_ReturnsError

    5. Integration tests
       - TestCheckoutFlow_Stripe (existing, update)
       - TestCheckoutFlow_PayPal (new)
       - TestProviderSwitch_MidTransaction

    Mock Requirements:
    ```go
    // payment/mock.go
    type MockProvider struct {
        mock.Mock
    }

    func (m *MockProvider) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*Payment, error) {
        args := m.Called(ctx, req)
        return args.Get(0).(*Payment), args.Error(1)
    }
    ```
```

## Step 4: Execute Safely

### Make Changes Incrementally

```
User: I've created the interface. Now help me verify the Stripe wrapper works correctly

AI: [Guides verification]

    Verification Checklist for Stripe Provider:

    1. Compile Check
       ```bash
       go build ./internal/payment/...
       ```
       ✓ Should compile with no errors

    2. Interface Compliance
       ```go
       // In stripe.go, add compile-time check:
       var _ PaymentProvider = (*StripeProvider)(nil)
       ```
       ✓ Compiler verifies all methods implemented

    3. Unit Tests
       ```bash
       go test ./internal/payment/stripe_test.go -v
       ```
       Expected: All tests pass

    4. Behavior Comparison
       Create a test that runs both old and new code:
       ```go
       func TestStripeProvider_MatchesOldBehavior(t *testing.T) {
           // Old way
           oldSvc := services.NewPaymentService(...)
           oldResult, _ := oldSvc.ProcessPayment(ctx, order)

           // New way
           provider := stripe.NewProvider(config)
           newResult, _ := provider.CreatePayment(ctx, toRequest(order))

           // Compare
           assert.Equal(t, oldResult.ID, newResult.ProviderID)
           assert.Equal(t, oldResult.Status, newResult.Status)
       }
       ```

    5. Integration Test
       ```bash
       # With Stripe test keys
       STRIPE_API_KEY=sk_test_xxx go test ./internal/payment/... -tags=integration
       ```

    6. Shadow Mode (Production Verification)
       ```go
       // Temporarily in handlers/checkout.go
       func (h *CheckoutHandler) HandleCheckout(c echo.Context) error {
           // Old path (still active)
           result, err := h.paymentService.ProcessPayment(ctx, order)

           // New path (shadow, log only)
           newResult, newErr := h.provider.CreatePayment(ctx, req)
           if newErr != nil || newResult.ID != result.ID {
               log.Warn("Payment provider mismatch",
                   "old", result, "new", newResult, "err", newErr)
           }

           return result, err  // Still use old result
       }
       ```
```

### Monitor for Issues

```
User: The refactoring is deployed. What should I monitor?

AI: [Provides monitoring guidance]

    Post-Deployment Monitoring:

    Key Metrics to Watch:
    ─────────────────────────────────────────

    1. Payment Success Rate
       Query: payment_success_total / payment_attempt_total
       Alert: < 98% (baseline)
       Dashboard: Grafana payment panel

    2. Payment Latency
       Query: histogram_quantile(0.95, payment_duration_seconds)
       Alert: p95 > 3s (baseline: 1.5s)
       Watch for: Provider initialization overhead

    3. Error Rates by Type
       Query: rate(payment_errors_total[5m]) by (error_type)
       New errors to watch:
       - "provider_not_found"
       - "interface_mismatch"
       - "webhook_validation_failed"

    4. Provider Distribution
       Query: payment_total by (provider)
       Expected: 100% stripe (until PayPal enabled)

    Log Queries:
    ─────────────────────────────────────────
    ```
    # Errors in payment package
    level:error payment

    # Provider selection issues
    "provider" AND ("not found" OR "unsupported")

    # Webhook issues
    webhook AND (failed OR invalid OR error)
    ```

    Alerts to Add:
    ─────────────────────────────────────────
    1. PaymentProviderError
       Condition: Any error from new provider code
       Severity: Warning (first 24h), Critical (after)

    2. PaymentLatencySpike
       Condition: p95 latency > 2x baseline
       Severity: Warning

    3. WebhookProcessingDelay
       Condition: Webhook queue depth > 100
       Severity: Warning

    Rollback Triggers:
    ─────────────────────────────────────────
    - Payment success rate drops below 95%
    - Error rate exceeds 5%
    - Multiple customer complaints
    - Provider API outage

    Rollback Procedure:
    1. Set config: payment.provider = "stripe" (legacy)
    2. Restart services
    3. Verify metrics return to baseline
    4. Investigate root cause
```

## Refactoring Quick Reference

| Step | Prompt |
|------|--------|
| Map code | "Show me the current [feature] implementation" |
| Find dependencies | "What depends on [component]? What would break?" |
| Find patterns | "Do we have similar patterns I should follow?" |
| Design interface | "Help me design a [Component] interface" |
| Plan migration | "Create a step-by-step plan to safely refactor" |
| Identify tests | "What tests do I need to add or update?" |
| Verify changes | "Help me verify [change] works correctly" |
| Monitor | "What should I monitor after deployment?" |

## Refactoring Principles

1. **Never break existing functionality** - Old code works until new code is proven

2. **Small, reversible changes** - Each commit should be deployable and rollback-able

3. **Tests before changes** - Write tests that verify current behavior first

4. **Interface segregation** - Extract minimal interfaces, don't over-abstract

5. **Feature flags** - New behavior behind flags until proven

6. **Shadow mode** - Run old and new in parallel, compare results

7. **Monitor everything** - You can't fix what you can't see
