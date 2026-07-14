import { describe, expect, it } from 'vitest'

import enLanding from '@/i18n/locales/en/landing'
import zhLanding from '@/i18n/locales/zh/landing'

describe('MindAI landing page copy', () => {
  it('keeps the Chinese team-oriented product message', () => {
    expect(zhLanding.home.tags).toEqual({
      subscriptionToApi: '统一接入',
      stickySession: '用量透明',
      realtimeBilling: '额度可控',
    })
    expect(zhLanding.home.features).toEqual({
      unifiedGateway: '统一接入',
      unifiedGatewayDesc: '一个 API 密钥接入已开通模型，减少多平台重复配置。',
      multiAccount: '用量透明',
      multiAccountDesc: '团队成员的调用记录、费用和消耗清晰可查。',
      balanceQuota: '额度可控',
      balanceQuotaDesc: '支持按团队或账号设置额度，避免超预算和失控使用。',
    })
    expect(zhLanding.home.providers.title).toBe('已支持的 AI 能力')
  })

  it('keeps the English team-oriented product message', () => {
    expect(enLanding.home.tags).toEqual({
      subscriptionToApi: 'Unified Access',
      stickySession: 'Transparent Usage',
      realtimeBilling: 'Controlled Quotas',
    })
    expect(enLanding.home.features).toEqual({
      unifiedGateway: 'Unified Access',
      unifiedGatewayDesc: 'Use one API key to reach enabled models and avoid repetitive multi-platform setup.',
      multiAccount: 'Transparent Usage',
      multiAccountDesc: 'Keep team requests, costs, and consumption visible in one place.',
      balanceQuota: 'Controlled Quotas',
      balanceQuotaDesc: 'Set quotas by team or account to prevent overspending and uncontrolled usage.',
    })
    expect(enLanding.home.providers.title).toBe('Supported AI Capabilities')
  })
})
