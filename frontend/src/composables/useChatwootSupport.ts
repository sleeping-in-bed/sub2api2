import { onBeforeUnmount, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { getLocale } from '@/i18n'
import { userAPI, type ChatwootSupportSession } from '@/api/user'

const CHATWOOT_SCRIPT_ID = 'chatwoot-sdk-script'

let initializedBaseURL = ''
let initializedWebsiteToken = ''
let scriptLoadingPromise: Promise<void> | null = null
let syncRequestSeq = 0

function resetChatwootSession() {
  try {
    window.$chatwoot?.reset?.()
  } catch (error) {
    console.error('Failed to reset Chatwoot session:', error)
  }
}

function setChatwootLocale() {
  const locale = getLocale()
  try {
    window.chatwootSettings = {
      ...(window.chatwootSettings || {}),
      locale,
    }
    window.$chatwoot?.setLocale?.(locale)
  } catch (error) {
    console.error('Failed to update Chatwoot locale:', error)
  }
}

async function ensureChatwootSDK(session: ChatwootSupportSession): Promise<void> {
  const baseUrl = String(session.base_url || '').trim().replace(/\/$/, '')
  const websiteToken = String(session.website_token || '').trim()
  if (!baseUrl || !websiteToken) {
    return
  }

  setChatwootLocale()

  if (
    window.chatwootSDK &&
    initializedBaseURL === baseUrl &&
    initializedWebsiteToken === websiteToken
  ) {
    return
  }

  if (scriptLoadingPromise) {
    await scriptLoadingPromise
    return
  }

  scriptLoadingPromise = new Promise<void>((resolve, reject) => {
    const existing = document.getElementById(CHATWOOT_SCRIPT_ID) as HTMLScriptElement | null

    const finalize = () => {
      if (!window.chatwootSDK?.run) {
        reject(new Error('Chatwoot SDK is unavailable'))
        return
      }
      window.chatwootSDK.run({
        websiteToken,
        baseUrl,
      })
      initializedBaseURL = baseUrl
      initializedWebsiteToken = websiteToken
      resolve()
    }

    if (existing) {
      finalize()
      return
    }

    const script = document.createElement('script')
    script.id = CHATWOOT_SCRIPT_ID
    script.src = `${baseUrl}/packs/js/sdk.js`
    script.async = true
    script.defer = true
    script.onload = () => finalize()
    script.onerror = () => reject(new Error('Failed to load Chatwoot SDK'))
    document.head.appendChild(script)
  })

  try {
    await scriptLoadingPromise
  } finally {
    scriptLoadingPromise = null
  }
}

async function waitForChatwootBridge(): Promise<ChatwootSessionLike | null> {
  for (let attempt = 0; attempt < 50; attempt += 1) {
    if (window.$chatwoot?.setUser) {
      return window.$chatwoot
    }
    await new Promise((resolve) => setTimeout(resolve, 200))
  }
  return window.$chatwoot || null
}

async function syncChatwootUser(authStore: ReturnType<typeof useAuthStore>) {
  if (!authStore.isAuthenticated) {
    resetChatwootSession()
    return
  }

  const requestSeq = ++syncRequestSeq
  const session = await userAPI.getChatwootSupportSession()
  if (requestSeq !== syncRequestSeq) {
    return
  }
  if (!session.enabled) {
    resetChatwootSession()
    return
  }

  await ensureChatwootSDK(session)
  const bridge = await waitForChatwootBridge()
  if (!bridge?.setUser || !session.identifier) {
    return
  }

  setChatwootLocale()
  bridge.setUser(session.identifier, {
    email: session.email,
    name: session.name,
    avatar_url: session.avatar_url,
    identifier_hash: session.identifier_hash,
  })

  if (session.custom_attributes && bridge.setCustomAttributes) {
    bridge.setCustomAttributes(session.custom_attributes)
  }
}

export function useChatwootSupport() {
  const authStore = useAuthStore()

  const stop = watch(
    () => [authStore.isAuthenticated, authStore.user?.id, authStore.user?.email, authStore.user?.username] as const,
    () => {
      syncChatwootUser(authStore).catch((error) => {
        console.error('Failed to sync Chatwoot support widget:', error)
      })
    },
    { immediate: true }
  )

  const handleVisibilityChange = () => {
    if (document.visibilityState === 'visible' && authStore.isAuthenticated) {
      syncChatwootUser(authStore).catch((error) => {
        console.error('Failed to refresh Chatwoot support widget:', error)
      })
    }
  }

  document.addEventListener('visibilitychange', handleVisibilityChange)

  onBeforeUnmount(() => {
    stop()
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  })
}
