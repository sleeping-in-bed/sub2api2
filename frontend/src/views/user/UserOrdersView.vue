<template>
  <AppLayout>
    <div class="space-y-4">
      <!-- Filters -->
      <div class="card p-4">
        <div class="flex flex-wrap items-center gap-3">
          <Select v-model="currentFilter" :options="statusFilters" class="w-36" @change="fetchOrders" />
          <div class="flex flex-1 items-center justify-end gap-2">
            <button @click="fetchOrders" :disabled="loading" class="btn btn-secondary" :title="t('common.refresh')">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button class="btn btn-primary" @click="router.push('/purchase')">{{ t('payment.result.backToRecharge') }}</button>
          </div>
        </div>
      </div>

      <!-- Table -->
      <OrderTable :orders="orders" :loading="loading">
        <template #actions="{ row }">
          <div class="flex items-center gap-2">
            <button v-if="row.status === 'PENDING'" @click="handleCancel(row.id)" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-yellow-600 hover:bg-yellow-50 dark:text-yellow-400 dark:hover:bg-yellow-900/20">
              <Icon name="x" size="sm" />
              <span>{{ t('payment.orders.cancel') }}</span>
            </button>
            <button v-if="canRequestRefund(row)" @click="openRefundDialog(row)" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-purple-600 hover:bg-purple-50 dark:text-purple-400 dark:hover:bg-purple-900/20">
              <Icon name="dollar" size="sm" />
              <span>{{ t('payment.orders.requestRefund') }}</span>
            </button>
            <button
              v-if="row.status === 'COMPLETED' && !row.invoice"
              @click="openInvoiceDialog(row)"
              class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-blue-600 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/20"
            >
              <Icon name="document" size="sm" />
              <span>{{ t('payment.invoice.request') }}</span>
            </button>
            <button
              v-else-if="row.invoice?.status === 'REQUESTED'"
              @click="openInvoiceDialog(row)"
              class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-amber-600 hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/20"
            >
              <Icon name="clock" size="sm" />
              <span>{{ t('payment.invoice.requested') }}</span>
            </button>
            <button
              v-else-if="row.invoice?.status === 'ISSUED'"
              :disabled="downloadingInvoiceId === row.invoice.id"
              @click="downloadInvoice(row)"
              class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-green-600 hover:bg-green-50 disabled:cursor-not-allowed disabled:opacity-60 dark:text-green-400 dark:hover:bg-green-900/20"
            >
              <Icon name="download" size="sm" />
              <span>{{ downloadingInvoiceId === row.invoice.id ? t('common.processing') : t('payment.invoice.download') }}</span>
            </button>
            <button
              v-else-if="row.invoice?.status === 'FAILED'"
              @click="openInvoiceDialog(row)"
              class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
            >
              <Icon name="exclamationTriangle" size="sm" />
              <span>{{ t('payment.invoice.failed') }}</span>
            </button>
          </div>
        </template>
      </OrderTable>

      <!-- Pagination -->
      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>

    <!-- Cancel Confirm Dialog -->
    <BaseDialog :show="!!cancelTargetId" :title="t('payment.orders.cancel')" width="narrow" @close="cancelTargetId = null">
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('payment.confirmCancel') }}</p>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="cancelTargetId = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" :disabled="actionLoading" @click="confirmCancel">{{ actionLoading ? t('common.processing') : t('payment.orders.cancel') }}</button>
        </div>
      </template>
    </BaseDialog>

    <!-- Refund Dialog -->
    <BaseDialog :show="!!refundTarget" :title="t('payment.orders.requestRefund')" @close="refundTarget = null">
      <div v-if="refundTarget" class="space-y-4">
        <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
          <div class="flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.orders.orderId') }}</span>
            <span class="font-mono text-gray-900 dark:text-white">#{{ refundTarget.id }}</span>
          </div>
          <div class="mt-2 flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.orders.amount') }}</span>
            <span class="text-gray-900 dark:text-white">${{ refundTarget.amount.toFixed(2) }}</span>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('payment.refundReason') }}</label>
          <textarea v-model="refundReason" rows="3" class="input mt-1 w-full" :placeholder="t('payment.refundReasonPlaceholder')" />
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="refundTarget = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="actionLoading || !refundReason.trim()" @click="confirmRefund">{{ actionLoading ? t('common.processing') : t('payment.orders.requestRefund') }}</button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="!!invoiceTarget" :title="t('payment.invoice.title')" @close="closeInvoiceDialog">
      <div v-if="invoiceTarget" class="space-y-4">
        <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
          <div class="flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.orders.orderId') }}</span>
            <span class="font-mono text-gray-900 dark:text-white">#{{ invoiceTarget.id }}</span>
          </div>
          <div class="mt-2 flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.orders.orderNo') }}</span>
            <span class="text-gray-900 dark:text-white">{{ invoiceTarget.out_trade_no }}</span>
          </div>
          <div class="mt-2 flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.orders.amount') }}</span>
            <span class="text-gray-900 dark:text-white">{{ invoiceTarget.order_type === 'balance' ? '$' : '¥' }}{{ invoiceTarget.amount.toFixed(2) }}</span>
          </div>
        </div>

        <template v-if="!invoiceTarget.invoice">
          <div>
            <label class="input-label">{{ t('payment.invoice.titleName') }}</label>
            <input v-model="invoiceForm.title_name" type="text" class="input mt-1 w-full" :placeholder="t('payment.invoice.titleNamePlaceholder')" />
          </div>
          <div>
            <label class="input-label">{{ t('payment.invoice.taxId') }}</label>
            <input v-model="invoiceForm.tax_id" type="text" class="input mt-1 w-full" :placeholder="t('payment.invoice.taxIdPlaceholder')" />
          </div>
        </template>

        <template v-else>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.titleName') }}</div>
              <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ invoiceTarget.invoice.title_name }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.taxId') }}</div>
              <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ invoiceTarget.invoice.tax_id }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.statusLabel') }}</div>
              <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ invoiceStatusLabel(invoiceTarget.invoice.status) }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.requestedAt') }}</div>
              <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ formatDateTime(invoiceTarget.invoice.requested_at) }}</div>
            </div>
          </div>
          <p v-if="invoiceTarget.invoice.status === 'REQUESTED'" class="rounded-xl bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
            {{ t('payment.invoice.requestedHint') }}
          </p>
          <p v-if="invoiceTarget.invoice.status === 'FAILED' && invoiceTarget.invoice.failed_reason" class="rounded-xl bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">
            {{ t('payment.invoice.failedReason') }}: {{ invoiceTarget.invoice.failed_reason }}
          </p>
          <p v-if="invoiceTarget.invoice.status === 'ISSUED'" class="rounded-xl bg-green-50 px-4 py-3 text-sm text-green-700 dark:bg-green-900/20 dark:text-green-300">
            {{ t('payment.invoice.issuedHint') }}
          </p>
        </template>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="closeInvoiceDialog">{{ t('common.close') }}</button>
          <button
            v-if="invoiceTarget && !invoiceTarget.invoice"
            class="btn btn-primary"
            :disabled="invoiceSubmitting || !invoiceForm.title_name.trim() || !invoiceForm.tax_id.trim()"
            @click="submitInvoice"
          >
            {{ invoiceSubmitting ? t('common.processing') : t('payment.invoice.submit') }}
          </button>
          <button
            v-else-if="invoiceTarget?.invoice?.status === 'ISSUED'"
            class="btn btn-primary"
            :disabled="invoiceSubmitting"
            @click="downloadInvoice(invoiceTarget)"
          >
            {{ invoiceSubmitting ? t('common.processing') : t('payment.invoice.download') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores'
import { paymentAPI } from '@/api/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { PaymentOrder } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderTable from '@/components/payment/OrderTable.vue'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const loading = ref(false)
const actionLoading = ref(false)
const invoiceSubmitting = ref(false)
const downloadingInvoiceId = ref<number | null>(null)
const orders = ref<PaymentOrder[]>([])
const refundEligibleProviders = ref<Set<string>>(new Set())
const currentFilter = ref('')
const cancelTargetId = ref<number | null>(null)
const refundTarget = ref<PaymentOrder | null>(null)
const invoiceTarget = ref<PaymentOrder | null>(null)
const refundReason = ref('')
const invoiceForm = reactive({
  title_name: '',
  tax_id: '',
})
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const statusFilters = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'PENDING', label: t('payment.status.pending') },
  { value: 'COMPLETED', label: t('payment.status.completed') },
  { value: 'FAILED', label: t('payment.status.failed') },
  { value: 'REFUNDED', label: t('payment.status.refunded') },
])

