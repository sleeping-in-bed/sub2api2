<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 pb-4 dark:border-dark-600">
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('payment.invoices.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('payment.invoices.available', { amount: summary.available_pay_amount.toFixed(2), count: summary.available_order_count }) }}
          </p>
        </div>
        <button class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="loadAll">
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>
      </div>

      <section class="space-y-3">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('payment.invoices.selectOrders') }}</h2>
        <div class="overflow-x-auto border border-gray-200 dark:border-dark-600">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="w-12 px-4 py-3"></th>
                <th class="px-4 py-3 text-left">{{ t('payment.orders.orderNo') }}</th>
                <th class="px-4 py-3 text-right">{{ t('payment.orders.payAmount') }}</th>
                <th class="px-4 py-3 text-left">{{ t('payment.orders.completedAt') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-600 dark:bg-dark-900">
              <tr v-for="order in availableOrders" :key="order.id">
                <td class="px-4 py-3"><input v-model="selectedOrderIDs" type="checkbox" :value="order.id" /></td>
                <td class="px-4 py-3 font-mono text-gray-700 dark:text-gray-300">{{ order.out_trade_no }}</td>
                <td class="px-4 py-3 text-right font-medium text-gray-900 dark:text-white">{{ order.pay_amount.toFixed(2) }}</td>
                <td class="px-4 py-3 text-gray-500 dark:text-gray-400">{{ formatDate(order.completed_at) }}</td>
              </tr>
              <tr v-if="availableOrders.length === 0">
                <td colspan="4" class="px-4 py-8 text-center text-gray-500">{{ t('payment.invoices.noAvailableOrders') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <input v-model="titleName" class="input" :placeholder="t('payment.invoices.titleName')" />
          <input v-model="taxID" class="input" :placeholder="t('payment.invoices.taxId')" />
        </div>
        <div class="flex justify-end">
          <button class="btn btn-primary" :disabled="submitting || selectedOrderIDs.length === 0 || !titleName.trim() || !taxID.trim()" @click="submitInvoice">
            <Icon name="document" size="md" />
            {{ submitting ? t('common.processing') : t('payment.invoices.submit') }}
          </button>
        </div>
      </section>

      <section class="space-y-3">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('payment.invoices.history') }}</h2>
        <div class="overflow-x-auto border border-gray-200 dark:border-dark-600">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left">ID</th>
                <th class="px-4 py-3 text-left">{{ t('payment.invoices.titleName') }}</th>
                <th class="px-4 py-3 text-right">{{ t('payment.invoices.amount') }}</th>
                <th class="px-4 py-3 text-left">{{ t('payment.orders.status') }}</th>
                <th class="px-4 py-3 text-right">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-600 dark:bg-dark-900">
              <tr v-for="invoice in invoices" :key="invoice.id">
                <td class="px-4 py-3 font-mono">#{{ invoice.id }}</td>
                <td class="px-4 py-3">{{ invoice.title_name }}</td>
                <td class="px-4 py-3 text-right">{{ invoice.total_pay_amount.toFixed(2) }}</td>
                <td class="px-4 py-3">{{ t(`payment.invoices.status.${invoice.status}`) }}</td>
                <td class="px-4 py-3 text-right">
                  <button v-if="invoice.status === 'ISSUED'" class="btn btn-secondary" :title="t('common.download')" @click="download(invoice)">
                    <Icon name="download" size="md" />
                  </button>
                  <span v-else-if="invoice.failed_reason" class="text-xs text-red-600 dark:text-red-400">{{ invoice.failed_reason }}</span>
                </td>
              </tr>
              <tr v-if="invoices.length === 0">
                <td colspan="5" class="px-4 py-8 text-center text-gray-500">{{ t('payment.invoices.noHistory') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { paymentAPI } from '@/api/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import type { PaymentInvoice, PaymentInvoiceSummaryResponse, PaymentOrder } from '@/types/payment'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const submitting = ref(false)
const summary = ref<PaymentInvoiceSummaryResponse>({ available_pay_amount: 0, available_order_count: 0, minimum_pay_amount: 100 })
const availableOrders = ref<PaymentOrder[]>([])
const invoices = ref<PaymentInvoice[]>([])
const selectedOrderIDs = ref<number[]>([])
const titleName = ref('')
const taxID = ref('')

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : '-'
}

async function loadAll() {
  loading.value = true
  try {
    const [summaryResponse, ordersResponse, invoicesResponse] = await Promise.all([
      paymentAPI.getInvoiceSummary(),
      paymentAPI.getInvoiceAvailableOrders({ page: 1, page_size: 100 }),
      paymentAPI.getMyInvoices({ page: 1, page_size: 100 })
    ])
    summary.value = summaryResponse.data
    availableOrders.value = ordersResponse.data.items || []
    invoices.value = invoicesResponse.data.items || []
    selectedOrderIDs.value = selectedOrderIDs.value.filter(id => availableOrders.value.some(order => order.id === id))
  } catch (error) {
    appStore.showError((error as Error).message)
  } finally {
    loading.value = false
  }
}

async function submitInvoice() {
  submitting.value = true
  try {
    await paymentAPI.createInvoice({ order_ids: selectedOrderIDs.value, title_name: titleName.value.trim(), tax_id: taxID.value.trim() })
    selectedOrderIDs.value = []
    titleName.value = ''
    taxID.value = ''
    appStore.showSuccess(t('payment.invoices.submitted'))
    await loadAll()
  } catch (error) {
    appStore.showError((error as Error).message)
  } finally {
    submitting.value = false
  }
}

async function download(invoice: PaymentInvoice) {
  const blob = await paymentAPI.downloadInvoice(invoice.id)
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = invoice.file_name || `invoice-${invoice.id}.pdf`
  link.click()
  URL.revokeObjectURL(url)
}

onMounted(loadAll)
</script>
