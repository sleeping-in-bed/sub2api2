<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex-1 sm:max-w-72">
            <input
              v-model="filters.keyword"
              type="text"
              :placeholder="t('payment.invoice.admin.searchPlaceholder')"
              class="input"
              @input="debounceLoadInvoices"
            />
          </div>
          <Select v-model="filters.status" :options="statusOptions" class="w-40" @change="loadInvoices" />
          <div class="flex flex-1 justify-end">
            <button @click="loadInvoices" :disabled="loading" class="btn btn-secondary" :title="t('common.refresh')">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </div>

      <DataTable :columns="columns" :data="invoices" :loading="loading">
        <template #cell-order="{ row }">
          <div class="text-sm">
            <div class="font-medium text-gray-900 dark:text-white">{{ row.order?.out_trade_no || '-' }}</div>
            <div class="text-xs text-gray-500 dark:text-gray-400">#{{ row.order_id }}</div>
          </div>
        </template>
        <template #cell-user="{ row }">
          <div class="text-sm">
            <div class="text-gray-900 dark:text-white">{{ row.order?.user_email || row.order?.user_name || '#' + row.user_id }}</div>
            <div v-if="row.order?.user_notes" class="text-xs text-gray-500 dark:text-gray-400">{{ row.order.user_notes }}</div>
          </div>
        </template>
        <template #cell-title_name="{ row }">
          <div class="text-sm text-gray-900 dark:text-white">{{ row.title_name }}</div>
        </template>
        <template #cell-tax_id="{ row }">
          <div class="font-mono text-sm text-gray-700 dark:text-gray-300">{{ row.tax_id }}</div>
        </template>
        <template #cell-status="{ row }">
          <span class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium" :class="invoiceStatusClass(row.status)">
            {{ invoiceStatusLabel(row.status) }}
          </span>
        </template>
        <template #cell-requested_at="{ value }">
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(value) }}</span>
        </template>
        <template #cell-actions="{ row }">
          <div class="flex items-center gap-1">
            <button @click="openDetailDialog(row)" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-600">
              <Icon name="eye" size="sm" />
              {{ t('common.view') }}
            </button>
            <button
              v-if="row.status !== 'ISSUED'"
              @click="openIssueDialog(row)"
              class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-blue-600 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/20"
            >
              <Icon name="upload" size="sm" />
              {{ t('payment.invoice.admin.issue') }}
            </button>
            <button
              v-if="row.status !== 'ISSUED'"
              @click="openFailDialog(row)"
              class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
            >
              <Icon name="exclamationTriangle" size="sm" />
              {{ t('payment.invoice.admin.markFailed') }}
            </button>
            <button
              v-if="row.status === 'ISSUED'"
              :disabled="downloadingInvoiceId === row.id"
              @click="downloadInvoice(row)"
              class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-green-600 hover:bg-green-50 disabled:cursor-not-allowed disabled:opacity-60 dark:text-green-400 dark:hover:bg-green-900/20"
            >
              <Icon name="download" size="sm" />
              {{ downloadingInvoiceId === row.id ? t('common.processing') : t('payment.invoice.download') }}
            </button>
          </div>
        </template>
      </DataTable>

      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>

    <BaseDialog :show="showDetailDialog" :title="t('payment.invoice.admin.detailTitle')" width="wide" @close="showDetailDialog = false">
      <div v-if="selectedInvoice" class="space-y-4">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.titleName') }}</p>
            <p class="text-sm text-gray-900 dark:text-white">{{ selectedInvoice.title_name }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.taxId') }}</p>
            <p class="font-mono text-sm text-gray-900 dark:text-white">{{ selectedInvoice.tax_id }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.statusLabel') }}</p>
            <p class="text-sm text-gray-900 dark:text-white">{{ invoiceStatusLabel(selectedInvoice.status) }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.requestedAt') }}</p>
            <p class="text-sm text-gray-900 dark:text-white">{{ formatDateTime(selectedInvoice.requested_at) }}</p>
          </div>
          <div v-if="selectedInvoice.issued_at">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.issuedAt') }}</p>
            <p class="text-sm text-gray-900 dark:text-white">{{ formatDateTime(selectedInvoice.issued_at) }}</p>
          </div>
          <div v-if="selectedInvoice.file_name">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.fileName') }}</p>
            <p class="text-sm text-gray-900 dark:text-white">{{ selectedInvoice.file_name }}</p>
          </div>
          <div v-if="selectedInvoice.failed_reason" class="sm:col-span-2">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.failedReason') }}</p>
            <p class="text-sm text-red-600 dark:text-red-400">{{ selectedInvoice.failed_reason }}</p>
          </div>
        </div>

        <div v-if="selectedInvoice.order" class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <p class="mb-3 text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('payment.invoice.admin.orderInfo') }}</p>
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.orders.orderNo') }}</p>
              <p class="text-sm text-gray-900 dark:text-white">{{ selectedInvoice.order.out_trade_no }}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.admin.colUser') }}</p>
              <p class="text-sm text-gray-900 dark:text-white">{{ selectedInvoice.order.user_email || selectedInvoice.order.user_name }}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.orders.amount') }}</p>
              <p class="text-sm text-gray-900 dark:text-white">{{ selectedInvoice.order.order_type === 'balance' ? '$' : '¥' }}{{ selectedInvoice.order.amount.toFixed(2) }}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.orders.payAmount') }}</p>
              <p class="text-sm text-gray-900 dark:text-white">¥{{ selectedInvoice.order.pay_amount.toFixed(2) }}</p>
            </div>
          </div>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog :show="!!issueTarget" :title="t('payment.invoice.admin.issueTitle')" @close="closeIssueDialog">
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('payment.invoice.admin.issueHint') }}</p>
        <input type="file" accept="application/pdf" class="input w-full" @change="handleIssueFileChange" />
        <p v-if="issueFile" class="text-sm text-gray-500 dark:text-gray-400">{{ issueFile.name }}</p>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="closeIssueDialog">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="issueSubmitting || !issueFile" @click="submitIssue">
            {{ issueSubmitting ? t('common.processing') : t('payment.invoice.admin.issue') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="!!failTarget" :title="t('payment.invoice.admin.failTitle')" @close="closeFailDialog">
      <div class="space-y-4">
        <label class="input-label">{{ t('payment.invoice.failedReason') }}</label>
        <textarea v-model="failReason" rows="4" class="input w-full" :placeholder="t('payment.invoice.admin.failPlaceholder')" />
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="closeFailDialog">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" :disabled="failSubmitting || !failReason.trim()" @click="submitFail">
            {{ failSubmitting ? t('common.processing') : t('payment.invoice.admin.markFailed') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Column } from '@/components/common/types'
import type { AdminPaymentInvoice } from '@/types/payment'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI } from '@/api/admin/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const issueSubmitting = ref(false)
const failSubmitting = ref(false)
const downloadingInvoiceId = ref<number | null>(null)
const invoices = ref<AdminPaymentInvoice[]>([])
const selectedInvoice = ref<AdminPaymentInvoice | null>(null)
const issueTarget = ref<AdminPaymentInvoice | null>(null)
const failTarget = ref<AdminPaymentInvoice | null>(null)
const issueFile = ref<File | null>(null)
const failReason = ref('')
const showDetailDialog = ref(false)

const filters = reactive({
  keyword: '',
  status: '',
})

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
})

let debounceTimer: ReturnType<typeof setTimeout> | null = null

const columns = computed((): Column[] => [
  { key: 'order', label: t('payment.invoice.admin.orderColumn') },
  { key: 'user', label: t('payment.admin.colUser') },
  { key: 'title_name', label: t('payment.invoice.titleName') },
  { key: 'tax_id', label: t('payment.invoice.taxId') },
  { key: 'status', label: t('payment.invoice.statusLabel') },
  { key: 'requested_at', label: t('payment.invoice.requestedAt') },
  { key: 'actions', label: t('common.actions') },
])

const statusOptions = computed(() => [
  { value: '', label: t('payment.invoice.admin.allStatuses') },
  { value: 'REQUESTED', label: t('payment.invoice.requested') },
  { value: 'ISSUED', label: t('payment.invoice.issued') },
  { value: 'FAILED', label: t('payment.invoice.failed') },
])

function debounceLoadInvoices() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    pagination.page = 1
    loadInvoices()
  }, 300)
}