async function fetchOrders() {
  loading.value = true
  try {
    const res = await paymentAPI.getMyOrders({
      page: pagination.page,
      page_size: pagination.page_size,
      status: currentFilter.value || undefined,
    })
    orders.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) { pagination.page = page; fetchOrders() }
function handlePageSizeChange(size: number) { pagination.page_size = size; pagination.page = 1; fetchOrders() }

function handleCancel(orderId: number) { cancelTargetId.value = orderId }

async function confirmCancel() {
  if (!cancelTargetId.value) return
  actionLoading.value = true
  try {
    await paymentAPI.cancelOrder(cancelTargetId.value)
    appStore.showSuccess(t('common.success'))
    cancelTargetId.value = null
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function openRefundDialog(order: PaymentOrder) { refundTarget.value = order; refundReason.value = '' }

function openInvoiceDialog(order: PaymentOrder) {
  invoiceTarget.value = order
  invoiceForm.title_name = order.invoice?.title_name || ''
  invoiceForm.tax_id = order.invoice?.tax_id || ''
}

function closeInvoiceDialog() {
  invoiceTarget.value = null
  invoiceForm.title_name = ''
  invoiceForm.tax_id = ''
}

async function confirmRefund() {
  if (!refundTarget.value || !refundReason.value.trim()) return
  actionLoading.value = true
  try {
    await paymentAPI.requestRefund(refundTarget.value.id, { reason: refundReason.value.trim() })
    appStore.showSuccess(t('common.success'))
    refundTarget.value = null
    refundReason.value = ''
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function canRequestRefund(order: PaymentOrder): boolean {
  if (order.status !== 'COMPLETED') return false
  if (!order.provider_instance_id) return false
  return refundEligibleProviders.value.has(order.provider_instance_id)
}

async function submitInvoice() {
  if (!invoiceTarget.value) return
  invoiceSubmitting.value = true
  try {
    await paymentAPI.createInvoice(invoiceTarget.value.id, {
      title_name: invoiceForm.title_name.trim(),
      tax_id: invoiceForm.tax_id.trim(),
    })
    appStore.showSuccess(t('payment.invoice.submitSuccess'))
    closeInvoiceDialog()
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    invoiceSubmitting.value = false
  }
}

async function downloadInvoice(order: PaymentOrder) {
  const invoice = order.invoice
  if (!invoice) return
  downloadingInvoiceId.value = invoice.id
  if (invoiceTarget.value?.invoice?.id === invoice.id) {
    invoiceSubmitting.value = true
  }
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
    invoiceSubmitting.value = false
  }
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

function formatDateTime(value?: string): string {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

async function loadRefundEligibility() {
  try {
    const res = await paymentAPI.getRefundEligibleProviders()
    refundEligibleProviders.value = new Set(res.data.provider_instance_ids || [])
  } catch { /* ignore — default to hiding refund button */ }
}

onMounted(() => { fetchOrders(); loadRefundEligibility() })
</script>
