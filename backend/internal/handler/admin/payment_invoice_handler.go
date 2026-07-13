package admin

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *PaymentHandler) ListInvoices(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	invoices, total, err := h.paymentService.ListInvoices(c.Request.Context(), service.PaymentInvoiceListParams{
		Page: page, PageSize: pageSize, Status: c.Query("status"), Keyword: c.Query("keyword"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, dto.PaymentInvoicesFromEnt(invoices), int64(total), page, pageSize)
}

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
	response.Success(c, dto.PaymentInvoiceFromEnt(invoice))
}

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
	if fileHeader.Size > 10<<20 {
		response.BadRequest(c, "Invoice file is too large")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.ErrorFrom(c, fmt.Errorf("open invoice file: %w", err))
		return
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(io.LimitReader(file, (10<<20)+1))
	if err != nil {
		response.ErrorFrom(c, fmt.Errorf("read invoice file: %w", err))
		return
	}
	operator := "admin:0"
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		operator = "admin:" + strconv.FormatInt(subject.UserID, 10)
	}
	invoice, err := h.paymentService.MarkInvoiceIssued(c.Request.Context(), invoiceID, fileHeader.Filename, content, operator)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.PaymentInvoiceFromEnt(invoice))
}

func (h *PaymentHandler) FailInvoice(c *gin.Context) {
	invoiceID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	operator := "admin:0"
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		operator = "admin:" + strconv.FormatInt(subject.UserID, 10)
	}
	invoice, err := h.paymentService.MarkInvoiceFailed(c.Request.Context(), invoiceID, req.Reason, operator)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.PaymentInvoiceFromEnt(invoice))
}

func (h *PaymentHandler) DownloadInvoice(c *gin.Context) {
	invoiceID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	invoice, path, err := h.paymentService.PrepareAdminInvoiceDownload(c.Request.Context(), invoiceID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	fileName := fmt.Sprintf("invoice-%d.pdf", invoice.ID)
	if invoice.FileName != nil && strings.TrimSpace(*invoice.FileName) != "" {
		fileName = strings.TrimSpace(*invoice.FileName)
	}
	c.FileAttachment(path, fileName)
}
