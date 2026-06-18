package service

import "testing"

func TestAccountTokenMultiplier_DefaultAndConfigured(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		account *Account
		want    float64
	}{
		{name: "nil account", account: nil, want: 1},
		{name: "missing extra", account: &Account{}, want: 1},
		{name: "missing key", account: &Account{Extra: map[string]any{}}, want: 1},
		{name: "configured float", account: &Account{Extra: map[string]any{accountTokenMultiplierExtraKey: 1.25}}, want: 1.25},
		{name: "configured string", account: &Account{Extra: map[string]any{accountTokenMultiplierExtraKey: "1.5"}}, want: 1.5},
		{name: "non-positive falls back", account: &Account{Extra: map[string]any{accountTokenMultiplierExtraKey: 0}}, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.account.TokenMultiplier(); got != tt.want {
				t.Fatalf("TokenMultiplier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyAccountTokenMultiplierToUsageLog(t *testing.T) {
	t.Parallel()

	account := &Account{
		Extra: map[string]any{
			accountTokenMultiplierExtraKey: 1.5,
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
	if log.OutputTokens != 8 {
		t.Fatalf("OutputTokens = %d, want 8", log.OutputTokens)
	}
	if log.CacheReadTokens != 14 {
		t.Fatalf("CacheReadTokens = %d, want 14", log.CacheReadTokens)
	}
	if log.CacheCreation5mTokens != 3 {
		t.Fatalf("CacheCreation5mTokens = %d, want 3", log.CacheCreation5mTokens)
	}
	if log.CacheCreation1hTokens != 5 {
		t.Fatalf("CacheCreation1hTokens = %d, want 5", log.CacheCreation1hTokens)
	}
	if log.CacheCreationTokens != 8 {
		t.Fatalf("CacheCreationTokens = %d, want 8", log.CacheCreationTokens)
	}
	if log.ImageOutputTokens != 6 {
		t.Fatalf("ImageOutputTokens = %d, want 6", log.ImageOutputTokens)
	}
}

func TestApplyAccountTokenMultiplierToUsageLog_ScalesAggregateWhenNoBreakdown(t *testing.T) {
	t.Parallel()

	account := &Account{
		Extra: map[string]any{
			accountTokenMultiplierExtraKey: 0.6,
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
