package service

import "math"

func applyAccountTokenMultiplierToCostBreakdown(account *Account, cost *CostBreakdown) {
	if account == nil || cost == nil {
		return
	}

	inputMultiplier := account.InputTokenMultiplier()
	outputMultiplier := account.OutputTokenMultiplier()
	cacheCreationMultiplier := account.CacheCreationTokenMultiplier()
	cacheReadMultiplier := account.CacheReadTokenMultiplier()
	if inputMultiplier == 1 && outputMultiplier == 1 && cacheCreationMultiplier == 1 && cacheReadMultiplier == 1 {
		return
	}

	actualCostRatio := 1.0
	if cost.TotalCost > 0 && cost.ActualCost > 0 {
		actualCostRatio = cost.ActualCost / cost.TotalCost
	}

	cost.InputCost = scaleUsageCostAmount(cost.InputCost, inputMultiplier)
	cost.OutputCost = scaleUsageCostAmount(cost.OutputCost, outputMultiplier)
	cost.CacheCreationCost = scaleUsageCostAmount(cost.CacheCreationCost, cacheCreationMultiplier)
	cost.CacheReadCost = scaleUsageCostAmount(cost.CacheReadCost, cacheReadMultiplier)
	cost.TotalCost = cost.InputCost + cost.OutputCost + cost.ImageOutputCost + cost.CacheCreationCost + cost.CacheReadCost
	cost.ActualCost = cost.TotalCost * actualCostRatio
}

func applyAccountTokenMultiplierToUsageLog(account *Account, usageLog *UsageLog) {
	if account == nil || usageLog == nil {
		return
	}

	inputMultiplier := account.InputTokenMultiplier()
	outputMultiplier := account.OutputTokenMultiplier()
	cacheCreationMultiplier := account.CacheCreationTokenMultiplier()
	cacheReadMultiplier := account.CacheReadTokenMultiplier()
	if inputMultiplier == 1 && outputMultiplier == 1 && cacheCreationMultiplier == 1 && cacheReadMultiplier == 1 {
		return
	}

	usageLog.InputTokens = scaleUsageTokenCount(usageLog.InputTokens, inputMultiplier)
	usageLog.OutputTokens = scaleUsageTokenCount(usageLog.OutputTokens, outputMultiplier)
	usageLog.CacheReadTokens = scaleUsageTokenCount(usageLog.CacheReadTokens, cacheReadMultiplier)

	if usageLog.CacheCreation5mTokens > 0 || usageLog.CacheCreation1hTokens > 0 {
		usageLog.CacheCreation5mTokens = scaleUsageTokenCount(usageLog.CacheCreation5mTokens, cacheCreationMultiplier)
		usageLog.CacheCreation1hTokens = scaleUsageTokenCount(usageLog.CacheCreation1hTokens, cacheCreationMultiplier)
		usageLog.CacheCreationTokens = usageLog.CacheCreation5mTokens + usageLog.CacheCreation1hTokens
	} else {
		usageLog.CacheCreationTokens = scaleUsageTokenCount(usageLog.CacheCreationTokens, cacheCreationMultiplier)
	}
}

func scaleUsageTokenCount(raw int, multiplier float64) int {
	if raw <= 0 {
		return 0
	}
	if multiplier <= 0 {
		return 0
	}
	return int(math.Round(float64(raw) * multiplier))
}

func scaleUsageCostAmount(raw float64, multiplier float64) float64 {
	if raw <= 0 {
		return 0
	}
	if multiplier <= 0 {
		return 0
	}
	return raw * multiplier
}
