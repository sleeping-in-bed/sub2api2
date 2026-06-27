//go:build unit

package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestRequestInvoiceCreatesRecordForCompletedOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, orders := seedCompletedInvoiceOrders(t, ctx, client, 120, 30)
	svc := &PaymentService{entClient: client}

	invoice, err := svc.RequestInvoice(ctx, []int64{orders[0].ID, orders[1].ID}, user.ID, "上海某某科技有限公司", "91310113685471496R")
	require.NoError(t, err)
	require.NotNil(t, invoice)
	require.Equal(t, InvoiceStatusRequested, invoice.Status)
	require.Equal(t, user.ID, invoice.UserID)
	require.Len(t, invoice.Edges.Orders, 2)
	require.ElementsMatch(t, []int64{orders[0].ID, orders[1].ID}, []int64{invoice.Edges.Orders[0].ID, invoice.Edges.Orders[1].ID})
}

func TestRequestInvoiceRejectsDuplicateRequest(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, orders := seedCompletedInvoiceOrders(t, ctx, client, 120, 30)
	svc := &PaymentService{entClient: client}

	_, err := svc.RequestInvoice(ctx, []int64{orders[0].ID, orders[1].ID}, user.ID, "上海某某科技有限公司", "91310113685471496R")
	require.NoError(t, err)

	_, err = svc.RequestInvoice(ctx, []int64{orders[0].ID}, user.ID, "上海某某科技有限公司", "91310113685471496R")
	require.Error(t, err)
	require.Equal(t, "INVALID_ORDER_SELECTION", infraerrors.Reason(err))
}

func TestRequestInvoiceRejectsAmountBelowThreshold(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, orders := seedCompletedInvoiceOrders(t, ctx, client, 30, 40)
	svc := &PaymentService{entClient: client}

	_, err := svc.RequestInvoice(ctx, []int64{orders[0].ID, orders[1].ID}, user.ID, "上海某某科技有限公司", "91310113685471496R")
	require.Error(t, err)
	require.Equal(t, "INVOICE_AMOUNT_TOO_LOW", infraerrors.Reason(err))
}

func TestMarkInvoiceIssuedStoresPDFAndAllowsDownload(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	t.Setenv("DATA_DIR", t.TempDir())

	user, orders := seedCompletedInvoiceOrders(t, ctx, client, 120)
	invoice, err := client.PaymentInvoice.Create().
		SetUserID(user.ID).
		SetTitleName("上海某某科技有限公司").
		SetTaxID("91310113685471496R").
		SetStatus(InvoiceStatusRequested).
		SetRequestedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.PaymentOrder.UpdateOneID(orders[0].ID).SetInvoiceID(invoice.ID).Save(ctx)
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

	user, orders := seedCompletedInvoiceOrders(t, ctx, client, 120)
	invoice, err := client.PaymentInvoice.Create().
		SetUserID(user.ID).
		SetTitleName("上海某某科技有限公司").
		SetTaxID("91310113685471496R").
		SetStatus(InvoiceStatusRequested).
		SetRequestedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.PaymentOrder.UpdateOneID(orders[0].ID).SetInvoiceID(invoice.ID).Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client}
	updated, err := svc.MarkInvoiceFailed(ctx, invoice.ID, "税号校验失败", "admin:1")
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusFailed, updated.Status)
	require.NotNil(t, updated.FailedReason)
	require.Equal(t, "税号校验失败", *updated.FailedReason)
}

func TestMarkInvoiceFailedReleasesOrdersForReinvoicing(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, orders := seedCompletedInvoiceOrders(t, ctx, client, 120, 30)
	invoice, err := client.PaymentInvoice.Create().
		SetUserID(user.ID).
		SetTitleName("上海某某科技有限公司").
		SetTaxID("91310113685471496R").
		SetStatus(InvoiceStatusRequested).
		SetRequestedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	for _, order := range orders {
		_, err = client.PaymentOrder.UpdateOneID(order.ID).SetInvoiceID(invoice.ID).Save(ctx)
		require.NoError(t, err)
	}

	svc := &PaymentService{entClient: client}
	updated, err := svc.MarkInvoiceFailed(ctx, invoice.ID, "税号校验失败", "admin:1")
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusFailed, updated.Status)

	reloadedOrders, err := client.PaymentOrder.Query().
		Where(paymentorder.IDIn(orders[0].ID, orders[1].ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, reloadedOrders, 2)
	for _, order := range reloadedOrders {
		require.Nil(t, order.InvoiceID)
	}

	availableOrders, total, err := svc.ListInvoiceAvailableOrders(ctx, user.ID, OrderListParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, availableOrders, 2)
	require.ElementsMatch(t, []int64{orders[0].ID, orders[1].ID}, []int64{availableOrders[0].ID, availableOrders[1].ID})
}

func seedCompletedInvoiceOrders(t *testing.T, ctx context.Context, client *dbent.Client, payAmounts ...float64) (*dbent.User, []*dbent.PaymentOrder) {
	t.Helper()

	user, err := client.User.Create().
		SetEmail("invoice-user@example.com").
		SetPasswordHash("hash").
		SetUsername("invoice-user").
		Save(ctx)
	require.NoError(t, err)

	orders := make([]*dbent.PaymentOrder, 0, len(payAmounts))
	for idx, payAmount := range payAmounts {
		order, createErr := client.PaymentOrder.Create().
			SetUserID(user.ID).
			SetUserEmail(user.Email).
			SetUserName(user.Username).
			SetAmount(payAmount).
			SetPayAmount(payAmount).
			SetFeeRate(0).
			SetRechargeCode("INVOICE-ORDER").
			SetOutTradeNo(fmt.Sprintf("sub2_invoice_order_%d", idx+1)).
			SetPaymentType(payment.TypeAlipay).
			SetPaymentTradeNo(fmt.Sprintf("invoice-trade-no-%d", idx+1)).
			SetOrderType(payment.OrderTypeBalance).
			SetStatus(OrderStatusCompleted).
			SetExpiresAt(time.Now().Add(time.Hour)).
			SetPaidAt(time.Now()).
			SetCompletedAt(time.Now()).
			SetClientIP("127.0.0.1").
			SetSrcHost("api.example.com").
			Save(ctx)
		require.NoError(t, createErr)
		orders = append(orders, order)
	}

	return user, orders
}
