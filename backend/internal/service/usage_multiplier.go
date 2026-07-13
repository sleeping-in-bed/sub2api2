package service

import "math"

func scaleUsageTokenCount(raw int, multiplier float64) int {
	if raw <= 0 {
		return 0
	}
	return int(math.Round(float64(raw) * positiveMultiplierOrDefault(multiplier)))
}

func applyGroupTokenMultipliers(group *Group, raw UsageTokens) UsageTokens {
	adjusted := raw
	adjusted.InputTokens = scaleUsageTokenCount(raw.InputTokens, group.InputTokenMultiplierOrDefault())
	adjusted.OutputTokens = scaleUsageTokenCount(raw.OutputTokens, group.OutputTokenMultiplierOrDefault())
	adjusted.CacheReadTokens = scaleUsageTokenCount(raw.CacheReadTokens, group.CacheReadTokenMultiplierOrDefault())
	cacheCreationMultiplier := group.CacheCreationTokenMultiplierOrDefault()
	if raw.CacheCreation5mTokens > 0 || raw.CacheCreation1hTokens > 0 {
		adjusted.CacheCreation5mTokens = scaleUsageTokenCount(raw.CacheCreation5mTokens, cacheCreationMultiplier)
		adjusted.CacheCreation1hTokens = scaleUsageTokenCount(raw.CacheCreation1hTokens, cacheCreationMultiplier)
		adjusted.CacheCreationTokens = adjusted.CacheCreation5mTokens + adjusted.CacheCreation1hTokens
	} else {
		adjusted.CacheCreationTokens = scaleUsageTokenCount(raw.CacheCreationTokens, cacheCreationMultiplier)
	}
	return adjusted
}

func applyHiddenAndAccountTokenMultipliers(group *Group, account *Account, publicRateMultiplier float64, cost *CostBreakdown) {
	if cost == nil {
		return
	}
	if publicRateMultiplier < 0 {
		publicRateMultiplier = 1
	}
	cost.ActualCost =
		cost.InputCost*publicRateMultiplier*group.HiddenInputRateMultiplierOrDefault()*account.InputTokenMultiplier() +
		cost.OutputCost*publicRateMultiplier*group.HiddenOutputRateMultiplierOrDefault()*account.OutputTokenMultiplier() +
		cost.CacheCreationCost*publicRateMultiplier*group.HiddenCacheCreationRateMultiplierOrDefault()*account.CacheCreationTokenMultiplier() +
		cost.CacheReadCost*publicRateMultiplier*group.HiddenCacheReadRateMultiplierOrDefault()*account.CacheReadTokenMultiplier() +
		cost.ImageOutputCost*publicRateMultiplier
}

func buildUsageMultiplierSnapshot(group *Group, account *Account, raw, adjusted UsageTokens, publicRateMultiplier, accountRateMultiplier float64) map[string]any {
	return map[string]any{
		"raw_input_tokens": raw.InputTokens,
		"raw_output_tokens": raw.OutputTokens,
		"raw_cache_creation_tokens": raw.CacheCreationTokens,
		"raw_cache_read_tokens": raw.CacheReadTokens,
		"adjusted_input_tokens": adjusted.InputTokens,
		"adjusted_output_tokens": adjusted.OutputTokens,
		"adjusted_cache_creation_tokens": adjusted.CacheCreationTokens,
		"adjusted_cache_read_tokens": adjusted.CacheReadTokens,
		"group_input_token_multiplier": group.InputTokenMultiplierOrDefault(),
		"group_output_token_multiplier": group.OutputTokenMultiplierOrDefault(),
		"group_cache_creation_token_multiplier": group.CacheCreationTokenMultiplierOrDefault(),
		"group_cache_read_token_multiplier": group.CacheReadTokenMultiplierOrDefault(),
		"group_hidden_input_rate_multiplier": group.HiddenInputRateMultiplierOrDefault(),
		"group_hidden_output_rate_multiplier": group.HiddenOutputRateMultiplierOrDefault(),
		"group_hidden_cache_creation_rate_multiplier": group.HiddenCacheCreationRateMultiplierOrDefault(),
		"group_hidden_cache_read_rate_multiplier": group.HiddenCacheReadRateMultiplierOrDefault(),
		"account_input_token_multiplier": account.InputTokenMultiplier(),
		"account_output_token_multiplier": account.OutputTokenMultiplier(),
		"account_cache_creation_token_multiplier": account.CacheCreationTokenMultiplier(),
		"account_cache_read_token_multiplier": account.CacheReadTokenMultiplier(),
		"public_rate_multiplier": publicRateMultiplier,
		"account_rate_multiplier": accountRateMultiplier,
	}
}
