package service

import "math"

func applyAccountTokenMultiplierToUsageLog(account *Account, usageLog *UsageLog) {
	if account == nil || usageLog == nil {
		return
	}

	multiplier := account.TokenMultiplier()
	if multiplier == 1 {
		return
	}

	usageLog.InputTokens = scaleUsageTokenCount(usageLog.InputTokens, multiplier)
	usageLog.OutputTokens = scaleUsageTokenCount(usageLog.OutputTokens, multiplier)
	usageLog.CacheReadTokens = scaleUsageTokenCount(usageLog.CacheReadTokens, multiplier)

	if usageLog.CacheCreation5mTokens > 0 || usageLog.CacheCreation1hTokens > 0 {
		usageLog.CacheCreation5mTokens = scaleUsageTokenCount(usageLog.CacheCreation5mTokens, multiplier)
		usageLog.CacheCreation1hTokens = scaleUsageTokenCount(usageLog.CacheCreation1hTokens, multiplier)
		usageLog.CacheCreationTokens = usageLog.CacheCreation5mTokens + usageLog.CacheCreation1hTokens
	} else {
		usageLog.CacheCreationTokens = scaleUsageTokenCount(usageLog.CacheCreationTokens, multiplier)
	}

	usageLog.ImageOutputTokens = scaleUsageTokenCount(usageLog.ImageOutputTokens, multiplier)
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
