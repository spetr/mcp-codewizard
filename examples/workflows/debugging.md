# Debugging Workflow

How to use mcp-codewizard to find and fix bugs efficiently.

## Scenario

You've received a bug report: "Users sometimes see incorrect order totals." Let's use the AI assistant to track down and fix this issue.

## Step 1: Understand the Problem Area

### Find Relevant Code

```
User: I need to debug order total calculation. Where is the code that calculates order totals?

AI: [Uses search_code "order total calculate"]

    Order Total Calculation Code:

    Primary Location:
    internal/services/order.go:CalculateTotal (lines 89-134)

    ```go
    func (s *OrderService) CalculateTotal(order *Order) (decimal.Decimal, error) {
        var total decimal.Decimal

        for _, item := range order.Items {
            itemTotal := item.Price.Mul(decimal.NewFromInt(int64(item.Quantity)))
            total = total.Add(itemTotal)
        }

        // Apply discount
        if order.DiscountCode != "" {
            discount, err := s.discountService.GetDiscount(order.DiscountCode)
            if err == nil {
                total = applyDiscount(total, discount)
            }
        }

        // Add tax
        tax := s.taxService.CalculateTax(total, order.ShippingAddress)
        total = total.Add(tax)

        return total, nil
    }
    ```

    Called From:
    - CreateOrder (services/order.go:45)
    - UpdateOrder (services/order.go:156)
    - RecalculateOrder (services/order.go:201)

    Related Functions:
    - applyDiscount (services/discount.go:34)
    - CalculateTax (services/tax.go:23)
```

### Trace the Call Path

```
User: Show me the complete flow from HTTP request to total calculation

AI: [Uses get_callees and get_callers]

    Request Flow: POST /api/v1/orders

    1. HandleCreateOrder (handlers/order.go:34)
       │
       ├── Parse request body
       │
       └── orderService.CreateOrder(ctx, req)
           │
           ├── Validate items exist
           │   └── productRepo.FindByIDs()
           │
           ├── Get current prices  ← ⚠️ Potential issue
           │   └── productRepo.GetPrices()
           │
           ├── Build order items
           │   └── Uses prices from GetPrices()
           │
           ├── CalculateTotal(order)  ← Main calculation
           │   ├── Sum item prices
           │   ├── Apply discount
           │   └── Add tax
           │
           └── orderRepo.Create(order)

    Data Flow:
    Request → ProductPrices → OrderItems → CalculateTotal → StoredOrder
                   ↑
           Prices fetched here, but stored in OrderItem
           If price changes between steps, could cause issues
```

## Step 2: Identify Potential Issues

### Check for Race Conditions

```
User: Could there be a race condition in the order total calculation?

AI: [Analyzes code flow and concurrency]

    Race Condition Analysis:

    ⚠️ Potential Issue Found:

    In CreateOrder (services/order.go:45-88):

    ```go
    // Line 52: Fetch current prices
    prices, err := s.productRepo.GetPrices(productIDs)

    // ... other code ...

    // Line 67: Build order items with those prices
    for _, item := range req.Items {
        orderItem := OrderItem{
            ProductID: item.ProductID,
            Quantity:  item.Quantity,
            Price:     prices[item.ProductID],  // Uses fetched price
        }
        order.Items = append(order.Items, orderItem)
    }

    // Line 78: Calculate total
    total, err := s.CalculateTotal(order)
    ```

    Problem:
    - Price is fetched at line 52
    - If product price changes between line 52 and 78
    - The OrderItem.Price is correct, but...

    Wait, looking more carefully...

    ✓ OrderItem stores the price at time of order (good)
    ✓ CalculateTotal uses OrderItem.Price (good)

    Let me check the discount flow...

    ⚠️ FOUND IT: Discount Race Condition

    In CalculateTotal (services/order.go:105):
    ```go
    if order.DiscountCode != "" {
        discount, err := s.discountService.GetDiscount(order.DiscountCode)
        if err == nil {
            total = applyDiscount(total, discount)
        }
    }
    ```

    Problem:
    - Discount is fetched EVERY time CalculateTotal is called
    - If discount percentage changes between order creation and viewing
    - The displayed total will differ!

    Recommendation:
    - Store discount details in Order at creation time
    - Don't re-fetch discount on each calculation
```

### Find Similar Patterns

```
User: Are there other places where we re-fetch data that should be cached?

AI: [Uses search_code to find similar patterns]

    Similar Re-fetch Patterns Found:

    1. ⚠️ Tax Calculation (services/tax.go:23)
       ```go
       func (s *TaxService) CalculateTax(amount decimal.Decimal, addr Address) decimal.Decimal {
           rate := s.getTaxRate(addr)  // Fetches from external API
           return amount.Mul(rate)
       }
       ```
       Issue: Tax rate fetched on every call, could change

    2. ⚠️ Shipping Cost (services/shipping.go:45)
       ```go
       func (s *ShippingService) GetShippingCost(order *Order) decimal.Decimal {
           return s.carrier.GetRate(order.Weight, order.Destination)
       }
       ```
       Issue: Carrier rates fetched live

    3. ✓ Product Prices (services/order.go:52)
       Stored in OrderItem.Price - correct pattern

    Recommendation:
    Store all pricing components at order creation:
    - Subtotal
    - DiscountAmount (not percentage)
    - TaxAmount
    - ShippingAmount
    - Total
```

