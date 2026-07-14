package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentinvoice"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/pkg/datadir"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	InvoiceStatusRequested = "REQUESTED"
	InvoiceStatusIssued    = "ISSUED"
	InvoiceStatusFailed    = "FAILED"

	paymentInvoiceStorageProviderLocal = "local"
	paymentInvoiceMaxUploadBytes       = 10 << 20
	paymentInvoiceMinPayAmount float64 = 100
)

var paymentInvoiceTaxIDPattern = regexp.MustCompile(`^[0-9A-Z]{15,20}$`)

type PaymentInvoiceListParams struct {
	Page     int
	PageSize int
	Status   string
	Keyword  string
}

func (s *PaymentService) RequestInvoice(ctx context.Context, orderIDs []int64, userID int64, titleName, taxID string) (*dbent.PaymentInvoice, error) {
	if s == nil || s.entClient == nil {
		return nil, infraerrors.InternalServer("SERVICE_UNAVAILABLE", "payment service is unavailable")
	}
	titleName, taxID, err := normalizePaymentInvoiceInput(titleName, taxID)
	if err != nil {
		return nil, err
	}
	orderIDs = normalizeInvoiceOrderIDs(orderIDs)
	if len(orderIDs) == 0 {
		return nil, infraerrors.BadRequest("INVOICE_ORDERS_EMPTY", "invoice orders are required")
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin invoice request transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	orders, err := tx.PaymentOrder.Query().Where(
		paymentorder.IDIn(orderIDs...),
		paymentorder.UserIDEQ(userID),
		paymentorder.StatusEQ(OrderStatusCompleted),
		paymentorder.InvoiceIDIsNil(),
	).All(txCtx)
	if err != nil {
		return nil, fmt.Errorf("query invoice orders: %w", err)
	}
	if len(orders) != len(orderIDs) {
		return nil, infraerrors.BadRequest("INVALID_ORDER_SELECTION", "invoice orders must be completed, belong to the user, and not be invoiced")
	}
	totalPayAmount := sumPaymentOrdersPayAmount(orders)
	if totalPayAmount < paymentInvoiceMinPayAmount {
		return nil, infraerrors.BadRequest("INVOICE_AMOUNT_TOO_LOW", fmt.Sprintf("invoice pay amount must be at least %.2f", paymentInvoiceMinPayAmount))
	}

	now := time.Now()
	invoice, err := tx.PaymentInvoice.Create().
		SetUserID(userID).
		SetTitleName(titleName).
		SetTaxID(taxID).
		SetStatus(InvoiceStatusRequested).
		SetRequestedAt(now).
		Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("create payment invoice: %w", err)
	}
	updated, err := tx.PaymentOrder.Update().Where(
		paymentorder.IDIn(orderIDs...),
		paymentorder.UserIDEQ(userID),
		paymentorder.StatusEQ(OrderStatusCompleted),
		paymentorder.InvoiceIDIsNil(),
	).SetInvoiceID(invoice.ID).Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("attach invoice to orders: %w", err)
	}
	if updated != len(orderIDs) {
		return nil, infraerrors.Conflict("INVOICE_ALREADY_EXISTS", "some selected orders were invoiced concurrently")
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice request transaction: %w", err)
	}

	invoice, err = s.GetInvoiceByID(ctx, invoice.ID)
	if err != nil {
		return nil, err
	}
	for _, order := range invoice.Edges.Orders {
		s.writeAuditLog(ctx, order.ID, "INVOICE_REQUESTED", fmt.Sprintf("user:%d", userID), map[string]any{
			"invoiceID": invoice.ID, "invoiceOrderIDs": paymentInvoiceOrderIDs(invoice.Edges.Orders),
			"orderCount": len(invoice.Edges.Orders), "totalPayAmount": totalPayAmount,
		})
	}
	return invoice, nil
}

