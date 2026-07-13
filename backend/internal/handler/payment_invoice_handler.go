package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type CreateInvoiceRequestBody struct {
	OrderIDs  []int64 `json:"order_ids" binding:"required,min=1"`
	TitleName string  `json:"title_name" binding:"required"`
	TaxID     string  `json:"tax_id" binding:"required"`
}

func (h *PaymentHandler) GetInvoiceSummary(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	amount, count, err := h.paymentService.GetInvoiceAvailableSummary(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"available_pay_amount": amount, "available_order_count": count, "minimum_pay_amount": 100})
}

func (h *PaymentHandler) ListInvoiceAvailableOrders(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	orders, total, err := h.paymentService.ListInvoiceAvailableOrders(c.Request.Context(), subject.UserID, service.OrderListParams{Page: page, PageSize: pageSize})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, sanitizePaymentOrdersForResponse(orders), int64(total), page, pageSize)
}

func (h *PaymentHandler) ListMyInvoices(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	invoices, total, err := h.paymentService.ListUserInvoices(c.Request.Context(), subject.UserID, service.PaymentInvoiceListParams{Page: page, PageSize: pageSize, Status: c.Query("status")})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, dto.PaymentInvoicesFromEnt(invoices), int64(total), page, pageSize)
}

func (h *PaymentHandler) GetMyInvoice(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	invoiceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid invoice ID")
		return
	}
	invoice, err := h.paymentService.GetUserInvoiceByID(c.Request.Context(), invoiceID, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.PaymentInvoiceFromEnt(invoice))
}

func (h *PaymentHandler) CreateInvoice(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req CreateInvoiceRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	invoice, err := h.paymentService.RequestInvoice(c.Request.Context(), req.OrderIDs, subject.UserID, req.TitleName, req.TaxID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.PaymentInvoiceFromEnt(invoice))
}

func (h *PaymentHandler) DownloadInvoice(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	invoiceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid invoice ID")
		return
	}
	invoice, path, err := h.paymentService.PrepareUserInvoiceDownload(c.Request.Context(), invoiceID, subject.UserID)
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