## Step 3: Verify the Bug

### Check Data Model

```
User: What's currently stored in the Order model? Does it store the calculated amounts?

AI: [Uses get_symbols to find Order struct]

    Order Model (internal/models/order.go:12-45):

    ```go
    type Order struct {
        ID              int64           `json:"id"`
        UserID          int64           `json:"user_id"`
        Status          OrderStatus     `json:"status"`
        DiscountCode    string          `json:"discount_code,omitempty"`
        ShippingAddress Address         `json:"shipping_address"`
        Items           []OrderItem     `json:"items"`
        Total           decimal.Decimal `json:"total"`  // ← Only stores final total
        CreatedAt       time.Time       `json:"created_at"`
        UpdatedAt       time.Time       `json:"updated_at"`
    }
    ```

    ⚠️ Missing Fields:
    - Subtotal (before discounts)
    - DiscountAmount (actual discount applied)
    - TaxAmount
    - ShippingAmount

    Current Total is recalculated dynamically, causing inconsistency.

    OrderItem Model (internal/models/order_item.go):
    ```go
    type OrderItem struct {
        ID        int64           `json:"id"`
        OrderID   int64           `json:"order_id"`
        ProductID int64           `json:"product_id"`
        Quantity  int             `json:"quantity"`
        Price     decimal.Decimal `json:"price"`  // ✓ Stores price at purchase
    }
    ```
    This part is correct - price is stored.
```

### Find Where Bug Manifests

```
User: Where does the order total get displayed to users? I want to see if it recalculates

AI: [Uses get_callers for CalculateTotal]

    Places Where Order Total is Used:

    1. Order Creation (correct total stored)
       services/order.go:CreateOrder
       → CalculateTotal called once
       → Result stored in Order.Total

    2. ⚠️ Order Details API (recalculates!)
       handlers/order.go:HandleGetOrder (line 89)
       ```go
       func (h *OrderHandler) HandleGetOrder(c echo.Context) error {
           order, err := h.orderService.GetOrder(ctx, orderID)
           if err != nil {
               return err
           }

           // BUG: Recalculates total with current discount rates!
           total, _ := h.orderService.CalculateTotal(order)
           order.Total = total

           return c.JSON(200, order)
       }
       ```

    3. ⚠️ Order List API (also recalculates)
       handlers/order.go:HandleListOrders (line 134)
       Same pattern - recalculates for each order

    4. ✓ Invoice Generation (uses stored total)
       services/invoice.go:GenerateInvoice
       Uses order.Total directly - correct

    5. ⚠️ Admin Dashboard (recalculates)
       handlers/admin/orders.go:HandleAdminOrders

    Root Cause Confirmed:
    - Total is correctly calculated and stored at creation
    - But viewing endpoints recalculate instead of using stored value
    - If discount/tax rates changed, displayed total differs
```

## Step 4: Design the Fix

### Understand Impact

```
User: If I fix this, what else might be affected?

AI: [Uses get_callers and impact analysis]

    Impact Analysis for Fixing Order Total Bug:

    Changes Needed:

    1. Order Model (models/order.go)
       Add fields: Subtotal, DiscountAmount, TaxAmount, ShippingAmount
       Impact: Database migration required

    2. CreateOrder Service (services/order.go)
       Store all amount components
       Impact: Low - isolated change

    3. CalculateTotal (services/order.go)
       Change to only calculate once, or use stored values
       Impact: Medium - called from 5 places

    4. HandleGetOrder (handlers/order.go)
       Remove recalculation, use stored Total
       Impact: Low - simple removal

    5. Database Migration
       Add columns, backfill existing orders
       Impact: Medium - need backfill strategy

    Callers of CalculateTotal (need review):
    ├── CreateOrder       - Keep calculation
    ├── UpdateOrder       - Need to recalculate
    ├── RecalculateOrder  - Intentional recalc (admin use)
    ├── HandleGetOrder    - REMOVE (use stored)
    └── HandleListOrders  - REMOVE (use stored)

    Tests to Update:
    - services/order_test.go (12 tests)
    - handlers/order_test.go (8 tests)
    - Add migration test

    Backwards Compatibility:
    - API response unchanged (same Total field)
    - New fields are additive
    - Backfill needed for accurate historical data
```

### Find Similar Fixes in Codebase

