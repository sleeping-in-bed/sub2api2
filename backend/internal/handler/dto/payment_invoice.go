package dto

import (
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type PaymentInvoiceSummary struct {
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

type PaymentInvoiceOrder struct {
	ID          int64      `json:"id"`
	OrderUUID   string     `json:"order_uuid"`
	Status      string     `json:"status"`
	OrderType   string     `json:"order_type"`
	PaymentType string     `json:"payment_type"`
	OutTradeNo  string     `json:"out_trade_no"`
	Amount      float64    `json:"amount"`
	PayAmount   float64    `json:"pay_amount"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type PaymentInvoice struct {
	PaymentInvoiceSummary
	Orders []PaymentInvoiceOrder `json:"orders"`
}

func PaymentInvoiceFromEnt(invoice *dbent.PaymentInvoice) *PaymentInvoice {
	if invoice == nil {
		return nil
	}
	result := &PaymentInvoice{Orders: make([]PaymentInvoiceOrder, 0, len(invoice.Edges.Orders))}
	result.ID = invoice.ID
	result.UserID = invoice.UserID
	result.TitleName = invoice.TitleName
	result.TaxID = invoice.TaxID
	result.Status = invoice.Status
	result.RequestedAt = invoice.RequestedAt
	result.IssuedAt = invoice.IssuedAt
	result.FailedAt = invoice.FailedAt
	result.FailedReason = invoice.FailedReason
	result.FileName = invoice.FileName
	result.ContentType = invoice.ContentType
	result.ByteSize = invoice.ByteSize
	for _, order := range invoice.Edges.Orders {
		if order == nil {
			continue
		}
		result.OrderCount++
		result.TotalAmount += order.Amount
		result.TotalPayAmount += order.PayAmount
		result.Orders = append(result.Orders, PaymentInvoiceOrder{
			ID: order.ID, OrderUUID: service.PaymentOrderUUID(order.ID), Status: order.Status, OrderType: order.OrderType, PaymentType: order.PaymentType,
			OutTradeNo: order.OutTradeNo, Amount: order.Amount, PayAmount: order.PayAmount,
			CreatedAt: order.CreatedAt, CompletedAt: order.CompletedAt,
		})
	}
	return result
}

func PaymentInvoicesFromEnt(invoices []*dbent.PaymentInvoice) []PaymentInvoice {
	result := make([]PaymentInvoice, 0, len(invoices))
	for _, invoice := range invoices {
		if item := PaymentInvoiceFromEnt(invoice); item != nil {
			result = append(result, *item)
		}
	}
	return result
}
