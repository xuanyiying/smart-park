package biz

import (
	"testing"
	"time"
)

func TestParseConditions(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "empty string",
			json:    "",
			wantErr: false,
		},
		{
			name:    "valid vehicle_type condition",
			json:    `{"type": "vehicle_type", "value": "monthly"}`,
			wantErr: false,
		},
		{
			name:    "valid duration condition",
			json:    `{"type": "duration_min", "operator": "gte", "value": 60}`,
			wantErr: false,
		},
		{
			name:    "valid time_range condition",
			json:    `{"type": "time_range", "value": {"start": 22.0, "end": 8.0}}`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{"type": "invalid"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond, err := ParseConditions(tt.json)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConditions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cond == nil && tt.json != "" {
				t.Error("ParseConditions() returned nil for non-empty input")
			}
		})
	}
}

func TestParseActions(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		wantLen int
	}{
		{
			name:    "empty string",
			json:    "",
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "valid fixed action",
			json:    `[{"type": "fixed", "amount": 5.0}]`,
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "valid per_hour action",
			json:    `[{"type": "per_hour", "amount": 2.0}]`,
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "valid percentage action",
			json:    `[{"type": "percentage", "percent": 20.0}]`,
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "multiple actions",
			json:    `[{"type": "per_hour", "amount": 2.0}, {"type": "cap", "cap": 50.0}]`,
			wantErr: false,
			wantLen: 2,
		},
		{
			name:    "invalid json",
			json:    `[{"type": "invalid"]`,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, err := ParseActions(tt.json)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseActions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(actions) != tt.wantLen {
				t.Errorf("ParseActions() returned %d actions, want %d", len(actions), tt.wantLen)
			}
		})
	}
}

