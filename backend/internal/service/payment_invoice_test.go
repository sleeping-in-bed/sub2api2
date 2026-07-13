package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestPaymentInvoiceRequestClaimsMultipleOrdersOnce(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentService{entClient: client}
	user := createPaymentInvoiceTestUser(t, ctx, client, "invoice-request@example.com")
	first := createPaymentInvoiceTestOrder(t, ctx, client, user, "request-1", 60)
	second := createPaymentInvoiceTestOrder(t, ctx, client, user, "request-2", 50)

	invoice, err := svc.RequestInvoice(ctx, []int64{first.ID, second.ID, first.ID}, user.ID, "Example Company", "91350211M000100Y43")
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusRequested, invoice.Status)
	require.Len(t, invoice.Edges.Orders, 2)
	require.ElementsMatch(t, []int64{first.ID, second.ID}, paymentInvoiceOrderIDs(invoice.Edges.Orders))
	require.InDelta(t, 110, sumPaymentOrdersPayAmount(invoice.Edges.Orders), 0.0001)

	for _, orderID := range []int64{first.ID, second.ID} {
		order, err := client.PaymentOrder.Get(ctx, orderID)
		require.NoError(t, err)
		require.NotNil(t, order.InvoiceID)
		require.Equal(t, invoice.ID, *order.InvoiceID)
	}

	_, err = svc.RequestInvoice(ctx, []int64{first.ID, second.ID}, user.ID, "Example Company", "91350211M000100Y43")
	require.Equal(t, "INVALID_ORDER_SELECTION", infraerrors.Reason(err))

	audits, err := client.PaymentAuditLog.Query().Where(paymentauditlog.ActionEQ("INVOICE_REQUESTED")).All(ctx)
	require.NoError(t, err)
	require.Len(t, audits, 2)
	for _, audit := range audits {
		var detail struct {
			InvoiceOrderIDs []int64 `json:"invoiceOrderIDs"`
		}
		require.NoError(t, json.Unmarshal([]byte(audit.Detail), &detail))
		require.ElementsMatch(t, []int64{first.ID, second.ID}, detail.InvoiceOrderIDs)
	}
}

func TestPaymentInvoiceFailureReleasesOrders(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentService{entClient: client}
	user := createPaymentInvoiceTestUser(t, ctx, client, "invoice-failed@example.com")
	first := createPaymentInvoiceTestOrder(t, ctx, client, user, "failed-1", 70)
	second := createPaymentInvoiceTestOrder(t, ctx, client, user, "failed-2", 40)
	invoice, err := svc.RequestInvoice(ctx, []int64{first.ID, second.ID}, user.ID, "Failure Company", "91350211M000100Y43")
	require.NoError(t, err)

	failed, err := svc.MarkInvoiceFailed(ctx, invoice.ID, "invoice data rejected", "admin:1")
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusFailed, failed.Status)
	require.NotNil(t, failed.FailedReason)
	require.Equal(t, "invoice data rejected", *failed.FailedReason)

	for _, orderID := range []int64{first.ID, second.ID} {
		order, err := client.PaymentOrder.Get(ctx, orderID)
		require.NoError(t, err)
		require.Nil(t, order.InvoiceID)
	}
	amount, count, err := svc.GetInvoiceAvailableSummary(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.InDelta(t, 110, amount, 0.0001)

	_, err = svc.MarkInvoiceIssued(ctx, invoice.ID, "failed.pdf", []byte("%PDF-1.4\n%%EOF\n"), "admin:1")
	require.Equal(t, "INVALID_STATUS", infraerrors.Reason(err))
}

func TestPaymentInvoiceIssueAndDownloadEnforcesOwnership(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentService{entClient: client}
	owner := createPaymentInvoiceTestUser(t, ctx, client, "invoice-owner@example.com")
	other := createPaymentInvoiceTestUser(t, ctx, client, "invoice-other@example.com")
	order := createPaymentInvoiceTestOrder(t, ctx, client, owner, "issued-1", 120)
	invoice, err := svc.RequestInvoice(ctx, []int64{order.ID}, owner.ID, "Owner Company", "91350211M000100Y43")
	require.NoError(t, err)

	t.Setenv("DATA_DIR", t.TempDir())
	pdf := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n")
	issued, err := svc.MarkInvoiceIssued(ctx, invoice.ID, "../../owner invoice.PDF", pdf, "admin:1")
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusIssued, issued.Status)
	require.NotNil(t, issued.StorageKey)
	require.NotNil(t, issued.FileName)
	require.Equal(t, "owner_invoice.pdf", *issued.FileName)
	require.Equal(t, int64(len(pdf)), issued.ByteSize)
	sum := sha256.Sum256(pdf)
	require.NotNil(t, issued.Sha256)
	require.Equal(t, hex.EncodeToString(sum[:]), *issued.Sha256)

	_, _, err = svc.PrepareUserInvoiceDownload(ctx, invoice.ID, other.ID)
	require.Equal(t, "FORBIDDEN", infraerrors.Reason(err))

	ownerInvoice, ownerPath, err := svc.PrepareUserInvoiceDownload(ctx, invoice.ID, owner.ID)
	require.NoError(t, err)
	require.Equal(t, invoice.ID, ownerInvoice.ID)
	require.FileExists(t, ownerPath)
	stored, err := os.ReadFile(ownerPath)
	require.NoError(t, err)
	require.Equal(t, pdf, stored)

	adminInvoice, adminPath, err := svc.PrepareAdminInvoiceDownload(ctx, invoice.ID)
	require.NoError(t, err)
	require.Equal(t, invoice.ID, adminInvoice.ID)
	require.Equal(t, ownerPath, adminPath)
}

func createPaymentInvoiceTestUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) *dbent.User {
	t.Helper()
	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hash").
		SetUsername("invoice-user").
		Save(ctx)
	require.NoError(t, err)
	return user
}

func createPaymentInvoiceTestOrder(t *testing.T, ctx context.Context, client *dbent.Client, user *dbent.User, suffix string, payAmount float64) *dbent.PaymentOrder {
	t.Helper()
	now := time.Now()
	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(payAmount).
		SetPayAmount(payAmount).
		SetFeeRate(0).
		SetRechargeCode("INVOICE-" + suffix).
		SetOutTradeNo("sub2_invoice_" + suffix).
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-" + suffix).
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(now.Add(time.Hour)).
		SetPaidAt(now.Add(-time.Minute)).
		SetCompletedAt(now).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)
	return order
}
