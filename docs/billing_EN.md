# Billing Service

## Module Overview

The Billing Service is the core billing engine of the Smart Park system, responsible for processing all parking fee calculations. The service adopts a rule engine design, supporting flexible configuration of multiple billing rules including hourly billing, time-based discounts, monthly pass management, VIP discounts, etc., meeting the personalized billing needs of different parking lots.

## Core Functions

### 1. Billing Rule Engine

#### Rule Types

| Rule Type | Description | Application Scenario |
|-----------|-------------|---------------------|
| `time` | Hourly billing | Standard temporary vehicle charging |
| `period` | Time-based billing | Night discounts, holiday charging |
| `monthly` | Monthly pass billing | Monthly vehicle management |
| `vip` | VIP billing | Privileged vehicle discounts |
| `discount` | Discount rules | Coupons, promotional discounts |
| `exemption` | Free rules | Free periods, special vehicles |

#### Rule Structure

```go
// BillingRule - Billing rule
type BillingRule struct {
    ID         uuid.UUID
    LotID      uuid.UUID              // Associated parking lot
    RuleName   string                 // Rule name
    RuleType   string                 // Rule type
    Conditions string                 // Conditions JSON
    Actions    string                 // Actions JSON
    Priority   int                    // Priority (higher number = higher priority)
    IsActive   bool                   // Whether enabled
}

// Condition - Condition definition
type Condition struct {
    Type       string                 // and/or/vehicle_type/duration_min/time_range/day_of_week/holiday
    Field      string                 // Field name
    Operator   string                 // Operator: ==/!=/>/</>=/<=
    Value      interface{}            // Value
    Conditions []*Condition           // Nested conditions
}

// Action - Billing action
type Action struct {
    Type    string   // fixed/per_hour/per_minute/percentage/cap/ceil/max_daily/min_charge/free_duration
    Amount  float64  // Amount
    Percent float64  // Percentage
    Unit    string   // Unit
    Cap     float64  // Cap amount
    Value   float64  // Specific value (e.g., free duration seconds)
}
```

### 2. Condition Evaluation Engine

#### Supported Condition Types

```go
// Vehicle type condition
{Type: "vehicle_type", Value: "temporary"}

// Duration condition (minutes)
{Type: "duration_min", Operator: "gte", Value: 60}

// Time range condition
{Type: "time_range", Value: {"start": 22.0, "end": 8.0}}

// Day of week condition
{Type: "day_of_week", Value: [1, 2, 3, 4, 5]}  // Weekdays

// Holiday condition
{Type: "holiday", Value: true}

// Combined condition
{
    Type: "and",
    Conditions: [
        {Type: "vehicle_type", Value: "temporary"},
        {Type: "duration_min", Operator: "gte", Value: 30}
    ]
}
```

#### Condition Evaluation Implementation

```go
func EvaluateCondition(cond *Condition, ctx *BillingContext) bool {
    switch cond.Type {
    case "and":
        for _, c := range cond.Conditions {
            if !EvaluateCondition(c, ctx) {
                return false
            }
        }
        return true
        
    case "vehicle_type":
        return ctx.VehicleType == cond.Value
        
    case "duration_min":
        minutes := ctx.Duration.Minutes()
        switch cond.Operator {
        case "gte":
            return minutes >= cond.Value.(float64)
        case "lte":
            return minutes <= cond.Value.(float64)
        }
        
    case "time_range":
        hour := float64(ctx.ExitTime.Hour()) + float64(ctx.ExitTime.Minute())/60.0
        start := cond.Value.(map[string]interface{})["start"].(float64)
        end := cond.Value.(map[string]interface{})["end"].(float64)
        return hour >= start && hour <= end
        
    // ... other condition types
    }
}
```

### 3. Billing Action Execution

#### Supported Action Types

```go
func applyActions(actions []*Action, duration time.Duration, exitTime time.Time) float64 {
    var amount float64
    hours := duration.Hours()
    minutes := duration.Minutes()
    
    for _, a := range actions {
        switch a.Type {
        case "fixed":           // Fixed fee
            amount += a.Amount
            
        case "per_hour":        // Per hour billing
            amount += hours * a.Amount
            
        case "per_minute":      // Per minute billing
            amount += minutes * a.Amount
            
        case "percentage":      // Percentage discount
            amount -= amount * (a.Percent / 100)
            
        case "cap":             // Cap amount
            if amount > a.Cap {
                amount = a.Cap
            }
            
        case "max_daily":       // Daily cap
            days := int(hours/24) + 1
            maxAmount := a.Amount * float64(days)
            if amount > maxAmount {
                amount = maxAmount
            }
            
        case "min_charge":      // Minimum charge
            if amount < a.Amount {
                amount = a.Amount
            }
            
        case "free_duration":   // Free duration
            freeMinutes := a.Value / 60
            if duration.Minutes() <= float64(freeMinutes) {
                amount = 0
            }
            
        case "first_hour_free": // First hour free
            if hours <= 1 {
                amount = 0
            }
        }
    }
    
    return amount
}
```

