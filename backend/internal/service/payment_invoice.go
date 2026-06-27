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

	normalizedTitle, normalizedTaxID, err := normalizePaymentInvoiceInput(titleName, taxID)
	if err != nil {
		return nil, err
	}

	normalizedOrderIDs := normalizeInvoiceOrderIDs(orderIDs)
	if len(normalizedOrderIDs) == 0 {
		return nil, infraerrors.BadRequest("INVOICE_ORDERS_EMPTY", "invoice orders are required")
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin invoice request tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	orders, err := tx.PaymentOrder.Query().
		Where(
			paymentorder.IDIn(normalizedOrderIDs...),
			paymentorder.UserIDEQ(userID),
			paymentorder.StatusEQ(OrderStatusCompleted),
			paymentorder.InvoiceIDIsNil(),
		).
		All(txCtx)
	if err != nil {
		return nil, fmt.Errorf("query invoice orders: %w", err)
	}
	if len(orders) != len(normalizedOrderIDs) {
		return nil, infraerrors.BadRequest("INVALID_ORDER_SELECTION", "invoice orders must be completed, belong to the user, and not be invoiced")
	}

	totalPayAmount := sumPaymentOrdersPayAmount(orders)
	if totalPayAmount < paymentInvoiceMinPayAmount {
		return nil, infraerrors.BadRequest("INVOICE_AMOUNT_TOO_LOW", fmt.Sprintf("invoice pay amount must be at least %.2f", paymentInvoiceMinPayAmount))
	}

	now := time.Now()
	invoice, err := tx.PaymentInvoice.Create().
		SetUserID(userID).
		SetTitleName(normalizedTitle).
		SetTaxID(normalizedTaxID).
		SetStatus(InvoiceStatusRequested).
		SetRequestedAt(now).
		Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("create payment invoice: %w", err)
	}

	updatedCount, err := tx.PaymentOrder.Update().
		Where(
			paymentorder.IDIn(normalizedOrderIDs...),
			paymentorder.UserIDEQ(userID),
			paymentorder.StatusEQ(OrderStatusCompleted),
			paymentorder.InvoiceIDIsNil(),
		).
		SetInvoiceID(invoice.ID).
		Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("attach invoice to orders: %w", err)
	}
	if updatedCount != len(normalizedOrderIDs) {
		return nil, infraerrors.Conflict("INVOICE_ALREADY_EXISTS", "some selected orders were invoiced concurrently")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice request tx: %w", err)
	}

	invoice, err = s.GetInvoiceByID(ctx, invoice.ID)
	if err != nil {
		return nil, err
	}

	orderIDsForAudit := paymentInvoiceOrderIDs(invoice.Edges.Orders)
	for _, order := range invoice.Edges.Orders {
		s.writeAuditLog(ctx, order.ID, "INVOICE_REQUESTED", fmt.Sprintf("user:%d", userID), map[string]any{
			"invoiceID":      invoice.ID,
			"titleName":      invoice.TitleName,
			"taxID":          invoice.TaxID,
			"requestedAt":    now.Format(time.RFC3339),
			"invoiceOrderIDs": orderIDsForAudit,
			"orderCount":     len(orderIDsForAudit),
			"totalPayAmount": totalPayAmount,
		})
	}

	return invoice, nil
}

func (s *PaymentService) GetInvoiceByID(ctx context.Context, invoiceID int64) (*dbent.PaymentInvoice, error) {
	invoice, err := s.entClient.PaymentInvoice.Query().
		Where(paymentinvoice.IDEQ(invoiceID)).
		WithOrders().
		Only(ctx)
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
			paymentinvoice.HasOrdersWith(
				paymentorder.Or(
					paymentorder.OutTradeNoContainsFold(keyword),
					paymentorder.UserEmailContainsFold(keyword),
					paymentorder.UserNameContainsFold(keyword),
				),
			),
		))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count payment invoices: %w", err)
	}

	pageSize, page := applyPagination(p.PageSize, p.Page)
	invoices, err := q.
		Order(dbent.Desc(paymentinvoice.FieldRequestedAt)).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("query payment invoices: %w", err)
	}

	return invoices, total, nil
}

func (s *PaymentService) ListUserInvoices(ctx context.Context, userID int64, p PaymentInvoiceListParams) ([]*dbent.PaymentInvoice, int, error) {
	q := s.entClient.PaymentInvoice.Query().
		Where(paymentinvoice.UserIDEQ(userID)).
		WithOrders()
	if p.Status != "" {
		q = q.Where(paymentinvoice.StatusEQ(p.Status))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count user payment invoices: %w", err)
	}

	pageSize, page := applyPagination(p.PageSize, p.Page)
	invoices, err := q.
		Order(dbent.Desc(paymentinvoice.FieldRequestedAt)).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("query user payment invoices: %w", err)
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
	q := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.UserIDEQ(userID),
			paymentorder.StatusEQ(OrderStatusCompleted),
			paymentorder.InvoiceIDIsNil(),
		)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count invoice available orders: %w", err)
	}

	pageSize, page := applyPagination(p.PageSize, p.Page)
	orders, err := q.
		Order(dbent.Desc(paymentorder.FieldCompletedAt), dbent.Desc(paymentorder.FieldCreatedAt)).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("query invoice available orders: %w", err)
	}

	return orders, total, nil
}

