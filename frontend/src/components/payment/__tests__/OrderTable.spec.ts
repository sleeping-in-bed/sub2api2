import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import OrderTable from '../OrderTable.vue'
import type { PaymentOrder } from '@/types/payment'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

function buildOrder(overrides: Partial<PaymentOrder> = {}): PaymentOrder {
  return {
    id: 42,
    order_uuid: '41a4b0f3-d88f-5fdc-9b1f-9ff3f0e6f42b',
    user_id: 7,
    amount: 88,
    pay_amount: 88,
    fee_rate: 0,
    payment_type: 'alipay',
    out_trade_no: 'sub2_20260626abcd1234',
    status: 'PENDING',
    order_type: 'balance',
    created_at: '2026-06-26T12:00:00Z',
    expires_at: '2026-06-26T12:30:00Z',
    refund_amount: 0,
    ...overrides,
  }
}

describe('OrderTable', () => {
  it('默认显示订单 ID 列', () => {
    const wrapper = mount(OrderTable, {
      props: {
        orders: [buildOrder()],
        loading: false,
      },
      global: {
        stubs: {
          DataTable: {
            props: ['columns'],
            template: '<div>{{ columns.map(column => column.key).join(",") }}</div>',
          },
          OrderStatusBadge: true,
        },
      },
    })

    expect(wrapper.text()).toContain('order_uuid')
    expect(wrapper.text()).not.toContain('out_trade_no')
  })

  it('显式关闭后不显示订单 ID 列', () => {
    const wrapper = mount(OrderTable, {
      props: {
        orders: [buildOrder()],
        loading: false,
        showOrderId: false,
      },
      global: {
        stubs: {
          DataTable: {
            props: ['columns'],
            template: '<div>{{ columns.map(column => column.key).join(",") }}</div>',
          },
          OrderStatusBadge: true,
        },
      },
    })

    expect(wrapper.text()).not.toContain('order_uuid')
  })
})