### 4. Rule Priority and Stacking

#### Rule Matching Strategy

```go
func (uc *BillingUseCase) CalculateFee(ctx context.Context, req *v1.CalculateFeeRequest) (*v1.BillData, error) {
    // 1. Get all active rules
    rules, _ := uc.repo.GetRulesByLotID(ctx, lotID)
    
    // 2. Sort by priority
    sort.Slice(rules, func(i, j int) bool {
        return rules[i].Priority > rules[j].Priority
    })
    
    var baseAmount float64
    var discountAmount float64
    
    for _, rule := range rules {
        // 3. Condition matching
        cond, _ := ParseConditions(rule.Conditions)
        if !EvaluateCondition(cond, billingCtx) {
            continue
        }
        
        // 4. Execute actions
        actions, _ := ParseActions(rule.Actions)
        ruleAmount := applyActions(actions, duration, exitTime)
        
        // 5. Rule stacking strategy
        switch rule.RuleType {
        case "base", "time":
            // Base fee takes minimum value (mutually exclusive)
            if baseAmount == 0 || ruleAmount < baseAmount {
                baseAmount = ruleAmount
            }
        case "discount", "exemption":
            // Discounts can stack
            discountAmount += ruleAmount
        case "monthly":
            // Monthly pass full exemption
            if req.VehicleType == "monthly" {
                discountAmount = baseAmount
            }
        }
    }
    
    // 6. Calculate final amount
    finalAmount := baseAmount - discountAmount
    if finalAmount < 0 {
        finalAmount = 0
    }
    
    return &v1.BillData{
        BaseAmount:     baseAmount,
        DiscountAmount: discountAmount,
        FinalAmount:    finalAmount,
    }, nil
}
```

## Application Scenarios

### Scenario 1: Standard Temporary Vehicle Billing

**Requirements**:
- First hour: $5
- Subsequent hours: $3/hour
- Daily cap: $50
- Free for first 15 minutes

**Rule Configuration**:
```json
{
  "ruleName": "Standard Temporary Vehicle",
  "ruleType": "time",
  "conditions": {
    "type": "vehicle_type",
    "value": "temporary"
  },
  "actions": [
    {"type": "free_duration", "value": 900},
    {"type": "fixed", "amount": 5},
    {"type": "per_hour", "amount": 3},
    {"type": "max_daily", "amount": 50}
  ],
  "priority": 100
}
```

### Scenario 2: Night Discount

**Requirements**:
- 50% discount for exits between 22:00-08:00
- Can stack with standard billing

**Rule Configuration**:
```json
{
  "ruleName": "Night Discount",
  "ruleType": "discount",
  "conditions": {
    "type": "time_range",
    "value": {"start": 22.0, "end": 8.0}
  },
  "actions": [
    {"type": "percentage", "percent": 50}
  ],
  "priority": 50
}
```

### Scenario 3: Monthly Pass Vehicles

**Requirements**:
- Free during monthly pass validity period
- Charge as temporary vehicle after expiration

**Rule Configuration**:
```json
{
  "ruleName": "Monthly Pass Vehicle",
  "ruleType": "monthly",
  "conditions": {
    "type": "vehicle_type",
    "value": "monthly"
  },
  "actions": [
    {"type": "fixed", "amount": 0}
  ],
  "priority": 200
}
```

**Expiration Handling**:
```go
// Validate monthly pass validity when exiting
if vehicle.VehicleType == VehicleTypeMonthly {
    if vehicle.MonthlyValidUntil != nil && vehicle.MonthlyValidUntil.After(time.Now()) {
        finalAmount = 0  // Valid period, free
    } else {
        // Expired mark, recalculate as temporary vehicle
        record.Metadata["chargeAs"] = VehicleTypeTemporary
        // Recalculate using billing engine
        feeResult, _ = uc.billingClient.CalculateFee(ctx, record, VehicleTypeTemporary)
    }
}
```

## Technical Challenges and Solutions

### Challenge 1: Complex Rule Performance Optimization

**Problem Description**:
Parking lots may configure dozens of billing rules, and each billing calculation requires traversal matching, which may become a performance bottleneck.

**Solution**:

1. **Rule Indexing**
   ```go
   // Index by rule type
   type RuleIndex struct {
       ByType    map[string][]*BillingRule
       ByLotID   map[uuid.UUID][]*BillingRule
       ActiveOnly []*BillingRule
   }
   ```

2. **Caching Strategy**
   ```go
   // Rate rule cache for 24 hours
   rules, err := uc.cache.GetOrSet(ctx, 
       fmt.Sprintf("parking:v1:rules:%s", lotID),
       func() ([]*BillingRule, error) {
           return uc.repo.GetRulesByLotID(ctx, lotID)
       },
       24*time.Hour,
   )
   ```

3. **Pre-compiled Conditions**
   ```go
   // Pre-parse conditions when creating rules
   func (r *BillingRule) Precompile() error {
       cond, err := ParseConditions(r.Conditions)
       if err != nil {
           return err
       }
       r.parsedCondition = cond
       return nil
   }
   ```

