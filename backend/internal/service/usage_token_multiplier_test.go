package service

import "testing"

func TestAccountComponentTokenMultiplier_DefaultAndConfigured(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		account *Account
		want    float64
	}{
		{name: "nil account", account: nil, want: 1},
		{name: "missing extra", account: &Account{}, want: 1},
		{name: "missing key", account: &Account{Extra: map[string]any{}}, want: 1},
		{name: "configured float", account: &Account{Extra: map[string]any{accountInputTokenMultiplierExtraKey: 1.25}}, want: 1.25},
		{name: "configured string", account: &Account{Extra: map[string]any{accountInputTokenMultiplierExtraKey: "1.5"}}, want: 1.5},
		{name: "non-positive falls back", account: &Account{Extra: map[string]any{accountInputTokenMultiplierExtraKey: 0}}, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.account.InputTokenMultiplier(); got != tt.want {
				t.Fatalf("InputTokenMultiplier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccountComponentTokenMultiplier_NoFallbackBetweenFields(t *testing.T) {
	t.Parallel()

	account := &Account{
		Extra: map[string]any{
			accountInputTokenMultiplierExtraKey:     2.0,
			accountCacheReadTokenMultiplierExtraKey: 4.0,
		},
	}

	if got := account.InputTokenMultiplier(); got != 2 {
		t.Fatalf("InputTokenMultiplier() = %v, want 2", got)
	}
	if got := account.OutputTokenMultiplier(); got != 1 {
		t.Fatalf("OutputTokenMultiplier() = %v, want 1", got)
	}
	if got := account.CacheCreationTokenMultiplier(); got != 1 {
		t.Fatalf("CacheCreationTokenMultiplier() = %v, want 1", got)
	}
	if got := account.CacheReadTokenMultiplier(); got != 4 {
		t.Fatalf("CacheReadTokenMultiplier() = %v, want 4", got)
	}
}

func TestApplyAccountTokenMultiplierToUsageLog(t *testing.T) {
	t.Parallel()

	account := &Account{
		Extra: map[string]any{
			accountInputTokenMultiplierExtraKey:         1.5,
			accountOutputTokenMultiplierExtraKey:        2.0,
			accountCacheCreationTokenMultiplierExtraKey: 3.0,
			accountCacheReadTokenMultiplierExtraKey:     4.0,
		},
	}
	log := &UsageLog{
		InputTokens:           3,
		OutputTokens:          5,
		CacheCreationTokens:   7,
		CacheReadTokens:       9,
		CacheCreation5mTokens: 2,
		CacheCreation1hTokens: 3,
		ImageOutputTokens:     4,
	}

	applyAccountTokenMultiplierToUsageLog(account, log)

	if log.InputTokens != 5 {
		t.Fatalf("InputTokens = %d, want 5", log.InputTokens)
	}
	if log.OutputTokens != 10 {
		t.Fatalf("OutputTokens = %d, want 10", log.OutputTokens)
	}
	if log.CacheReadTokens != 36 {
		t.Fatalf("CacheReadTokens = %d, want 36", log.CacheReadTokens)
	}
	if log.CacheCreation5mTokens != 6 {
		t.Fatalf("CacheCreation5mTokens = %d, want 6", log.CacheCreation5mTokens)
	}
	if log.CacheCreation1hTokens != 9 {
		t.Fatalf("CacheCreation1hTokens = %d, want 9", log.CacheCreation1hTokens)
	}
	if log.CacheCreationTokens != 15 {
		t.Fatalf("CacheCreationTokens = %d, want 15", log.CacheCreationTokens)
	}
}

func TestApplyAccountTokenMultiplierToCostBreakdown(t *testing.T) {
	t.Parallel()

	account := &Account{
		Extra: map[string]any{
			accountInputTokenMultiplierExtraKey:         2.0,
			accountOutputTokenMultiplierExtraKey:        3.0,
			accountCacheCreationTokenMultiplierExtraKey: 4.0,
			accountCacheReadTokenMultiplierExtraKey:     5.0,
		},
	}
	cost := &CostBreakdown{
		InputCost:         1,
		OutputCost:        2,
		ImageOutputCost:   3,
		CacheCreationCost: 4,
		CacheReadCost:     5,
		TotalCost:         15,
		ActualCost:        22.5,
	}

	applyAccountTokenMultiplierToCostBreakdown(account, cost)

	if cost.InputCost != 2 {
		t.Fatalf("InputCost = %v, want 2", cost.InputCost)
	}
	if cost.OutputCost != 6 {
		t.Fatalf("OutputCost = %v, want 6", cost.OutputCost)
	}
	if cost.ImageOutputCost != 3 {
		t.Fatalf("ImageOutputCost = %v, want 3", cost.ImageOutputCost)
	}
	if cost.CacheCreationCost != 16 {
		t.Fatalf("CacheCreationCost = %v, want 16", cost.CacheCreationCost)
	}
	if cost.CacheReadCost != 25 {
		t.Fatalf("CacheReadCost = %v, want 25", cost.CacheReadCost)
	}
	if cost.TotalCost != 52 {
		t.Fatalf("TotalCost = %v, want 52", cost.TotalCost)
	}
	if cost.ActualCost != 78 {
		t.Fatalf("ActualCost = %v, want 78", cost.ActualCost)
	}
}

func TestApplyAccountTokenMultiplierToUsageLog_ScalesAggregateWhenNoBreakdown(t *testing.T) {
	t.Parallel()

	account := &Account{
		Extra: map[string]any{
			accountCacheCreationTokenMultiplierExtraKey: 0.6,
		},
	}
	log := &UsageLog{
		CacheCreationTokens: 5,
	}

	applyAccountTokenMultiplierToUsageLog(account, log)

	if log.CacheCreationTokens != 3 {
		t.Fatalf("CacheCreationTokens = %d, want 3", log.CacheCreationTokens)
	}
	if log.CacheCreation5mTokens != 0 || log.CacheCreation1hTokens != 0 {
		t.Fatalf("unexpected cache breakdown values: 5m=%d 1h=%d", log.CacheCreation5mTokens, log.CacheCreation1hTokens)
	}
}
