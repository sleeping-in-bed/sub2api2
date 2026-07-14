ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS invoice_id BIGINT NULL REFERENCES payment_invoices(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS payment_orders_invoice_id_idx
    ON payment_orders (invoice_id)
    WHERE invoice_id IS NOT NULL;

UPDATE payment_orders AS po
SET invoice_id = pi.id
FROM payment_invoices AS pi
WHERE pi.order_id = po.id
  AND (po.invoice_id IS NULL OR po.invoice_id <> pi.id);

ALTER TABLE payment_invoices
    DROP CONSTRAINT IF EXISTS payment_invoices_order_id_key;

ALTER TABLE payment_invoices
    DROP COLUMN IF EXISTS order_id;
