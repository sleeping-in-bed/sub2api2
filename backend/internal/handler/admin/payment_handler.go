package admin

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles admin payment management.
type PaymentHandler struct {
	paymentService *service.PaymentService
	configService  *service.PaymentConfigService
}

// NewPaymentHandler creates a new admin PaymentHandler.
func NewPaymentHandler(paymentService *service.PaymentService, configService *service.PaymentConfigService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		configService:  configService,
	}
}

// --- Dashboard ---

// GetDashboard returns payment dashboard statistics.
// GET /api/v1/admin/payment/dashboard
func (h *PaymentHandler) GetDashboard(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			days = v
		}
	}
	stats, err := h.paymentService.GetDashboardStats(c.Request.Context(), days)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

// --- Orders ---

// ListOrders returns a paginated list of all payment orders.
// GET /api/v1/admin/payment/orders
func (h *PaymentHandler) ListOrders(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	var userID int64
	if uid := c.Query("user_id"); uid != "" {
		if v, err := strconv.ParseInt(uid, 10, 64); err == nil {
			userID = v
		}
	}
	orders, total, err := h.paymentService.AdminListOrders(c.Request.Context(), userID, service.OrderListParams{
		Page:        page,
		PageSize:    pageSize,
		Status:      c.Query("status"),
		OrderType:   c.Query("order_type"),
		PaymentType: c.Query("payment_type"),
		Keyword:     c.Query("keyword"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, sanitizeAdminPaymentOrdersForResponse(orders), int64(total), page, pageSize)
}

// GetOrderDetail returns detailed information about a single order.
// GET /api/v1/admin/payment/orders/:id
func (h *PaymentHandler) GetOrderDetail(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	order, err := h.paymentService.GetOrderByID(c.Request.Context(), orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	auditLogs, _ := h.paymentService.GetOrderAuditLogs(c.Request.Context(), orderID)
	response.Success(c, gin.H{"order": sanitizeAdminPaymentOrderForResponse(order), "auditLogs": auditLogs})
}

// CancelOrder cancels a pending order (admin).
// POST /api/v1/admin/payment/orders/:id/cancel
func (h *PaymentHandler) CancelOrder(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	msg, err := h.paymentService.AdminCancelOrder(c.Request.Context(), orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": msg})
}

// RetryFulfillment retries fulfillment for a paid order.
// POST /api/v1/admin/payment/orders/:id/retry
func (h *PaymentHandler) RetryFulfillment(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.paymentService.RetryFulfillment(c.Request.Context(), orderID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "fulfillment retried"})
}

type AdminPaymentOrderResult struct {
	ID                  int64                       `json:"id"`
	UserID              int64                       `json:"user_id"`
	UserEmail           string                      `json:"user_email"`
	UserName            string                      `json:"user_name"`
	UserNotes           *string                     `json:"user_notes,omitempty"`
	Amount              float64                     `json:"amount"`
	PayAmount           float64                     `json:"pay_amount"`
	FeeRate             float64                     `json:"fee_rate"`
	Currency            string                      `json:"currency"`
	PaymentType         string                      `json:"payment_type"`
	OutTradeNo          string                      `json:"out_trade_no"`
	Status              string                      `json:"status"`
	OrderType           string                      `json:"order_type"`
	CreatedAt           time.Time                   `json:"created_at"`
	ExpiresAt           time.Time                   `json:"expires_at"`
	PaidAt              *time.Time                  `json:"paid_at,omitempty"`
	CompletedAt         *time.Time                  `json:"completed_at,omitempty"`
	FailedAt            *time.Time                  `json:"failed_at,omitempty"`
	FailedReason        *string                     `json:"failed_reason,omitempty"`
	RefundAmount        float64                     `json:"refund_amount"`
	RefundReason        *string                     `json:"refund_reason,omitempty"`
	RefundRequestedAt   *time.Time                  `json:"refund_requested_at,omitempty"`
	RefundRequestedBy   *string                     `json:"refund_requested_by,omitempty"`
	RefundRequestReason *string                     `json:"refund_request_reason,omitempty"`
	PlanID              *int64                      `json:"plan_id,omitempty"`
	ProviderInstanceID  *string                     `json:"provider_instance_id,omitempty"`
	Invoice             *AdminPaymentInvoiceSummary `json:"invoice,omitempty"`
}

type AdminPaymentInvoiceSummary struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"user_id"`
	TitleName      string     `json:"title_name"`
	TaxID          string     `json:"tax_id"`
	Status         string     `json:"status"`
	RequestedAt    time.Time  `json:"requested_at"`
	IssuedAt       *time.Time `json:"issued_at,omitempty"`
	FailedAt       *time.Time `json:"failed_at,omitempty"`
	FailedReason   *string    `json:"failed_reason,omitempty"`
	FileName       *string    `json:"file_name,omitempty"`
	ContentType    *string    `json:"content_type,omitempty"`
	ByteSize       int64      `json:"byte_size"`
	OrderCount     int        `json:"order_count"`
	TotalAmount    float64    `json:"total_amount"`
	TotalPayAmount float64    `json:"total_pay_amount"`
}

type AdminPaymentInvoiceResult struct {
	AdminPaymentInvoiceSummary
	Orders []AdminPaymentInvoiceOrder `json:"orders,omitempty"`
}

type AdminPaymentInvoiceOrder struct {
	ID          int64      `json:"id"`
	OrderUUID   string     `json:"order_uuid"`
	UserID      int64      `json:"user_id"`
	UserEmail   string     `json:"user_email"`
	UserName    string     `json:"user_name"`
	UserNotes   *string    `json:"user_notes,omitempty"`
	OutTradeNo  string     `json:"out_trade_no"`
	Status      string     `json:"status"`
	OrderType   string     `json:"order_type"`
	PaymentType string     `json:"payment_type"`
	Amount      float64    `json:"amount"`
	PayAmount   float64    `json:"pay_amount"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func sanitizeAdminPaymentOrdersForResponse(orders []*dbent.PaymentOrder) []AdminPaymentOrderResult {
	out := make([]AdminPaymentOrderResult, 0, len(orders))
	for _, order := range orders {
		if item := sanitizeAdminPaymentOrderForResponse(order); item != nil {
			out = append(out, *item)
		}
	}
	return out
}

func sanitizeAdminPaymentOrderForResponse(order *dbent.PaymentOrder) *AdminPaymentOrderResult {
	if order == nil {
		return nil
	}
	return &AdminPaymentOrderResult{
		ID:                  order.ID,
		UserID:              order.UserID,
		UserEmail:           order.UserEmail,
		UserName:            order.UserName,
		UserNotes:           order.UserNotes,
		Amount:              order.Amount,
		PayAmount:           order.PayAmount,
		FeeRate:             order.FeeRate,
		Currency:            service.PaymentOrderCurrency(order),
		PaymentType:         order.PaymentType,
		OutTradeNo:          order.OutTradeNo,
		Status:              order.Status,
		OrderType:           order.OrderType,
		CreatedAt:           order.CreatedAt,
		ExpiresAt:           order.ExpiresAt,
		PaidAt:              order.PaidAt,
		CompletedAt:         order.CompletedAt,
		FailedAt:            order.FailedAt,
		FailedReason:        order.FailedReason,
		RefundAmount:        order.RefundAmount,
		RefundReason:        order.RefundReason,
		RefundRequestedAt:   order.RefundRequestedAt,
		RefundRequestedBy:   order.RefundRequestedBy,
		RefundRequestReason: order.RefundRequestReason,
		PlanID:              order.PlanID,
		ProviderInstanceID:  order.ProviderInstanceID,
		Invoice:             sanitizeAdminPaymentInvoiceSummary(order.Edges.Invoice),
	}
}

// AdminProcessRefundRequest is the request body for admin refund processing.
type AdminProcessRefundRequest struct {
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
	Force         bool    `json:"force"`
	DeductBalance bool    `json:"deduct_balance"`
}

// ProcessRefund processes a refund for an order (admin).
// POST /api/v1/admin/payment/orders/:id/refund
func (h *PaymentHandler) ProcessRefund(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req AdminProcessRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	plan, earlyResult, err := h.paymentService.PrepareRefund(c.Request.Context(), orderID, req.Amount, req.Reason, req.Force, req.DeductBalance)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if earlyResult != nil {
		response.Success(c, earlyResult)
		return
	}

	result, err := h.paymentService.ExecuteRefund(c.Request.Context(), plan)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// --- Invoices ---

// ListInvoices returns a paginated list of invoice requests.
// GET /api/v1/admin/payment/invoices
func (h *PaymentHandler) ListInvoices(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	invoices, total, err := h.paymentService.ListInvoices(c.Request.Context(), service.PaymentInvoiceListParams{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Keyword:  c.Query("keyword"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, sanitizeAdminPaymentInvoicesForResponse(invoices), int64(total), page, pageSize)
}

// GetInvoiceDetail returns detailed information for a single invoice request.
// GET /api/v1/admin/payment/invoices/:id
func (h *PaymentHandler) GetInvoiceDetail(c *gin.Context) {
	invoiceID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	invoice, err := h.paymentService.GetInvoiceByID(c.Request.Context(), invoiceID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sanitizeAdminPaymentInvoiceForResponse(invoice))
}

type AdminIssueInvoiceRequest struct {
	FileHeader string `form:"-"`
}

// IssueInvoice uploads a PDF invoice file and marks the request as issued.
// POST /api/v1/admin/payment/invoices/:id/issue
func (h *PaymentHandler) IssueInvoice(c *gin.Context) {
	invoiceID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "Invoice file is required")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.ErrorFrom(c, fmt.Errorf("open invoice file: %w", err))
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		response.ErrorFrom(c, fmt.Errorf("read invoice file: %w", err))
		return
	}

	operator := "admin"
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		operator = fmt.Sprintf("admin:%d", subject.UserID)
	}

	invoice, err := h.paymentService.MarkInvoiceIssued(c.Request.Context(), invoiceID, fileHeader.Filename, content, operator)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sanitizeAdminPaymentInvoiceForResponse(invoice))
}

type AdminFailInvoiceRequest struct {
	Reason string `json:"reason"`
}

// FailInvoice marks an invoice request as failed with a reason.
// POST /api/v1/admin/payment/invoices/:id/fail
func (h *PaymentHandler) FailInvoice(c *gin.Context) {
	invoiceID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req AdminFailInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	operator := "admin"
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		operator = fmt.Sprintf("admin:%d", subject.UserID)
	}

	invoice, err := h.paymentService.MarkInvoiceFailed(c.Request.Context(), invoiceID, req.Reason, operator)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sanitizeAdminPaymentInvoiceForResponse(invoice))
}