```
User: Have we solved similar problems before? Are there examples of storing calculated values?

AI: [Uses search_code for patterns]

    Similar Patterns in Codebase:

    1. ✓ Invoice Amounts (services/invoice.go)
       ```go
       type Invoice struct {
           Subtotal      decimal.Decimal
           DiscountTotal decimal.Decimal
           TaxTotal      decimal.Decimal
           GrandTotal    decimal.Decimal
           // All stored at creation time
       }
       ```
       This is the correct pattern to follow!

    2. ✓ Subscription Pricing (services/subscription.go)
       ```go
       type Subscription struct {
           PlanPriceAtSignup decimal.Decimal  // Locked in price
           CurrentPlanPrice  decimal.Decimal  // For display only
       }
       ```
       Stores price at signup, shows current separately

    3. ✓ Shipping Quotes (services/shipping.go)
       ```go
       type ShippingQuote struct {
           QuotedAt time.Time
           Amount   decimal.Decimal
           ValidFor time.Duration
       }
       ```
       Quotes are cached with expiry

    Recommended Pattern (from Invoice):
    ```go
    type Order struct {
        // ... existing fields ...

        // Stored at creation time, never recalculated
        Subtotal       decimal.Decimal `json:"subtotal"`
        DiscountAmount decimal.Decimal `json:"discount_amount"`
        TaxAmount      decimal.Decimal `json:"tax_amount"`
        ShippingAmount decimal.Decimal `json:"shipping_amount"`
        Total          decimal.Decimal `json:"total"`

        // For audit/display purposes
        DiscountCode   string  `json:"discount_code,omitempty"`
        DiscountRate   float64 `json:"discount_rate,omitempty"`
        TaxRate        float64 `json:"tax_rate,omitempty"`
    }
    ```
```

## Step 5: Implement and Verify

### Verify Fix Doesn't Break Tests

```
User: What tests exist for order calculation that I need to update?

AI: [Uses search_code in test files]

    Existing Tests for Order Calculation:

    services/order_test.go:
    ├── TestCalculateTotal_Basic (line 45)
    │   Tests: Simple order with 3 items
    │   Update: Should verify stored values match
    │
    ├── TestCalculateTotal_WithDiscount (line 78)
    │   Tests: 20% discount code
    │   Update: Verify DiscountAmount stored
    │
    ├── TestCalculateTotal_WithTax (line 112)
    │   Tests: California tax rate
    │   Update: Verify TaxAmount stored
    │
    └── TestCalculateTotal_EdgeCases (line 145)
        Tests: Empty order, zero prices
        Update: Verify all amounts are zero

    handlers/order_test.go:
    ├── TestHandleGetOrder (line 34)
    │   Tests: Returns order with correct total
    │   Update: Remove assertion on recalculated total
    │
    └── TestHandleCreateOrder (line 89)
        Tests: Full order creation flow
        Update: Assert new fields are populated

    Missing Tests (should add):
    - TestOrderTotal_DiscountChangedAfterCreation
    - TestOrderTotal_TaxRateChangedAfterCreation
    - TestOrderTotal_BackfillMigration
```

### Check for Edge Cases

```
User: What edge cases should I consider for this fix?

AI: [Analyzes code paths]

    Edge Cases to Handle:

    1. Partial Refunds
       Location: services/refund.go:ProcessRefund
       Issue: Refunds recalculate amounts
       Solution: Store refund amount separately, don't touch original

    2. Order Updates (add/remove items)
       Location: services/order.go:UpdateOrder
       Issue: Should recalculate when items change
       Solution: Recalculate ONLY on update, not on read

    3. Discount Code Removal
       Location: services/order.go:RemoveDiscount
       Issue: Should recalculate without discount
       Solution: Correct - this is intentional recalculation

    4. Currency Conversion
       Location: services/currency.go
       Issue: Displayed in different currency than stored?
       Check: No conversion found - amounts stored in single currency ✓

    5. Rounding Errors
       Issue: decimal.Decimal should prevent this
       Check: ✓ Using decimal.Decimal throughout

    6. Historical Orders
       Issue: Old orders don't have new fields
       Solution: Migration must backfill or set defaults

    7. Failed Payments
       Issue: Order exists but payment failed
       Check: Status = "pending_payment", total still correct ✓

    8. Subscription Orders
       Location: services/subscription.go
       Issue: Recurring orders created from template
       Solution: Each order instance stores its own amounts ✓
```

## Debugging Quick Reference

| Step | Prompt |
|------|--------|
| Find code | "Where is the code that handles [feature]?" |
| Trace flow | "Show me the flow from request to [operation]" |
| Check races | "Could there be a race condition in [code]?" |
| Find similar | "Are there other places with the same pattern?" |
| Check model | "What's stored in the [Model] struct?" |
| Find callers | "What calls [function]? Where might this break?" |
| Impact analysis | "If I change this, what else is affected?" |
| Find examples | "How have we solved similar problems?" |
| Find tests | "What tests exist for [feature]?" |
| Edge cases | "What edge cases should I consider?" |