func (s *PaymentService) GetInvoiceByID(ctx context.Context, invoiceID int64) (*dbent.PaymentInvoice, error) {
	if s == nil || s.entClient == nil {
		return nil, infraerrors.InternalServer("SERVICE_UNAVAILABLE", "payment service is unavailable")
	}
	invoice, err := s.entClient.PaymentInvoice.Query().Where(paymentinvoice.IDEQ(invoiceID)).WithOrders().Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("NOT_FOUND", "invoice not found")
		}
		return nil, fmt.Errorf("get invoice: %w", err)
	}
	return invoice, nil
}

func (s *PaymentService) ListInvoices(ctx context.Context, p PaymentInvoiceListParams) ([]*dbent.PaymentInvoice, int, error) {
	q := s.entClient.PaymentInvoice.Query().WithOrders()
	if p.Status != "" {
		q = q.Where(paymentinvoice.StatusEQ(p.Status))
	}
	if keyword := strings.TrimSpace(p.Keyword); keyword != "" {
		q = q.Where(paymentinvoice.Or(
			paymentinvoice.TitleNameContainsFold(keyword),
			paymentinvoice.TaxIDContainsFold(keyword),
			paymentinvoice.HasOrdersWith(paymentorder.Or(
				paymentorder.OutTradeNoContainsFold(keyword),
				paymentorder.UserEmailContainsFold(keyword),
				paymentorder.UserNameContainsFold(keyword),
			)),
		))
	}
	return listPaymentInvoices(ctx, q, p)
}

func (s *PaymentService) ListUserInvoices(ctx context.Context, userID int64, p PaymentInvoiceListParams) ([]*dbent.PaymentInvoice, int, error) {
	q := s.entClient.PaymentInvoice.Query().Where(paymentinvoice.UserIDEQ(userID)).WithOrders()
	if p.Status != "" {
		q = q.Where(paymentinvoice.StatusEQ(p.Status))
	}
	return listPaymentInvoices(ctx, q, p)
}

func listPaymentInvoices(ctx context.Context, q *dbent.PaymentInvoiceQuery, p PaymentInvoiceListParams) ([]*dbent.PaymentInvoice, int, error) {
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count payment invoices: %w", err)
	}
	pageSize, page := applyPagination(p.PageSize, p.Page)
	invoices, err := q.Order(dbent.Desc(paymentinvoice.FieldRequestedAt)).Limit(pageSize).Offset((page - 1) * pageSize).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("query payment invoices: %w", err)
	}
	return invoices, total, nil
}

func (s *PaymentService) GetUserInvoiceByID(ctx context.Context, invoiceID, userID int64) (*dbent.PaymentInvoice, error) {
	invoice, err := s.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice.UserID != userID {
		return nil, infraerrors.Forbidden("FORBIDDEN", "no permission for this invoice")
	}
	return invoice, nil
}

func (s *PaymentService) ListInvoiceAvailableOrders(ctx context.Context, userID int64, p OrderListParams) ([]*dbent.PaymentOrder, int, error) {
	q := s.entClient.PaymentOrder.Query().Where(
		paymentorder.UserIDEQ(userID),
		paymentorder.StatusEQ(OrderStatusCompleted),
		paymentorder.InvoiceIDIsNil(),
	)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count invoice available orders: %w", err)
	}
	pageSize, page := applyPagination(p.PageSize, p.Page)
	orders, err := q.Order(dbent.Desc(paymentorder.FieldCompletedAt), dbent.Desc(paymentorder.FieldCreatedAt)).Limit(pageSize).Offset((page - 1) * pageSize).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("query invoice available orders: %w", err)
	}
	return orders, total, nil
}

func (s *PaymentService) GetInvoiceAvailableSummary(ctx context.Context, userID int64) (float64, int, error) {
	orders, err := s.entClient.PaymentOrder.Query().Where(
		paymentorder.UserIDEQ(userID),
		paymentorder.StatusEQ(OrderStatusCompleted),
		paymentorder.InvoiceIDIsNil(),
	).All(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("query invoice available summary orders: %w", err)
	}
	return sumPaymentOrdersPayAmount(orders), len(orders), nil
}

