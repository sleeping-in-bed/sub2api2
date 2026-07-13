package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyGroupTokenMultipliers(t *testing.T) {
	group := &Group{
		InputTokenMultiplier:         1.5,
		OutputTokenMultiplier:        0.5,
		CacheCreationTokenMultiplier: 2,
		CacheReadTokenMultiplier:     0.25,
	}
	raw := UsageTokens{
		InputTokens:             3,
		OutputTokens:            5,
		CacheCreationTokens:     99,
		CacheReadTokens:         10,
		CacheCreation5mTokens:   2,
		CacheCreation1hTokens:   3,
		ImageOutputTokens:       7,
	}

	adjusted := applyGroupTokenMultipliers(group, raw)

	require.Equal(t, 5, adjusted.InputTokens)
	require.Equal(t, 3, adjusted.OutputTokens)
	require.Equal(t, 10, adjusted.CacheCreationTokens)
	require.Equal(t, 4, adjusted.CacheCreation5mTokens)
	require.Equal(t, 6, adjusted.CacheCreation1hTokens)
	require.Equal(t, 3, adjusted.CacheReadTokens)
	require.Equal(t, raw.ImageOutputTokens, adjusted.ImageOutputTokens)
}

func TestApplyHiddenAndAccountTokenMultipliers(t *testing.T) {
	group := &Group{
		HiddenInputRateMultiplier:         0.8,
		HiddenOutputRateMultiplier:        0.7,
		HiddenCacheCreationRateMultiplier: 0.6,
		HiddenCacheReadRateMultiplier:     0.5,
	}
	account := &Account{Extra: map[string]any{
		accountInputTokenMultiplierExtraKey:         1.1,
		accountOutputTokenMultiplierExtraKey:        1.2,
		accountCacheCreationTokenMultiplierExtraKey: 1.3,
		accountCacheReadTokenMultiplierExtraKey:     1.4,
	}}
	cost := &CostBreakdown{
		InputCost:         10,
		OutputCost:        20,
		CacheCreationCost: 30,
		CacheReadCost:     40,
		ImageOutputCost:   50,
		TotalCost:         150,
		ActualCost:        300,
	}

	applyHiddenAndAccountTokenMultipliers(group, account, 2, cost)

	want := 10*2*0.8*1.1 + 20*2*0.7*1.2 + 30*2*0.6*1.3 + 40*2*0.5*1.4 + 50*2
	require.InDelta(t, want, cost.ActualCost, 1e-12)
	require.Equal(t, 150.0, cost.TotalCost)
}

func TestBuildUsageMultiplierSnapshot(t *testing.T) {
	group := &Group{InputTokenMultiplier: 2, HiddenInputRateMultiplier: 0.8}
	account := &Account{Extra: map[string]any{accountInputTokenMultiplierExtraKey: 1.25}}
	raw := UsageTokens{InputTokens: 10, ImageOutputTokens: 4}
	adjusted := UsageTokens{InputTokens: 20, ImageOutputTokens: 4}

	snapshot := buildUsageMultiplierSnapshot(group, account, raw, adjusted, 1.5, 0.9)

	require.Equal(t, 10, snapshot["raw_input_tokens"])
	require.Equal(t, 20, snapshot["adjusted_input_tokens"])
	require.Equal(t, 2.0, snapshot["group_input_token_multiplier"])
	require.Equal(t, 0.8, snapshot["group_hidden_input_rate_multiplier"])
	require.Equal(t, 1.25, snapshot["account_input_token_multiplier"])
	require.Equal(t, 1.5, snapshot["public_rate_multiplier"])
	require.Equal(t, 0.9, snapshot["account_rate_multiplier"])
}