func (s *PaymentService) GetInvoiceAvailableSummary(ctx context.Context, userID int64) (float64, int, error) {
	orders, err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.UserIDEQ(userID),
			paymentorder.StatusEQ(OrderStatusCompleted),
			paymentorder.InvoiceIDIsNil(),
		).
		All(ctx)
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

	contentType := strings.TrimSpace(http.DetectContentType(content))
	if contentType != "application/pdf" {
		return nil, infraerrors.BadRequest("INVALID_FILE_TYPE", "invoice file must be a PDF")
	}

	invoice, err := s.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice.Status != InvoiceStatusRequested && invoice.Status != InvoiceStatusFailed {
		return nil, infraerrors.BadRequest("INVALID_STATUS", "invoice status does not allow upload")
	}

	fileName := sanitizePaymentInvoiceFileName(originalFileName, invoiceID)
	relativePath := filepath.Join("invoices", strconv.FormatInt(invoice.ID, 10), fileName)
	absolutePath, err := paymentInvoiceAbsolutePath(relativePath)
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
		ClearFailedAt().
		ClearFailedReason().
		SetStorageProvider(paymentInvoiceStorageProviderLocal).
		SetStorageKey(relativePath).
		SetFileName(fileName).
		SetContentType(contentType).
		SetByteSize(int64(len(content))).
		SetSha256(hex.EncodeToString(sum[:])).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update invoice as issued: %w", err)
	}

	orderIDs := paymentInvoiceOrderIDs(invoice.Edges.Orders)
	for _, order := range invoice.Edges.Orders {
		s.writeAuditLog(ctx, order.ID, "INVOICE_ISSUED", operator, map[string]any{
			"invoiceID":      invoice.ID,
			"fileName":       updated.FileName,
			"contentType":    updated.ContentType,
			"byteSize":       updated.ByteSize,
			"storageKey":     updated.StorageKey,
			"issuedAt":       now.Format(time.RFC3339),
			"invoiceOrderIDs": orderIDs,
		})
	}

	return updated, nil
}

func (s *PaymentService) MarkInvoiceFailed(ctx context.Context, invoiceID int64, reason, operator string) (*dbent.PaymentInvoice, error) {
	normalizedReason := strings.TrimSpace(reason)
	if normalizedReason == "" {
		return nil, infraerrors.BadRequest("INVALID_REASON", "invoice failure reason is required")
	}

	invoice, err := s.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice.Status != InvoiceStatusRequested && invoice.Status != InvoiceStatusFailed {
		return nil, infraerrors.BadRequest("INVALID_STATUS", "invoice status does not allow failure updates")
	}

	now := time.Now()
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin mark invoice failed tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.PaymentOrder.Update().
		Where(paymentorder.InvoiceIDEQ(invoice.ID)).
		ClearInvoiceID().
		Save(ctx); err != nil {
		return nil, fmt.Errorf("detach failed invoice orders: %w", err)
	}

	updated, err := tx.PaymentInvoice.UpdateOneID(invoice.ID).
		SetStatus(InvoiceStatusFailed).
		SetFailedAt(now).
		SetFailedReason(normalizedReason).
		ClearIssuedAt().
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update invoice as failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit mark invoice failed tx: %w", err)
	}

	orderIDs := paymentInvoiceOrderIDs(invoice.Edges.Orders)
	for _, order := range invoice.Edges.Orders {
		s.writeAuditLog(ctx, order.ID, "INVOICE_FAILED", operator, map[string]any{
			"invoiceID":      invoice.ID,
			"reason":         normalizedReason,
			"failedAt":       now.Format(time.RFC3339),
			"invoiceOrderIDs": orderIDs,
		})
	}

	return updated, nil
}

func (s *PaymentService) PrepareUserInvoiceDownload(ctx context.Context, invoiceID, userID int64) (*dbent.PaymentInvoice, string, error) {
	invoice, err := s.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, "", err
	}
	if invoice.UserID != userID {
		return nil, "", infraerrors.Forbidden("FORBIDDEN", "no permission for this invoice")
	}
	absolutePath, err := preparePaymentInvoiceDownloadPath(invoice)
	if err != nil {
		return nil, "", err
	}
	return invoice, absolutePath, nil
}