func (s *PaymentService) MarkInvoiceIssued(ctx context.Context, invoiceID int64, originalFileName string, content []byte, operator string) (*dbent.PaymentInvoice, error) {
	if len(content) == 0 {
		return nil, infraerrors.BadRequest("INVALID_FILE", "invoice file is required")
	}
	if len(content) > paymentInvoiceMaxUploadBytes {
		return nil, infraerrors.BadRequest("FILE_TOO_LARGE", "invoice file is too large")
	}
	if contentType := strings.TrimSpace(http.DetectContentType(content)); contentType != "application/pdf" {
		return nil, infraerrors.BadRequest("INVALID_FILE_TYPE", "invoice file must be a PDF")
	}
	invoice, err := s.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice.Status != InvoiceStatusRequested {
		return nil, infraerrors.BadRequest("INVALID_STATUS", "invoice status does not allow upload")
	}

	fileName := sanitizePaymentInvoiceFileName(originalFileName, invoice.ID)
	storageKey := filepath.Join("invoices", strconv.FormatInt(invoice.ID, 10), fileName)
	absolutePath, err := paymentInvoiceAbsolutePath(storageKey)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return nil, fmt.Errorf("create invoice directory: %w", err)
	}
	tempPath := absolutePath + ".tmp"
	if err := os.WriteFile(tempPath, content, 0o644); err != nil {
		return nil, fmt.Errorf("write invoice file: %w", err)
	}
	if err := os.Rename(tempPath, absolutePath); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("finalize invoice file: %w", err)
	}

	now := time.Now()
	sum := sha256.Sum256(content)
	updated, err := s.entClient.PaymentInvoice.UpdateOneID(invoice.ID).
		SetStatus(InvoiceStatusIssued).
		SetIssuedAt(now).
		ClearFailedAt().ClearFailedReason().
		SetStorageProvider(paymentInvoiceStorageProviderLocal).
		SetStorageKey(storageKey).
		SetFileName(fileName).
		SetContentType("application/pdf").
		SetByteSize(int64(len(content))).
		SetSha256(hex.EncodeToString(sum[:])).
		Save(ctx)
	if err != nil {
		_ = os.Remove(absolutePath)
		return nil, fmt.Errorf("update invoice as issued: %w", err)
	}
	for _, order := range invoice.Edges.Orders {
		s.writeAuditLog(ctx, order.ID, "INVOICE_ISSUED", operator, map[string]any{"invoiceID": invoice.ID, "fileName": fileName, "byteSize": len(content)})
	}
	return updated, nil
}

func (s *PaymentService) MarkInvoiceFailed(ctx context.Context, invoiceID int64, reason, operator string) (*dbent.PaymentInvoice, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, infraerrors.BadRequest("INVALID_REASON", "invoice failure reason is required")
	}
	invoice, err := s.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice.Status != InvoiceStatusRequested && invoice.Status != InvoiceStatusFailed {
		return nil, infraerrors.BadRequest("INVALID_STATUS", "invoice status does not allow failure updates")
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin mark invoice failed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	if _, err := tx.PaymentOrder.Update().Where(paymentorder.InvoiceIDEQ(invoice.ID)).ClearInvoiceID().Save(txCtx); err != nil {
		return nil, fmt.Errorf("detach failed invoice orders: %w", err)
	}
	now := time.Now()
	updated, err := tx.PaymentInvoice.UpdateOneID(invoice.ID).
		SetStatus(InvoiceStatusFailed).
		SetFailedAt(now).
		SetFailedReason(reason).
		ClearIssuedAt().
		Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("update invoice as failed: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit mark invoice failed transaction: %w", err)
	}
	for _, order := range invoice.Edges.Orders {
		s.writeAuditLog(ctx, order.ID, "INVOICE_FAILED", operator, map[string]any{"invoiceID": invoice.ID, "reason": reason})
	}
	return updated, nil
}

func (s *PaymentService) PrepareUserInvoiceDownload(ctx context.Context, invoiceID, userID int64) (*dbent.PaymentInvoice, string, error) {
	invoice, err := s.GetUserInvoiceByID(ctx, invoiceID, userID)
	if err != nil {
		return nil, "", err
	}
	path, err := preparePaymentInvoiceDownloadPath(invoice)
	return invoice, path, err
}