func TestEvaluateCondition(t *testing.T) {
	billingCtx := &BillingContext{
		VehicleType: "monthly",
		Duration:    90 * time.Minute,
		ExitTime:    time.Date(2026, 3, 26, 10, 30, 0, 0, time.UTC),
		IsHoliday:   false,
	}

	tests := []struct {
		name   string
		cond   *Condition
		ctx    *BillingContext
		expect bool
	}{
		{
			name:   "nil condition",
			cond:   nil,
			ctx:    billingCtx,
			expect: true,
		},
		{
			name: "vehicle_type match",
			cond: &Condition{
				Type:  "vehicle_type",
				Value: "monthly",
			},
			ctx:    billingCtx,
			expect: true,
		},
		{
			name: "vehicle_type no match",
			cond: &Condition{
				Type:  "vehicle_type",
				Value: "vip",
			},
			ctx:    billingCtx,
			expect: false,
		},
		{
			name: "duration_min gte true",
			cond: &Condition{
				Type:     "duration_min",
				Operator: "gte",
				Value:    60.0,
			},
			ctx:    billingCtx,
			expect: true,
		},
		{
			name: "duration_min gte false",
			cond: &Condition{
				Type:     "duration_min",
				Operator: "gte",
				Value:    120.0,
			},
			ctx:    billingCtx,
			expect: false,
		},
		{
			name: "time_range within range",
			cond: &Condition{
				Type: "time_range",
				Value: map[string]interface{}{
					"start": 9.0,
					"end":   18.0,
				},
			},
			ctx:    billingCtx,
			expect: true,
		},
		{
			name: "time_range outside range",
			cond: &Condition{
				Type: "time_range",
				Value: map[string]interface{}{
					"start": 22.0,
					"end":   8.0,
				},
			},
			ctx:    billingCtx,
			expect: false,
		},
		{
			name: "day_of_week match",
			cond: &Condition{
				Type:  "day_of_week",
				Value: []interface{}{4.0}, // Thursday
			},
			ctx:    billingCtx,
			expect: true,
		},
		{
			name: "day_of_week no match",
			cond: &Condition{
				Type:  "day_of_week",
				Value: []interface{}{1.0}, // Monday
			},
			ctx:    billingCtx,
			expect: false,
		},
		{
			name: "and conditions all true",
			cond: &Condition{
				Type: "and",
				Conditions: []*Condition{
					{Type: "vehicle_type", Value: "monthly"},
					{Type: "duration_min", Operator: "gte", Value: 60.0},
				},
			},
			ctx:    billingCtx,
			expect: true,
		},
		{
			name: "and conditions one false",
			cond: &Condition{
				Type: "and",
				Conditions: []*Condition{
					{Type: "vehicle_type", Value: "monthly"},
					{Type: "vehicle_type", Value: "vip"},
				},
			},
			ctx:    billingCtx,
			expect: false,
		},
		{
			name: "or conditions one true",
			cond: &Condition{
				Type: "or",
				Conditions: []*Condition{
					{Type: "vehicle_type", Value: "vip"},
					{Type: "vehicle_type", Value: "monthly"},
				},
			},
			ctx:    billingCtx,
			expect: true,
		},
		{
			name: "or conditions all false",
			cond: &Condition{
				Type: "or",
				Conditions: []*Condition{
					{Type: "vehicle_type", Value: "vip"},
					{Type: "vehicle_type", Value: "temporary"},
				},
			},
			ctx:    billingCtx,
			expect: false,
		},
		{
			name:   "unknown condition type",
			cond:   &Condition{Type: "unknown"},
			ctx:    billingCtx,
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateCondition(tt.cond, tt.ctx)
			if got != tt.expect {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestApplyActions(t *testing.T) {
	duration := 120 * time.Minute // 2 hours

	tests := []struct {
		name     string
		actions  []*Action
		duration time.Duration
		expected float64
	}{
		{
			name:     "fixed amount",
			actions:  []*Action{{Type: "fixed", Amount: 10.0}},
			duration: duration,
			expected: 10.0,
		},
		{
			name:     "per_hour",
			actions:  []*Action{{Type: "per_hour", Amount: 5.0}},
			duration: duration,
			expected: 10.0, // 2 hours * 5 = 10
		},
		{
			name:     "per_minute",
			actions:  []*Action{{Type: "per_minute", Amount: 0.1}},
			duration: duration,
			expected: 12.0, // 120 minutes * 0.1 = 12
		},
		{
			name:     "percentage discount",
			actions:  []*Action{{Type: "fixed", Amount: 100.0}, {Type: "percentage", Percent: 20.0}},
			duration: duration,
			expected: 80.0, // 100 - 20%
		},
		{
			name:     "cap applied",
			actions:  []*Action{{Type: "per_hour", Amount: 50.0}, {Type: "cap", Cap: 80.0}},
			duration: duration,
			expected: 80.0, // 2 * 50 = 100, capped to 80
		},
		{
			name:     "multiple actions chain",
			actions:  []*Action{{Type: "per_hour", Amount: 10.0}, {Type: "percentage", Percent: 10.0}, {Type: "cap", Cap: 50.0}},
			duration: duration,
			expected: 18.0, // 2*10=20, 20-10%=18, within cap
		},
		{
			name:     "empty actions",
			actions:  []*Action{},
			duration: duration,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyActions(tt.actions, tt.duration)
			if got != tt.expected {
				t.Errorf("applyActions() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalculateDefaultFee(t *testing.T) {
	tests := []struct {
		name   string
		hours  float64
		expect float64
	}{
		{
			name:   "less than 1 hour",
			hours:  0.5,
			expect: 5.0,
		},
		{
			name:   "exactly 1 hour",
			hours:  1.0,
			expect: 2.0,
		},
		{
			name:   "2 hours",
			hours:  2.0,
			expect: 4.0,
		},
		{
			name:   "5 hours",
			hours:  5.0,
			expect: 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDefaultFee(tt.hours)
			if got != tt.expect {
				t.Errorf("calculateDefaultFee() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestCeilToDecimal(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		decimals int
		expected float64
	}{
		{
			name:     "2 decimals",
			amount:   10.123,
			decimals: 2,
			expected: 10.13,
		},
		{
			name:     "0 decimals",
			amount:   10.9,
			decimals: 0,
			expected: 11.0,
		},
		{
			name:     "already rounded",
			amount:   10.00,
			decimals: 2,
			expected: 10.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ceilToDecimal(tt.amount, tt.decimals)
			if got != tt.expected {
				t.Errorf("ceilToDecimal() = %v, want %v", got, tt.expected)
			}
		})
	}
}