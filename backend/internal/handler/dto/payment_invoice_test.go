package dto

import (
	"encoding/json"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestPaymentInvoiceFromEntBuildsStableResponseWithoutStorageMetadata(t *testing.T) {
	storageKey := "invoices/7/private.pdf"
	sha256 := "secret-checksum"
	invoice := &dbent.PaymentInvoice{
		ID:          7,
		UserID:      8,
		TitleName:   "MindAI",
		TaxID:       "91310000MA1234567X",
		Status:      "ISSUED",
		RequestedAt: time.Unix(100, 0),
		StorageKey:  &storageKey,
		Sha256:      &sha256,
		Edges: dbent.PaymentInvoiceEdges{Orders: []*dbent.PaymentOrder{
			{ID: 1, Amount: 60, PayAmount: 55},
			nil,
			{ID: 2, Amount: 50, PayAmount: 45},
		}},
	}

	result := PaymentInvoiceFromEnt(invoice)
	require.NotNil(t, result)
	require.Equal(t, 2, result.OrderCount)
	require.Equal(t, 110.0, result.TotalAmount)
	require.Equal(t, 100.0, result.TotalPayAmount)
	require.Len(t, result.Orders, 2)
	require.Equal(t, "d5a5b456-1618-5dd4-bc70-5399b4a144a5", result.Orders[0].OrderUUID)
	require.Equal(t, "4d0b10bb-b402-50e2-97b2-5f6e09e90dd3", result.Orders[1].OrderUUID)

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "storage_key")
	require.NotContains(t, string(payload), "sha256")
}