async function loadInvoices() {
  loading.value = true
  try {
    const res = await adminPaymentAPI.getInvoices({
      page: pagination.page,
      page_size: pagination.page_size,
      status: filters.status || undefined,
      keyword: filters.keyword || undefined,
    })
    invoices.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadInvoices()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  loadInvoices()
}

async function openDetailDialog(invoice: AdminPaymentInvoice) {
  selectedInvoice.value = invoice
  showDetailDialog.value = true
  try {
    const res = await adminPaymentAPI.getInvoice(invoice.id)
    selectedInvoice.value = res.data
  } catch {
    // Keep list data as fallback.
  }
}

function openIssueDialog(invoice: AdminPaymentInvoice) {
  issueTarget.value = invoice
  issueFile.value = null
}

function closeIssueDialog() {
  issueTarget.value = null
  issueFile.value = null
}

function handleIssueFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  issueFile.value = target.files?.[0] || null
}

async function submitIssue() {
  if (!issueTarget.value || !issueFile.value) return
  issueSubmitting.value = true
  try {
    await adminPaymentAPI.issueInvoice(issueTarget.value.id, issueFile.value)
    appStore.showSuccess(t('payment.invoice.admin.issueSuccess'))
    closeIssueDialog()
    await loadInvoices()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    issueSubmitting.value = false
  }
}

function openFailDialog(invoice: AdminPaymentInvoice) {
  failTarget.value = invoice
  failReason.value = invoice.failed_reason || ''
}

function closeFailDialog() {
  failTarget.value = null
  failReason.value = ''
}

async function submitFail() {
  if (!failTarget.value || !failReason.value.trim()) return
  failSubmitting.value = true
  try {
    await adminPaymentAPI.failInvoice(failTarget.value.id, { reason: failReason.value.trim() })
    appStore.showSuccess(t('payment.invoice.admin.failSuccess'))
    closeFailDialog()
    await loadInvoices()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    failSubmitting.value = false
  }
}

async function downloadInvoice(invoice: AdminPaymentInvoice) {
  downloadingInvoiceId.value = invoice.id
  try {
    const blob = await adminPaymentAPI.downloadInvoice(invoice.id)
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

onMounted(() => {
  loadInvoices()
})
</script>