func (s *PaymentService) PrepareAdminInvoiceDownload(ctx context.Context, invoiceID int64) (*dbent.PaymentInvoice, string, error) {
	invoice, err := s.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, "", err
	}
	path, err := preparePaymentInvoiceDownloadPath(invoice)
	return invoice, path, err
}

func normalizeInvoiceOrderIDs(orderIDs []int64) []int64 {
	seen := make(map[int64]struct{}, len(orderIDs))
	normalized := make([]int64, 0, len(orderIDs))
	for _, orderID := range orderIDs {
		if orderID <= 0 {
			continue
		}
		if _, ok := seen[orderID]; ok {
			continue
		}
		seen[orderID] = struct{}{}
		normalized = append(normalized, orderID)
	}
	return normalized
}

func sumPaymentOrdersPayAmount(orders []*dbent.PaymentOrder) float64 {
	total := 0.0
	for _, order := range orders {
		if order != nil {
			total += order.PayAmount
		}
	}
	return total
}

func paymentInvoiceOrderIDs(orders []*dbent.PaymentOrder) []int64 {
	ids := make([]int64, 0, len(orders))
	for _, order := range orders {
		if order != nil {
			ids = append(ids, order.ID)
		}
	}
	return ids
}

func normalizePaymentInvoiceInput(titleName, taxID string) (string, string, error) {
	titleName = strings.TrimSpace(titleName)
	if titleName == "" || len([]rune(titleName)) > 200 {
		return "", "", infraerrors.BadRequest("INVALID_TITLE_NAME", "invoice title is required and must not exceed 200 characters")
	}
	taxID = strings.ToUpper(strings.TrimSpace(taxID))
	if !paymentInvoiceTaxIDPattern.MatchString(taxID) {
		return "", "", infraerrors.BadRequest("INVALID_TAX_ID", "tax id format is invalid")
	}
	return titleName, taxID, nil
}

func sanitizePaymentInvoiceFileName(original string, invoiceID int64) string {
	name := strings.TrimSpace(filepath.Base(original))
	base := strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
	if base == "" {
		base = fmt.Sprintf("invoice-%d", invoiceID)
	}
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, base)
	return base + ".pdf"
}

func paymentInvoiceAbsolutePath(storageKey string) (string, error) {
	baseDir, err := filepath.Abs(datadir.Resolve())
	if err != nil {
		return "", fmt.Errorf("resolve invoice data directory: %w", err)
	}
	storageKey = filepath.Clean(strings.TrimSpace(storageKey))
	if storageKey == "." || filepath.IsAbs(storageKey) {
		return "", infraerrors.BadRequest("INVALID_STORAGE_KEY", "invoice file path is invalid")
	}
	absolutePath := filepath.Join(baseDir, storageKey)
	relativePath, err := filepath.Rel(baseDir, absolutePath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", infraerrors.BadRequest("INVALID_STORAGE_KEY", "invoice file path is invalid")
	}
	return absolutePath, nil
}

func preparePaymentInvoiceDownloadPath(invoice *dbent.PaymentInvoice) (string, error) {
	if invoice == nil {
		return "", infraerrors.NotFound("NOT_FOUND", "invoice not found")
	}
	if invoice.Status != InvoiceStatusIssued || invoice.StorageKey == nil || strings.TrimSpace(*invoice.StorageKey) == "" {
		return "", infraerrors.BadRequest("INVOICE_NOT_READY", "invoice is not ready for download")
	}
	absolutePath, err := paymentInvoiceAbsolutePath(*invoice.StorageKey)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolutePath)
	if err != nil || info.IsDir() {
		if err == nil || os.IsNotExist(err) {
			return "", infraerrors.NotFound("FILE_NOT_FOUND", "invoice file is missing")
		}
		return "", fmt.Errorf("stat invoice file: %w", err)
	}
	return absolutePath, nil
}