// DownloadInvoice streams an issued invoice file to the administrator.
// GET /api/v1/admin/payment/invoices/:id/download
func (h *PaymentHandler) DownloadInvoice(c *gin.Context) {
	invoiceID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	invoice, absolutePath, err := h.paymentService.PrepareAdminInvoiceDownload(c.Request.Context(), invoiceID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if invoice.ContentType != nil && strings.TrimSpace(*invoice.ContentType) != "" {
		c.Header("Content-Type", strings.TrimSpace(*invoice.ContentType))
	}
	fileName := fmt.Sprintf("invoice-%d.pdf", invoice.ID)
	if invoice.FileName != nil && strings.TrimSpace(*invoice.FileName) != "" {
		fileName = strings.TrimSpace(*invoice.FileName)
	}
	c.FileAttachment(absolutePath, fileName)
}

func sanitizeAdminPaymentInvoiceSummary(invoice *dbent.PaymentInvoice) *AdminPaymentInvoiceSummary {
	if invoice == nil {
		return nil
	}
	return &AdminPaymentInvoiceSummary{
		ID:             invoice.ID,
		UserID:         invoice.UserID,
		TitleName:      invoice.TitleName,
		TaxID:          invoice.TaxID,
		Status:         invoice.Status,
		RequestedAt:    invoice.RequestedAt,
		IssuedAt:       invoice.IssuedAt,
		FailedAt:       invoice.FailedAt,
		FailedReason:   invoice.FailedReason,
		FileName:       invoice.FileName,
		ContentType:    invoice.ContentType,
		ByteSize:       invoice.ByteSize,
		OrderCount:     len(invoice.Edges.Orders),
		TotalAmount:    sumAdminInvoiceOrderAmount(invoice.Edges.Orders),
		TotalPayAmount: sumAdminInvoiceOrderPayAmount(invoice.Edges.Orders),
	}
}

func sanitizeAdminPaymentInvoiceForResponse(invoice *dbent.PaymentInvoice) *AdminPaymentInvoiceResult {
	if invoice == nil {
		return nil
	}

	orders := make([]AdminPaymentInvoiceOrder, 0, len(invoice.Edges.Orders))
	for _, order := range invoice.Edges.Orders {
		if order == nil {
			continue
		}
		orders = append(orders, AdminPaymentInvoiceOrder{
			ID:          order.ID,
			OrderUUID:   service.PaymentOrderUUID(order.ID),
			UserID:      order.UserID,
			UserEmail:   order.UserEmail,
			UserName:    order.UserName,
			UserNotes:   order.UserNotes,
			OutTradeNo:  order.OutTradeNo,
			Status:      order.Status,
			OrderType:   order.OrderType,
			PaymentType: order.PaymentType,
			Amount:      order.Amount,
			PayAmount:   order.PayAmount,
			CreatedAt:   order.CreatedAt,
			CompletedAt: order.CompletedAt,
		})
	}

	return &AdminPaymentInvoiceResult{
		AdminPaymentInvoiceSummary: *sanitizeAdminPaymentInvoiceSummary(invoice),
		Orders:                     orders,
	}
}

func sanitizeAdminPaymentInvoicesForResponse(invoices []*dbent.PaymentInvoice) []AdminPaymentInvoiceResult {
	out := make([]AdminPaymentInvoiceResult, 0, len(invoices))
	for _, invoice := range invoices {
		if item := sanitizeAdminPaymentInvoiceForResponse(invoice); item != nil {
			out = append(out, *item)
		}
	}
	return out
}

func sumAdminInvoiceOrderAmount(orders []*dbent.PaymentOrder) float64 {
	total := 0.0
	for _, order := range orders {
		if order == nil {
			continue
		}
		total += order.Amount
	}
	return total
}

func sumAdminInvoiceOrderPayAmount(orders []*dbent.PaymentOrder) float64 {
	total := 0.0
	for _, order := range orders {
		if order == nil {
			continue
		}
		total += order.PayAmount
	}
	return total
}

// --- Subscription Plans ---

// ListPlans returns all subscription plans.
// GET /api/v1/admin/payment/plans
func (h *PaymentHandler) ListPlans(c *gin.Context) {
	plans, err := h.configService.ListPlans(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, plans)
}

// CreatePlan creates a new subscription plan.
// POST /api/v1/admin/payment/plans
func (h *PaymentHandler) CreatePlan(c *gin.Context) {
	var req service.CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	plan, err := h.configService.CreatePlan(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, plan)
}

// UpdatePlan updates an existing subscription plan.
// PUT /api/v1/admin/payment/plans/:id
func (h *PaymentHandler) UpdatePlan(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	plan, err := h.configService.UpdatePlan(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, plan)
}

// DeletePlan deletes a subscription plan.
// DELETE /api/v1/admin/payment/plans/:id
func (h *PaymentHandler) DeletePlan(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.configService.DeletePlan(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// --- Provider Instances ---

// ListProviders returns all payment provider instances.
// GET /api/v1/admin/payment/providers
func (h *PaymentHandler) ListProviders(c *gin.Context) {
	providers, err := h.configService.ListProviderInstancesWithConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, providers)
}

// CreateProvider creates a new payment provider instance.
// POST /api/v1/admin/payment/providers
func (h *PaymentHandler) CreateProvider(c *gin.Context) {
	var req service.CreateProviderInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	inst, err := h.configService.CreateProviderInstance(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request.Context())
	response.Created(c, inst)
}

// UpdateProvider updates an existing payment provider instance.
// PUT /api/v1/admin/payment/providers/:id
func (h *PaymentHandler) UpdateProvider(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdateProviderInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	inst, err := h.configService.UpdateProviderInstance(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request.Context())
	response.Success(c, inst)
}

// DeleteProvider deletes a payment provider instance.
// DELETE /api/v1/admin/payment/providers/:id
func (h *PaymentHandler) DeleteProvider(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.configService.DeleteProviderInstance(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request.Context())
	response.Success(c, gin.H{"message": "deleted"})
}

// parseIDParam parses an int64 path parameter.
// Returns the parsed ID and true on success; on failure it writes a BadRequest response and returns false.
func parseIDParam(c *gin.Context, paramName string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(paramName), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid "+paramName)
		return 0, false
	}
	return id, true
}

// --- Config ---

// GetConfig returns the payment configuration (admin view).
// GET /api/v1/admin/payment/config
func (h *PaymentHandler) GetConfig(c *gin.Context) {
	cfg, err := h.configService.GetPaymentConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

// UpdateConfig updates the payment configuration.
// PUT /api/v1/admin/payment/config
func (h *PaymentHandler) UpdateConfig(c *gin.Context) {
	var req service.UpdatePaymentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.configService.UpdatePaymentConfig(c.Request.Context(), req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "updated"})
}
