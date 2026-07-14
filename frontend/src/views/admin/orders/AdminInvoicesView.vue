<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="flex flex-wrap items-center gap-3 border-b border-gray-200 pb-4 dark:border-dark-600">
        <input v-model="keyword" class="input max-w-72" :placeholder="t('payment.invoices.search')" @keyup.enter="loadInvoices" />
        <button class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="loadInvoices">
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>
      </div>
      <div class="overflow-x-auto border border-gray-200 dark:border-dark-600">
        <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
          <thead class="bg-gray-50 dark:bg-dark-800">
            <tr>
              <th class="px-4 py-3 text-left">{{ t('payment.orders.orderId') }}</th>
              <th class="px-4 py-3 text-left">{{ t('payment.invoices.titleName') }}</th>
              <th class="px-4 py-3 text-left">{{ t('payment.invoices.taxId') }}</th>
              <th class="px-4 py-3 text-right">{{ t('payment.invoices.amount') }}</th>
              <th class="px-4 py-3 text-left">{{ t('payment.orders.status') }}</th>
              <th class="px-4 py-3 text-right">{{ t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-600 dark:bg-dark-900">
            <tr v-for="invoice in invoices" :key="invoice.id">
              <td class="px-4 py-3 font-mono">
                <div v-for="order in invoice.orders" :key="order.id" class="whitespace-nowrap">{{ order.order_uuid }}</div>
              </td>
              <td class="px-4 py-3">{{ invoice.title_name }}</td>
              <td class="px-4 py-3 font-mono">{{ invoice.tax_id }}</td>
              <td class="px-4 py-3 text-right">{{ invoice.total_pay_amount?.toFixed(2) || '-' }}</td>
              <td class="px-4 py-3">{{ t(`payment.invoices.status.${invoice.status}`) }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-end gap-2">
                  <label v-if="invoice.status !== 'ISSUED'" class="btn btn-secondary cursor-pointer" :title="t('payment.invoices.upload')">
                    <Icon name="upload" size="md" />
                    <input type="file" accept="application/pdf" class="hidden" @change="issueInvoice(invoice, $event)" />
                  </label>
                  <button v-if="invoice.status !== 'ISSUED'" class="btn btn-danger" :title="t('payment.invoices.fail')" @click="openFailure(invoice)">
                    <Icon name="x" size="md" />
                  </button>
                  <button v-if="invoice.status === 'ISSUED'" class="btn btn-secondary" :title="t('common.download')" @click="download(invoice)">
                    <Icon name="download" size="md" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <BaseDialog :show="failureTarget !== null" :title="t('payment.invoices.fail')" width="narrow" @close="failureTarget = null">
      <textarea v-model="failureReason" class="input w-full" rows="4" :placeholder="t('payment.invoices.failureReason')" />
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn btn-secondary" @click="failureTarget = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" :disabled="!failureReason.trim()" @click="failInvoice">{{ t('common.confirm') }}</button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminPaymentAPI } from '@/api/admin/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import type { PaymentInvoice } from '@/types/payment'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const keyword = ref('')
const invoices = ref<PaymentInvoice[]>([])
const failureTarget = ref<PaymentInvoice | null>(null)
const failureReason = ref('')

async function loadInvoices() {
  loading.value = true
  try {
    const response = await adminPaymentAPI.getInvoices({ page: 1, page_size: 100, keyword: keyword.value.trim() || undefined })
    invoices.value = response.data.items || []
  } catch (error) {
    appStore.showError((error as Error).message)
  } finally {
    loading.value = false
  }
}

async function issueInvoice(invoice: PaymentInvoice, event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    await adminPaymentAPI.issueInvoice(invoice.id, file)
    appStore.showSuccess(t('payment.invoices.issued'))
    await loadInvoices()
  } catch (error) {
    appStore.showError((error as Error).message)
  } finally {
    input.value = ''
  }
}

function openFailure(invoice: PaymentInvoice) {
  failureTarget.value = invoice
  failureReason.value = ''
}

async function failInvoice() {
  if (!failureTarget.value) return
  try {
    await adminPaymentAPI.failInvoice(failureTarget.value.id, failureReason.value.trim())
    failureTarget.value = null
    await loadInvoices()
  } catch (error) {
    appStore.showError((error as Error).message)
  }
}

async function download(invoice: PaymentInvoice) {
  const blob = await adminPaymentAPI.downloadInvoice(invoice.id)
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = invoice.file_name || `invoice-${invoice.id}.pdf`
  link.click()
  URL.revokeObjectURL(url)
}

onMounted(loadInvoices)
</script>
