//go:build integration

package service

import (
	"context"
	"database/sql"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPaymentInvoiceConcurrentRequestsClaimOrdersOnce(t *testing.T) {
	ctx := context.Background()
	client := newPaymentInvoicePostgresTestClient(t, ctx)
	svc := &PaymentService{entClient: client}
	user := createPaymentInvoiceTestUser(t, ctx, client, "invoice-concurrent@example.com")
	first := createPaymentInvoiceTestOrder(t, ctx, client, user, "concurrent-1", 60)
	second := createPaymentInvoiceTestOrder(t, ctx, client, user, "concurrent-2", 50)

	type result struct {
		invoice *dbent.PaymentInvoice
		err     error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for range 2 {
		go func() {
			<-start
			invoice, err := svc.RequestInvoice(ctx, []int64{first.ID, second.ID}, user.ID, "Concurrent Company", "91350211M000100Y43")
			results <- result{invoice: invoice, err: err}
		}()
	}
	close(start)

	successes := 0
	failures := 0
	for range 2 {
		result := <-results
		if result.err == nil {
			successes++
			require.NotNil(t, result.invoice)
			continue
		}
		failures++
		require.Contains(t, []string{"INVALID_ORDER_SELECTION", "INVOICE_ALREADY_EXISTS"}, infraerrors.Reason(result.err), result.err)
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, failures)
	require.Equal(t, 1, client.PaymentInvoice.Query().CountX(ctx))
}

func newPaymentInvoicePostgresTestClient(t *testing.T, ctx context.Context) *dbent.Client {
	t.Helper()
	container, err := tcpostgres.Run(
		ctx,
		"postgres:18.1-alpine3.23",
		tcpostgres.WithDatabase("sub2api_invoice_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, db.PingContext(ctx))

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	return client
}