func normalizeInvoiceOrderIDs(orderIDs []int64) []int64 {
	if len(orderIDs) == 0 {
		return nil
	}
	unique := make(map[int64]struct{}, len(orderIDs))
	normalized := make([]int64, 0, len(orderIDs))
	for _, orderID := range orderIDs {
		if orderID <= 0 {
			continue
		}
		if _, exists := unique[orderID]; exists {
			continue
		}
		unique[orderID] = struct{}{}
		normalized = append(normalized, orderID)
	}
	return normalized
}

func sumPaymentOrdersPayAmount(orders []*dbent.PaymentOrder) float64 {
	total := 0.0
	for _, order := range orders {
		if order == nil {
			continue
		}
		total += order.PayAmount
	}
	return total
}

func paymentInvoiceOrderIDs(orders []*dbent.PaymentOrder) []int64 {
	orderIDs := make([]int64, 0, len(orders))
	for _, order := range orders {
		if order == nil {
			continue
		}
		orderIDs = append(orderIDs, order.ID)
	}
	return orderIDs
}

func (s *PaymentService) PrepareAdminInvoiceDownload(ctx context.Context, invoiceID int64) (*dbent.PaymentInvoice, string, error) {
	invoice, err := s.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, "", err
	}
	absolutePath, err := preparePaymentInvoiceDownloadPath(invoice)
	if err != nil {
		return nil, "", err
	}
	return invoice, absolutePath, nil
}

func normalizePaymentInvoiceInput(titleName, taxID string) (string, string, error) {
	normalizedTitle := strings.TrimSpace(titleName)
	if normalizedTitle == "" {
		return "", "", infraerrors.BadRequest("INVALID_TITLE_NAME", "invoice title is required")
	}
	if len([]rune(normalizedTitle)) > 200 {
		return "", "", infraerrors.BadRequest("INVALID_TITLE_NAME", "invoice title is too long")
	}

	normalizedTaxID := strings.ToUpper(strings.TrimSpace(taxID))
	if normalizedTaxID == "" {
		return "", "", infraerrors.BadRequest("INVALID_TAX_ID", "tax id is required")
	}
	if !paymentInvoiceTaxIDPattern.MatchString(normalizedTaxID) {
		return "", "", infraerrors.BadRequest("INVALID_TAX_ID", "tax id format is invalid")
	}

	return normalizedTitle, normalizedTaxID, nil
}

func sanitizePaymentInvoiceFileName(original string, invoiceID int64) string {
	name := strings.TrimSpace(filepath.Base(original))
	if name == "" {
		name = fmt.Sprintf("invoice-%d.pdf", invoiceID)
	}
	ext := strings.ToLower(filepath.Ext(name))
	base := strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
	if base == "" {
		base = fmt.Sprintf("invoice-%d", invoiceID)
	}
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, base)
	if ext != ".pdf" {
		ext = ".pdf"
	}
	return base + ext
}

func paymentInvoiceAbsolutePath(storageKey string) (string, error) {
	baseDir := filepath.Clean(paymentInvoiceDataDir())
	relativePath := filepath.Clean(strings.TrimSpace(storageKey))
	if relativePath == "" {
		return "", infraerrors.BadRequest("INVALID_STORAGE_KEY", "invoice file is missing")
	}
	if filepath.IsAbs(relativePath) {
		return "", infraerrors.BadRequest("INVALID_STORAGE_KEY", "invoice file path is invalid")
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", infraerrors.BadRequest("INVALID_STORAGE_KEY", "invoice file path is invalid")
	}
	return filepath.Join(baseDir, relativePath), nil
}

func paymentInvoiceDataDir() string {
	if dir := strings.TrimSpace(os.Getenv("DATA_DIR")); dir != "" {
		return dir
	}
	dockerDataDir := "/app/data"
	if info, err := os.Stat(dockerDataDir); err == nil && info.IsDir() {
		testFile := filepath.Join(dockerDataDir, ".invoice_write_test")
		if f, err := os.Create(testFile); err == nil {
			_ = f.Close()
			_ = os.Remove(testFile)
			return dockerDataDir
		}
	}
	return "."
}

func preparePaymentInvoiceDownloadPath(invoice *dbent.PaymentInvoice) (string, error) {
	if invoice == nil {
		return "", infraerrors.NotFound("NOT_FOUND", "invoice not found")
	}
	if invoice.Status != InvoiceStatusIssued {
		return "", infraerrors.BadRequest("INVOICE_NOT_READY", "invoice is not ready for download")
	}
	if strings.TrimSpace(psStringValue(invoice.StorageKey)) == "" {
		return "", infraerrors.NotFound("FILE_NOT_FOUND", "invoice file is missing")
	}

	absolutePath, err := paymentInvoiceAbsolutePath(psStringValue(invoice.StorageKey))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolutePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", infraerrors.NotFound("FILE_NOT_FOUND", "invoice file is missing")
		}
		return "", fmt.Errorf("stat invoice file: %w", err)
	}
	if info.IsDir() {
		return "", infraerrors.NotFound("FILE_NOT_FOUND", "invoice file is missing")
	}
	return absolutePath, nil
}
