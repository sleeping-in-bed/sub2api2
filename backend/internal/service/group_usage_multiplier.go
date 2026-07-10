package service

func applyGroupTokenMultiplierToUsageTokens(group *Group, tokens UsageTokens) UsageTokens {
	if group == nil {
		return tokens
	}

	tokens.InputTokens = scaleUsageTokenCount(tokens.InputTokens, group.InputTokenMultiplierOrDefault())
	tokens.OutputTokens = scaleUsageTokenCount(tokens.OutputTokens, group.OutputTokenMultiplierOrDefault())
	tokens.CacheReadTokens = scaleUsageTokenCount(tokens.CacheReadTokens, group.CacheReadTokenMultiplierOrDefault())

	cacheCreationMultiplier := group.CacheCreationTokenMultiplierOrDefault()
	if tokens.CacheCreation5mTokens > 0 || tokens.CacheCreation1hTokens > 0 {
		tokens.CacheCreation5mTokens = scaleUsageTokenCount(tokens.CacheCreation5mTokens, cacheCreationMultiplier)
		tokens.CacheCreation1hTokens = scaleUsageTokenCount(tokens.CacheCreation1hTokens, cacheCreationMultiplier)
		tokens.CacheCreationTokens = tokens.CacheCreation5mTokens + tokens.CacheCreation1hTokens
	} else {
		tokens.CacheCreationTokens = scaleUsageTokenCount(tokens.CacheCreationTokens, cacheCreationMultiplier)
	}

	return tokens
}

func applyGroupHiddenRateMultipliersToCostBreakdown(group *Group, cost *CostBreakdown) {
	if group == nil || cost == nil || cost.TotalCost <= 0 || cost.ActualCost <= 0 {
		return
	}

	if cost.InputCost <= 0 &&
		cost.OutputCost <= 0 &&
		cost.CacheCreationCost <= 0 &&
		cost.CacheReadCost <= 0 &&
		cost.ImageOutputCost <= 0 {
		return
	}

	publicRateMultiplier := cost.ActualCost / cost.TotalCost
	cost.ActualCost =
		(cost.InputCost * publicRateMultiplier * group.HiddenInputRateMultiplierOrDefault()) +
		(cost.OutputCost * publicRateMultiplier * group.HiddenOutputRateMultiplierOrDefault()) +
		(cost.CacheCreationCost * publicRateMultiplier * group.HiddenCacheCreationRateMultiplierOrDefault()) +
		(cost.CacheReadCost * publicRateMultiplier * group.HiddenCacheReadRateMultiplierOrDefault()) +
		(cost.ImageOutputCost * publicRateMultiplier)
}
