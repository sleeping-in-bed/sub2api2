import type { PublicSettings } from '@/types'

declare global {
  interface ChatwootUserPayload {
    email?: string
    name?: string
    avatar_url?: string
    identifier_hash?: string
  }

  interface ChatwootSessionLike {
    setUser?: (identifier: string, user?: ChatwootUserPayload) => void
    setCustomAttributes?: (attributes: Record<string, unknown>) => void
    setLocale?: (locale: string) => void
    reset?: () => void
    toggle?: (state?: 'open' | 'close') => void
  }

  interface ChatwootSDKLike {
    run: (options: { websiteToken: string; baseUrl: string }) => void
  }

  interface Window {
    __APP_CONFIG__?: PublicSettings
    chatwootSettings?: Record<string, unknown>
    chatwootSDK?: ChatwootSDKLike
    $chatwoot?: ChatwootSessionLike
  }
}

export {}
