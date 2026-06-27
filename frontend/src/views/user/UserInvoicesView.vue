<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex flex-wrap items-center gap-3">
          <div class="rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-300">
            <div class="font-semibold">{{ t('payment.invoice.availableAmount') }}</div>
            <div class="mt-1 text-xl font-bold">¥{{ invoiceSummary.available_pay_amount.toFixed(2) }}</div>
            <div class="mt-1 text-xs opacity-80">
              {{ t('payment.invoice.amountThresholdHint', { amount: invoiceSummary.minimum_pay_amount.toFixed(2) }) }}
            </div>
          </div>
          <div class="rounded-2xl bg-sky-50 px-4 py-3 text-sm text-sky-800 dark:bg-sky-900/20 dark:text-sky-300">
            <div class="font-semibold">{{ t('payment.invoice.selectedAmount') }}</div>
            <div class="mt-1 text-xl font-bold">¥{{ selectedPayAmount.toFixed(2) }}</div>
            <div class="mt-1 text-xs opacity-80">
              {{ t('payment.invoice.selectedOrders') }}: {{ selectedOrders.length }}
            </div>
          </div>
          <div class="flex flex-1 items-center justify-end gap-2">
            <button class="btn btn-secondary" @click="refreshCurrentTab" :disabled="loadingAvailable || loadingInvoices">
              {{ t('common.refresh') }}
            </button>
            <button class="btn btn-primary" :disabled="activeTab !== 'create' || selectedOrders.length === 0 || selectedPayAmount < invoiceSummary.minimum_pay_amount" @click="showCreateDialog = true">
              {{ t('payment.invoice.createRequest') }}
            </button>
          </div>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="flex gap-2 border-b border-gray-200 px-4 pt-3 dark:border-dark-700">
          <button class="tab" :class="{ 'tab-active': activeTab === 'create' }" @click="setActiveTab('create')">
            {{ t('payment.invoice.tabs.create') }}
          </button>
          <button class="tab" :class="{ 'tab-active': activeTab === 'history' }" @click="setActiveTab('history')">
            {{ t('payment.invoice.tabs.history') }}
          </button>
        </div>

        <div v-if="activeTab === 'create'" class="space-y-4 p-4">
          <div class="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-gray-50 px-4 py-3 text-sm dark:bg-dark-800">
            <div class="text-gray-600 dark:text-gray-300">{{ t('payment.invoice.createdOrdersHint') }}</div>
            <div class="flex flex-wrap items-center gap-2">
              <button class="btn btn-secondary" @click="toggleVisibleSelection(!allVisibleSelected)" :disabled="availableOrders.length === 0">
                {{ t('payment.invoice.selectAllVisible') }}
              </button>
              <button class="btn btn-secondary" @click="selectAllEligibleOrders" :disabled="invoiceSummary.available_order_count === 0 || selectingAll">
                {{ selectingAll ? t('common.processing') : t('common.selectAll') }}
              </button>
              <button class="btn btn-secondary" @click="clearSelection" :disabled="selectedOrders.length === 0">
                {{ t('payment.invoice.clearSelection') }}
              </button>
            </div>
          </div>

          <DataTable :columns="availableOrderColumns" :data="availableOrders" :loading="loadingAvailable">
            <template #cell-select="{ row }">
              <input
                type="checkbox"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="isSelected(row.id)"
                @change="toggleSelection(row)"
              />
            </template>
            <template #cell-order="{ row }">
              <div class="text-sm">
                <div class="font-mono text-gray-900 dark:text-white">{{ row.order_uuid }}</div>
              </div>
            </template>
            <template #cell-pay_amount="{ row }">
              <span class="text-sm font-semibold text-gray-900 dark:text-white">¥{{ row.pay_amount.toFixed(2) }}</span>
            </template>
            <template #cell-created_at="{ row }">
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(row.completed_at || row.created_at) }}</span>
            </template>
            <template #empty>
              <div class="py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('payment.invoice.noEligibleOrders') }}
              </div>
            </template>
          </DataTable>

          <Pagination
            v-if="availablePagination.total > 0"
            :page="availablePagination.page"
            :total="availablePagination.total"
            :page-size="availablePagination.page_size"
            @update:page="handleAvailablePageChange"
            @update:pageSize="handleAvailablePageSizeChange"
          />
        </div>

        <div v-else class="space-y-4 p-4">
          <div class="rounded-xl bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-gray-300">
            {{ t('payment.invoice.historyHint') }}
          </div>

          <DataTable :columns="historyColumns" :data="invoices" :loading="loadingInvoices">
            <template #cell-summary="{ row }">
              <div class="text-sm">
                <div class="font-medium text-gray-900 dark:text-white">{{ t('payment.invoice.orderCount') }}: {{ row.order_count }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">¥{{ row.total_pay_amount.toFixed(2) }}</div>
              </div>
            </template>
            <template #cell-title_name="{ row }">
              <div class="text-sm text-gray-900 dark:text-white">{{ row.title_name }}</div>
            </template>
            <template #cell-status="{ row }">
              <span class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium" :class="invoiceStatusClass(row.status)">
                {{ invoiceStatusLabel(row.status) }}
              </span>
            </template>
            <template #cell-requested_at="{ row }">
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(row.requested_at) }}</span>
            </template>
            <template #cell-actions="{ row }">
              <div class="flex items-center gap-2">
                <button class="btn btn-secondary" @click="openHistoryDetail(row)">{{ t('common.view') }}</button>
                <button v-if="row.status === 'ISSUED'" class="btn btn-primary" :disabled="downloadingInvoiceId === row.id" @click="downloadInvoice(row)">
                  {{ downloadingInvoiceId === row.id ? t('common.processing') : t('payment.invoice.download') }}
                </button>
              </div>
            </template>
            <template #empty>
              <div class="py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('payment.invoice.noHistory') }}
              </div>
            </template>
          </DataTable>

          <Pagination
            v-if="invoicePagination.total > 0"
            :page="invoicePagination.page"
            :total="invoicePagination.total"
            :page-size="invoicePagination.page_size"
            @update:page="handleInvoicePageChange"
            @update:pageSize="handleInvoicePageSizeChange"
          />
        </div>
      </div>
    </div>

    <BaseDialog :show="showCreateDialog" :title="t('payment.invoice.title')" width="wide" @close="showCreateDialog = false">
      <div class="space-y-4">
        <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
          <div class="mb-3 text-sm font-medium text-gray-900 dark:text-white">{{ t('payment.invoice.orderList') }}</div>
          <div class="max-h-72 space-y-3 overflow-y-auto">
            <div v-for="order in selectedOrders" :key="order.id" class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-900">
              <div class="grid grid-cols-1 gap-2 text-sm sm:grid-cols-3">
                <div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.orders.orderId') }}</div>
                  <div class="font-mono text-gray-900 dark:text-white">{{ order.order_uuid }}</div>
                </div>
                <div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.orders.amount') }}</div>
                  <div class="font-semibold text-gray-900 dark:text-white">¥{{ order.pay_amount.toFixed(2) }}</div>
                </div>
              </div>
            </div>
          </div>
          <div class="mt-3 flex items-center justify-between border-t border-gray-200 pt-3 text-sm dark:border-dark-600">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.invoice.selectedAmount') }}</span>
            <span class="font-semibold text-gray-900 dark:text-white">¥{{ selectedPayAmount.toFixed(2) }}</span>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('payment.invoice.titleName') }}</label>
          <input v-model="invoiceForm.title_name" type="text" class="input mt-1 w-full" :placeholder="t('payment.invoice.titleNamePlaceholder')" />
        </div>
        <div>
          <label class="input-label">{{ t('payment.invoice.taxId') }}</label>
          <input v-model="invoiceForm.tax_id" type="text" class="input mt-1 w-full" :placeholder="t('payment.invoice.taxIdPlaceholder')" />
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="showCreateDialog = false">{{ t('common.close') }}</button>
          <button class="btn btn-primary" :disabled="invoiceSubmitting || !canSubmitInvoice" @click="submitInvoice">
            {{ invoiceSubmitting ? t('common.processing') : t('payment.invoice.submit') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="showHistoryDialog" :title="t('payment.invoice.title')" width="wide" @close="showHistoryDialog = false">
      <div v-if="historyDetail" class="space-y-4">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.titleName') }}</div>
            <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ historyDetail.title_name }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.taxId') }}</div>
            <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ historyDetail.tax_id }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.statusLabel') }}</div>
            <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ invoiceStatusLabel(historyDetail.status) }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.requestedAt') }}</div>
            <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ formatDateTime(historyDetail.requested_at) }}</div>
          </div>
          <div v-if="historyDetail.failed_reason" class="sm:col-span-2">
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.failedReason') }}</div>
            <div class="mt-1 text-sm text-red-600 dark:text-red-400">{{ historyDetail.failed_reason }}</div>
          </div>
        </div>

        <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
          <div class="mb-3 flex items-center justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.invoice.orderCount') }}</span>
            <span class="font-semibold text-gray-900 dark:text-white">{{ historyDetail.order_count }}</span>
          </div>
          <div class="mb-3 flex items-center justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.invoice.selectedAmount') }}</span>
            <span class="font-semibold text-gray-900 dark:text-white">¥{{ historyDetail.total_pay_amount.toFixed(2) }}</span>
          </div>
          <div class="space-y-3">
            <div v-for="order in historyDetail.orders || []" :key="order.id" class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-900">
              <div class="grid grid-cols-1 gap-2 text-sm sm:grid-cols-3">
                <div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.orders.orderId') }}</div>
                  <div class="font-mono text-gray-900 dark:text-white">{{ order.order_uuid }}</div>
                </div>
                <div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.orders.amount') }}</div>
                  <div class="font-semibold text-gray-900 dark:text-white">¥{{ order.pay_amount.toFixed(2) }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="showHistoryDialog = false">{{ t('common.close') }}</button>
          <button
            v-if="historyDetail?.status === 'ISSUED'"
            class="btn btn-primary"
            :disabled="downloadingInvoiceId === historyDetail.id"
            @click="downloadInvoice(historyDetail)"
          >
            {{ downloadingInvoiceId === historyDetail.id ? t('common.processing') : t('payment.invoice.download') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import type { Column } from '@/components/common/types'
import { paymentAPI } from '@/api/payment'
import type { PaymentInvoice, PaymentOrder } from '@/types/payment'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const activeTab = ref<'create' | 'history'>('create')
const loadingAvailable = ref(false)
const loadingInvoices = ref(false)
const selectingAll = ref(false)
const invoiceSubmitting = ref(false)
const downloadingInvoiceId = ref<number | null>(null)
const showCreateDialog = ref(false)
const showHistoryDialog = ref(false)
const availableOrders = ref<PaymentOrder[]>([])
const invoices = ref<PaymentInvoice[]>([])
const historyDetail = ref<PaymentInvoice | null>(null)
const selectedOrderMap = ref<Map<number, PaymentOrder>>(new Map())
const invoiceForm = reactive({
  title_name: '',
  tax_id: '',
})
const invoiceSummary = reactive({
  available_pay_amount: 0,
  available_order_count: 0,
  minimum_pay_amount: 100,
})
const availablePagination = reactive({ page: 1, page_size: 20, total: 0 })
const invoicePagination = reactive({ page: 1, page_size: 20, total: 0 })

const availableOrderColumns = computed((): Column[] => [
  { key: 'select', label: '' },
  { key: 'order', label: t('payment.orders.orderId') },
  { key: 'pay_amount', label: t('payment.invoice.availableAmount') },
  { key: 'created_at', label: t('payment.orders.createdAt') },
])

const historyColumns = computed((): Column[] => [
  { key: 'summary', label: t('payment.invoice.orderCount') },
  { key: 'title_name', label: t('payment.invoice.titleName') },
  { key: 'status', label: t('payment.invoice.statusLabel') },
  { key: 'requested_at', label: t('payment.invoice.requestedAt') },
  { key: 'actions', label: t('common.actions') },
])

const selectedOrders = computed(() => Array.from(selectedOrderMap.value.values()).sort((left, right) => left.id - right.id))
const selectedPayAmount = computed(() => selectedOrders.value.reduce((total, order) => total + order.pay_amount, 0))
const allVisibleSelected = computed(() => availableOrders.value.length > 0 && availableOrders.value.every((order) => selectedOrderMap.value.has(order.id)))
const canSubmitInvoice = computed(() => {
  return selectedOrders.value.length > 0 &&
    selectedPayAmount.value >= invoiceSummary.minimum_pay_amount &&
    invoiceForm.title_name.trim() !== '' &&
    invoiceForm.tax_id.trim() !== ''
})

function isSelected(orderId: number): boolean {
  return selectedOrderMap.value.has(orderId)
}

function setSelectedOrderMap(next: Map<number, PaymentOrder>) {
  selectedOrderMap.value = next
}

function toggleSelection(order: PaymentOrder) {
  const next = new Map(selectedOrderMap.value)
  if (next.has(order.id)) {
    next.delete(order.id)
  } else {
    next.set(order.id, order)
  }
  setSelectedOrderMap(next)
}

function toggleVisibleSelection(checked: boolean) {
  const next = new Map(selectedOrderMap.value)
  for (const order of availableOrders.value) {
    if (checked) {
      next.set(order.id, order)
    } else {
      next.delete(order.id)
    }
  }
  setSelectedOrderMap(next)
}

function clearSelection() {
  setSelectedOrderMap(new Map())
}

async function selectAllEligibleOrders() {
  selectingAll.value = true
  try {
    const res = await paymentAPI.getInvoiceAvailableOrders({
      page: 1,
      page_size: Math.max(invoiceSummary.available_order_count, availablePagination.page_size),
    })
    const next = new Map<number, PaymentOrder>()
    for (const order of res.data.items || []) {
      next.set(order.id, order)
    }
    setSelectedOrderMap(next)
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    selectingAll.value = false
  }
}

async function loadInvoiceSummary() {
  const res = await paymentAPI.getInvoiceSummary()
  invoiceSummary.available_pay_amount = res.data.available_pay_amount || 0
  invoiceSummary.available_order_count = res.data.available_order_count || 0
  invoiceSummary.minimum_pay_amount = res.data.minimum_pay_amount || 100
}

async function loadAvailableOrders() {
  loadingAvailable.value = true
  try {
    const res = await paymentAPI.getInvoiceAvailableOrders({
      page: availablePagination.page,
      page_size: availablePagination.page_size,
    })
    availableOrders.value = res.data.items || []
    availablePagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loadingAvailable.value = false
  }
}

async function loadInvoices() {
  loadingInvoices.value = true
  try {
    const res = await paymentAPI.getMyInvoices({
      page: invoicePagination.page,
      page_size: invoicePagination.page_size,
    })
    invoices.value = res.data.items || []
    invoicePagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loadingInvoices.value = false
  }
}

async function refreshCurrentTab() {
  await loadInvoiceSummary()
  if (activeTab.value === 'create') {
    await loadAvailableOrders()
    return
  }
  await loadInvoices()
}

function setActiveTab(tab: 'create' | 'history') {
  activeTab.value = tab
  router.replace({
    path: '/invoices',
    query: tab === 'history' ? { tab: 'history' } : {},
  })
}

async function submitInvoice() {
  if (!canSubmitInvoice.value) return
  invoiceSubmitting.value = true
  try {
    await paymentAPI.createInvoice({
      order_ids: selectedOrders.value.map((order) => order.id),
      title_name: invoiceForm.title_name.trim(),
      tax_id: invoiceForm.tax_id.trim(),
    })
    appStore.showSuccess(t('payment.invoice.submitSuccess'))
    showCreateDialog.value = false
    invoiceForm.title_name = ''
    invoiceForm.tax_id = ''
    clearSelection()
    await loadInvoiceSummary()
    await loadAvailableOrders()
    await loadInvoices()
    setActiveTab('history')
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    invoiceSubmitting.value = false
  }
}

async function openHistoryDetail(invoice: PaymentInvoice) {
  try {
    const res = await paymentAPI.getInvoice(invoice.id)
    historyDetail.value = res.data
  } catch {
    historyDetail.value = invoice
  }
  showHistoryDialog.value = true
}

async function downloadInvoice(invoice: PaymentInvoice) {
  downloadingInvoiceId.value = invoice.id
  try {
    const blob = await paymentAPI.downloadInvoice(invoice.id)
    const url = window.URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = invoice.file_name || `invoice-${invoice.id}.pdf`
    document.body.appendChild(anchor)
    anchor.click()
    document.body.removeChild(anchor)
    window.URL.revokeObjectURL(url)
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    downloadingInvoiceId.value = null
  }
}

function handleAvailablePageChange(page: number) {
  availablePagination.page = page
  clearSelection()
  loadAvailableOrders()
}

function handleAvailablePageSizeChange(pageSize: number) {
  availablePagination.page_size = pageSize
  availablePagination.page = 1
  clearSelection()
  loadAvailableOrders()
}

function handleInvoicePageChange(page: number) {
  invoicePagination.page = page
  loadInvoices()
}

function handleInvoicePageSizeChange(pageSize: number) {
  invoicePagination.page_size = pageSize
  invoicePagination.page = 1
  loadInvoices()
}

function invoiceStatusLabel(status: string): string {
  switch (status) {
    case 'REQUESTED':
      return t('payment.invoice.requested')
    case 'ISSUED':
      return t('payment.invoice.issued')
    case 'FAILED':
      return t('payment.invoice.failed')
    default:
      return status
  }
}

function invoiceStatusClass(status: string): string {
  switch (status) {
    case 'REQUESTED':
      return 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300'
    case 'ISSUED':
      return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300'
    case 'FAILED':
      return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300'
  }
}

function formatDateTime(value?: string): string {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

watch(
  () => route.query.tab,
  (tab) => {
    activeTab.value = tab === 'history' ? 'history' : 'create'
  },
  { immediate: true }
)

onMounted(async () => {
  try {
    await loadInvoiceSummary()
    await Promise.all([loadAvailableOrders(), loadInvoices()])
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  }
})
</script>
