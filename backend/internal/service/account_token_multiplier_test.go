package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyAccountTokenMultiplierUpdates(t *testing.T) {
	input := 1.25
	output := 0.8
	cacheCreation := 1.5
	cacheRead := 0.6

	extra, err := applyAccountTokenMultiplierUpdates(nil, &input, &output, &cacheCreation, &cacheRead)

	require.NoError(t, err)
	require.Equal(t, input, extra[accountInputTokenMultiplierExtraKey])
	require.Equal(t, output, extra[accountOutputTokenMultiplierExtraKey])
	require.Equal(t, cacheCreation, extra[accountCacheCreationTokenMultiplierExtraKey])
	require.Equal(t, cacheRead, extra[accountCacheReadTokenMultiplierExtraKey])
}

func TestApplyAccountTokenMultiplierUpdatesPreservesExistingExtra(t *testing.T) {
	extra := map[string]any{"base_rpm": 100}
	input := 1.2

	updated, err := applyAccountTokenMultiplierUpdates(extra, &input, nil, nil, nil)

	require.NoError(t, err)
	require.Equal(t, 100, updated["base_rpm"])
	require.Equal(t, input, updated[accountInputTokenMultiplierExtraKey])
}

func TestApplyAccountTokenMultiplierUpdatesRejectsNonPositiveValueWithoutMutation(t *testing.T) {
	extra := map[string]any{"base_rpm": 100}
	input := 1.2
	cacheRead := 0.0

	updated, err := applyAccountTokenMultiplierUpdates(extra, &input, nil, nil, &cacheRead)

	require.ErrorContains(t, err, accountCacheReadTokenMultiplierExtraKey)
	require.Nil(t, updated)
	require.Equal(t, map[string]any{"base_rpm": 100}, extra)
}
