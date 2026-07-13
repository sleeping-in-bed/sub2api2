package dto

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupMultiplierDTOVisibility(t *testing.T) {
	group := &service.Group{
		ID:                                    1,
		InputTokenMultiplier:                  1.2,
		HiddenInputRateMultiplier:             0.8,
		HiddenOutputRateMultiplier:            0.7,
		HiddenCacheCreationRateMultiplier:     0.6,
		HiddenCacheReadRateMultiplier:         0.5,
	}

	publicJSON, err := json.Marshal(GroupFromService(group))
	require.NoError(t, err)
	require.Contains(t, string(publicJSON), `"input_token_multiplier":1.2`)
	require.NotContains(t, string(publicJSON), "hidden_input_rate_multiplier")

	adminJSON, err := json.Marshal(GroupFromServiceAdmin(group))
	require.NoError(t, err)
	require.Contains(t, string(adminJSON), `"hidden_input_rate_multiplier":0.8`)
	require.Contains(t, string(adminJSON), `"hidden_cache_read_rate_multiplier":0.5`)
}
