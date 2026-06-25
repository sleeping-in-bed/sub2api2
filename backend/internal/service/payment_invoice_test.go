//go:build unit

package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestRequestInvoiceCreatesRecordForCompletedOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, order := seedCompletedInvoiceOrder(t, ctx, client)
	svc := &PaymentService{entClient: client}

	invoice, err := svc.RequestInvoice(ctx, order.ID, user.ID, "上海某某科技有限公司", "91310113685471496R")
	require.NoError(t, err)
	require.NotNil(t, invoice)
	require.Equal(t, InvoiceStatusRequested, invoice.Status)
	require.Equal(t, order.ID, invoice.OrderID)
	require.Equal(t, user.ID, invoice.UserID)
}

func TestRequestInvoiceRejectsDuplicateRequest(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, order := seedCompletedInvoiceOrder(t, ctx, client)
	svc := &PaymentService{entClient: client}

	_, err := svc.RequestInvoice(ctx, order.ID, user.ID, "上海某某科技有限公司", "91310113685471496R")
	require.NoError(t, err)

	_, err = svc.RequestInvoice(ctx, order.ID, user.ID, "上海某某科技有限公司", "91310113685471496R")
	require.Error(t, err)
	require.Equal(t, "INVOICE_ALREADY_EXISTS", infraerrors.Reason(err))
}

func TestMarkInvoiceIssuedStoresPDFAndAllowsDownload(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	t.Setenv("DATA_DIR", t.TempDir())

	user, order := seedCompletedInvoiceOrder(t, ctx, client)
	invoice, err := client.PaymentInvoice.Create().
		SetOrderID(order.ID).
		SetUserID(user.ID).
		SetTitleName("上海某某科技有限公司").
		SetTaxID("91310113685471496R").
		SetStatus(InvoiceStatusRequested).
		SetRequestedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client}
	updated, err := svc.MarkInvoiceIssued(ctx, invoice.ID, "invoice.pdf", []byte("%PDF-1.4\ninvoice\n"), "admin:1")
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusIssued, updated.Status)
	require.NotNil(t, updated.StorageKey)

	downloadInvoice, absolutePath, err := svc.PrepareUserInvoiceDownload(ctx, invoice.ID, user.ID)
	require.NoError(t, err)
	require.Equal(t, updated.ID, downloadInvoice.ID)

	content, err := os.ReadFile(absolutePath)
	require.NoError(t, err)
	require.Equal(t, "%PDF-1.4\ninvoice\n", string(content))
	require.FileExists(t, filepath.Clean(absolutePath))
}

func TestMarkInvoiceFailedStoresReason(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, order := seedCompletedInvoiceOrder(t, ctx, client)
	invoice, err := client.PaymentInvoice.Create().
		SetOrderID(order.ID).
		SetUserID(user.ID).
		SetTitleName("上海某某科技有限公司").
		SetTaxID("91310113685471496R").
		SetStatus(InvoiceStatusRequested).
		SetRequestedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client}
	updated, err := svc.MarkInvoiceFailed(ctx, invoice.ID, "税号校验失败", "admin:1")
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusFailed, updated.Status)
	require.NotNil(t, updated.FailedReason)
	require.Equal(t, "税号校验失败", *updated.FailedReason)
}

func seedCompletedInvoiceOrder(t *testing.T, ctx context.Context, client *dbent.Client) (*dbent.User, *dbent.PaymentOrder) {
	t.Helper()

	user, err := client.User.Create().
		SetEmail("invoice-user@example.com").
		SetPasswordHash("hash").
		SetUsername("invoice-user").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(99).
		SetPayAmount(99).
		SetFeeRate(0).
		SetRechargeCode("INVOICE-ORDER").
		SetOutTradeNo("sub2_invoice_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("invoice-trade-no").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetCompletedAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	return user, order
}
