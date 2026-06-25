package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PaymentInvoice holds the schema definition for the PaymentInvoice entity.
//
// 删除策略：硬删除
// PaymentInvoice 使用硬删除而非软删除，原因如下：
//   - 发票申请与订单是一对一的附属数据，订单删除时可一并清理
//   - 发票状态通过 status 字段追踪，无需额外软删除过滤
//   - 减少查询复杂度，避免后台列表额外处理软删除条件
type PaymentInvoice struct {
	ent.Schema
}

func (PaymentInvoice) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "payment_invoices"},
	}
}

func (PaymentInvoice) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("order_id"),
		field.Int64("user_id"),
		field.String("title_name").
			MaxLen(200),
		field.String("tax_id").
			MaxLen(32),
		field.String("status").
			MaxLen(30).
			Default("REQUESTED"),
		field.Time("requested_at").
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("issued_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("failed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("failed_reason").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("storage_provider").
			MaxLen(20).
			Default("local"),
		field.String("storage_key").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("file_name").
			Optional().
			Nillable().
			MaxLen(255),
		field.String("content_type").
			Optional().
			Nillable().
			MaxLen(100),
		field.Int64("byte_size").
			Default(0),
		field.String("sha256").
			Optional().
			Nillable().
			MaxLen(64),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (PaymentInvoice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", PaymentOrder.Type).
			Ref("invoice").
			Field("order_id").
			Unique().
			Required(),
		edge.From("user", User.Type).
			Ref("payment_invoices").
			Field("user_id").
			Unique().
			Required(),
	}
}

func (PaymentInvoice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id").
			Unique(),
		index.Fields("user_id"),
		index.Fields("status"),
		index.Fields("requested_at"),
		index.Fields("issued_at"),
		index.Fields("created_at"),
	}
}
