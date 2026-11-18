const API_BASE = '/admin/api'
const KEY_STORAGE = 'ollarouter.adminKey'

// When api_keys are configured, the admin API needs "Authorization: Bearer <key>".
// The key is kept in sessionStorage only, so it is gone when the tab closes.
function getAdminKey(): string | null {
  try {
    return sessionStorage.getItem(KEY_STORAGE)
  } catch {
    return null
  }
}

function setAdminKey(key: string | null) {
  try {
    if (key) sessionStorage.setItem(KEY_STORAGE, key)
    else sessionStorage.removeItem(KEY_STORAGE)
  } catch {
    // Storage unavailable (private mode): the key lives for this request only.
  }
}

function authHeaders(): Record<string, string> {
  const key = getAdminKey()
  return key ? { Authorization: `Bearer ${key}` } : {}
}

// One prompt is shared by requests that hit 401 at the same time.
let pendingPrompt: Promise<string | null> | null = null

function askForKey(): Promise<string | null> {
  if (!pendingPrompt) {
    pendingPrompt = Promise.resolve(
      window.prompt('OllaRouter admin: enter an API key from providers.yaml'),
    ).finally(() => {
      setTimeout(() => {
        pendingPrompt = null
      }, 0)
    })
  }
  return pendingPrompt
}

// fetchWithAuth adds the stored key and, on a 401, asks for a key once and retries.
async function fetchWithAuth(url: string, options?: RequestInit): Promise<Response> {
  const send = () =>
    fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...authHeaders(),
        ...(options?.headers as Record<string, string> | undefined),
      },
    })

  let resp = await send()
  if (resp.status === 401) {
    const entered = await askForKey()
    if (entered) {
      setAdminKey(entered.trim())
      resp = await send()
      if (resp.status === 401) setAdminKey(null)
    }
  }
  return resp
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const resp = await fetchWithAuth(`${API_BASE}${path}`, options)
  if (!resp.ok) {
    const error = await resp.json().catch(() => ({ error: resp.statusText }))
    throw new Error(error.error || `HTTP ${resp.status}`)
  }
  return resp.json()
}

export interface Provider {
  name: string
  type: string
  base_url: string
  status: string
  total_cost: number
  models: string[]
  models_count: number
  auto_discover: boolean
  supports_balance?: boolean
  circuit_breaker: {
    state: string
    failures: number
  }
  pricing?: {
    input_per_1k: number
    output_per_1k: number
  }
}

export interface ProviderListResponse {
  providers: Provider[]
}

export interface StatsSummary {
  total_requests: number
  total_cost: number
  total_tokens: number
  providers_active: number
}

export interface UsageRecord {
  timestamp: string
  provider: string
  model: string
  request_type: string
  status: number
  latency_ms: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  cost_usd: number
}

export interface HealthResponse {
  status: string
  version: string
  uptime_seconds: number
  providers_total: number
  providers_online: number
}

export interface ModelCost {
  model: string
  requests: number
  total_tokens: number
  total_cost: number
}

export interface CostByModelResponse {
  cost_by_model: Record<string, ModelCost[]>
}

export interface ProviderBalance {
  currency?: string
  total_credits?: number
  total_usage?: number
  remaining_credits?: number
  source?: string
}

export const api = {
  // Providers
  listProviders: () => request<ProviderListResponse>('/providers'),
  addProvider: (data: any) => request<{ status: string; name: string }>('/providers', { method: 'POST', body: JSON.stringify(data) }),
  updateProvider: (name: string, data: any) => request<{ status: string; name: string }>(`/providers/${name}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteProvider: (name: string) => request<{ status: string; name: string }>(`/providers/${name}`, { method: 'DELETE' }),
  testProvider: (name: string) => request<{ status: string; latency_ms: number; error?: string }>(`/providers/${name}/test`),
  getProviderBalance: (name: string) => request<{ provider: string; type: string; balance: ProviderBalance }>(`/providers/${name}/balance`),
  discoverProviderModels: (data: any) => request<{ models: string[] }>('/providers/discover', { method: 'POST', body: JSON.stringify(data) }),

  // Stats
  getStatsSummary: () => request<StatsSummary>('/stats/summary'),
  getCostByProvider: () => request<{ cost_by_provider: Record<string, number> }>('/stats/cost-by-provider'),
  getCostByModel: () => request<CostByModelResponse>('/stats/cost-by-model'),

  // Logs
  getLogs: (provider?: string, limit = 100) => {
    const params = new URLSearchParams()
    if (provider) params.set('provider', provider)
    params.set('limit', String(limit))
    return request<{ logs: UsageRecord[] }>(`/logs?${params.toString()}`)
  },

  // Settings
  getSettings: () => request<any>('/settings'),
  updateSettings: (data: any) => request<any>('/settings', { method: 'PUT', body: JSON.stringify(data) }),
  getConfigYAML: () => fetchWithAuth(`${API_BASE}/config/yaml`).then(r => r.text()),

  // API Keys
  listApiKeys: () => request<{ api_keys: any[] }>('/api-keys'),
  createApiKey: (data: any) => request<{ key: string; name: string; warning: string }>('/api-keys', { method: 'POST', body: JSON.stringify(data) }),
  revokeApiKey: (id: string) => request<{ status: string }>(`/api-keys/${id}`, { method: 'DELETE' }),

  // Health
  getHealth: () => fetch('/health').then(r => r.json() as Promise<HealthResponse>),
}