### Challenge 2: Cross-Day Billing Processing

**Problem Description**:
Vehicles may park across multiple days, requiring correct handling of daily caps and cross-day billing.

**Solution**:

```go
func calculateCrossDayFee(entryTime, exitTime time.Time, dailyCap float64) float64 {
    totalDays := int(exitTime.Sub(entryTime).Hours() / 24) + 1
    var totalAmount float64
    
    for i := 0; i < totalDays; i++ {
        dayStart := entryTime.Add(time.Duration(i) * 24 * time.Hour)
        dayEnd := dayStart.Add(24 * time.Hour)
        if dayEnd.After(exitTime) {
            dayEnd = exitTime
        }
        
        dayDuration := dayEnd.Sub(dayStart)
        dayAmount := calculateDailyFee(dayDuration)
        
        // Daily cap
        if dayAmount > dailyCap {
            dayAmount = dailyCap
        }
        
        totalAmount += dayAmount
    }
    
    return totalAmount
}
```

### Challenge 3: Rule Conflict Handling

**Problem Description**:
Multiple rules may match simultaneously, requiring clear conflict resolution strategies.

**Solution**:

1. **Priority Mechanism**
   - Higher number = higher priority
   - Same priority sorted by creation time

2. **Rule Type Mutual Exclusion Strategy**
   ```go
   // Base fee rules are mutually exclusive (take minimum)
   if rule.RuleType == "base" || rule.RuleType == "time" {
       if baseAmount == 0 || ruleAmount < baseAmount {
           baseAmount = ruleAmount
       }
   }
   
   // Discount rules can stack
   if rule.RuleType == "discount" {
       discountAmount += ruleAmount
   }
   ```

3. **Rule Preview Feature**
   - Admin panel provides rule preview
   - Simulate fee calculation for various scenarios
   - Detect rule conflicts in advance

### Challenge 4: Dynamic Rule Effectiveness

**Problem Description**:
Billing rules may be adjusted during operation, requiring adjustments not to affect ongoing parking records.

**Solution**:

1. **Version Control**
   ```sql
   ALTER TABLE billing_rules ADD COLUMN version INT DEFAULT 1;
   ALTER TABLE billing_rules ADD COLUMN effective_from TIMESTAMP;
   ALTER TABLE billing_rules ADD COLUMN effective_to TIMESTAMP;
   ```

2. **Snapshot Mechanism**
   ```go
   // Record current rate version at entry
   record.BillingRuleVersion = getCurrentRuleVersion(lotID)
   
   // Use entry-time version for exit calculation
   rules := uc.repo.GetRulesByVersion(ctx, lotID, record.BillingRuleVersion)
   ```

3. **Canary Release**
   - New rules take effect for partial vehicles first
   - Full release after observation confirms no issues

## API Documentation

### Calculate Fee

```http
POST /api/v1/billing/calculate
Content-Type: application/json

{
  "recordId": "rec_xxx",
  "lotId": "lot_xxx",
  "entryTime": 1710916800,
  "exitTime": 1710924000,
  "vehicleType": "temporary"
}
```

**Response**:
```json
{
  "code": 0,
  "data": {
    "recordId": "rec_xxx",
    "baseAmount": 15.00,
    "discountAmount": 5.00,
    "finalAmount": 10.00,
    "appliedRules": [
      {
        "ruleId": "rule_001",
        "ruleName": "Standard Temporary Vehicle",
        "amount": 15.00
      },
      {
        "ruleId": "rule_003",
        "ruleName": "Night Discount",
        "amount": -5.00
      }
    ]
  }
}
```

### Create Billing Rule

```http
POST /api/v1/admin/billing/rules
Content-Type: application/json

{
  "lotId": "lot_xxx",
  "ruleName": "Weekday Rate",
  "ruleType": "time",
  "conditionsJson": "{\"type\":\"vehicle_type\",\"value\":\"temporary\"}",
  "actionsJson": "[{\"type\":\"fixed\",\"amount\":5},{\"type\":\"per_hour\",\"amount\":3}]",
  "priority": 100,
  "isActive": true
}
```

## Configuration

```yaml
# configs/billing.yaml
server:
  http:
    addr: 0.0.0.0:8002
  grpc:
    addr: 0.0.0.0:9002

data:
  database:
    driver: postgres
    source: postgres://postgres:postgres@localhost:5432/parking?sslmode=disable

billing:
  defaultCurrency: "CNY"
  minChargeAmount: 0.01
  maxChargeAmount: 10000.00
  roundDecimals: 2
```

## Monitoring Metrics

| Metric | Type | Description |
|--------|------|-------------|
| billing_calculate_total | Counter | Total billing calculations |
| billing_calculate_duration | Histogram | Billing calculation duration |
| billing_rule_matches | Counter | Rule match count (by rule ID) |
| billing_amount_histogram | Histogram | Fee distribution |

## Related Documentation

- [Vehicle Service Documentation](vehicle_EN.md)
- [Payment Service Documentation](payment_EN.md)
- [Deployment Documentation](deployment_EN.md)
