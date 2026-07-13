import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../UserOrdersView.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('UserOrdersView invoice entry', () => {
  it('shows the invoice summary and opens invoice management from the orders page', () => {
    expect(componentSource).toContain("t('payment.invoice.availableAmount')")
    expect(componentSource).toContain("router.push('/invoices')")
    expect(componentSource).toContain('paymentAPI.getInvoiceSummary()')
    expect(componentSource).toContain('loadInvoiceSummary()')
  })
})
