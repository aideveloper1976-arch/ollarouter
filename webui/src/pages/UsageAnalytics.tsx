import { useEffect, useState } from 'react'
import { api, type CostByModelResponse, type ProviderBalance, type ProviderListResponse } from '../api/client'

function formatNumber(n: number): string {
  return n.toLocaleString('en-US')
}

type BalanceState = { balance?: ProviderBalance; error?: string; loading: boolean }

export default function UsageAnalytics() {
  const [data, setData] = useState<CostByModelResponse | null>(null)
  const [providersData, setProvidersData] = useState<ProviderListResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [balances, setBalances] = useState<Record<string, BalanceState>>({})

  useEffect(() => {
    async function load() {
      try {
        const [usageResult, providerResult] = await Promise.all([
          api.getCostByModel(),
          api.listProviders(),
        ])
        setData(usageResult)
        setProvidersData(providerResult)
      } catch (err) {
        console.error('Failed to load usage data:', err)
      } finally {
        setLoading(false)
      }
    }
    load()
    const interval = setInterval(load, 30000)
    return () => clearInterval(interval)
  }, [])

  // Balances hit live billing endpoints, so fetch them once (not on the
  // 30s usage-refresh interval) and let the user refresh individually.
  useEffect(() => {
    const balanceCapable = providersData?.providers.filter((p) => p.supports_balance) ?? []
    if (balanceCapable.length === 0) return
    for (const p of balanceCapable) {
      if (balances[p.name]) continue
      loadBalance(p.name)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [providersData])

  async function loadBalance(name: string) {
    setBalances((prev) => ({ ...prev, [name]: { ...prev[name], loading: true } }))
    try {
      const result = await api.getProviderBalance(name)
      setBalances((prev) => ({ ...prev, [name]: { balance: result.balance, loading: false } }))
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch balance'
      setBalances((prev) => ({ ...prev, [name]: { error: message, loading: false } }))
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-400">Loading usage data...</div>
      </div>
    )
  }

  const providers = data?.cost_by_model ?? {}
  const configuredProviders = providersData?.providers.map((provider) => provider.name) ?? []
  const providerNames = Array.from(new Set([...configuredProviders, ...Object.keys(providers)])).sort()

  // Compute totals across all models
  let totalRequests = 0
  let totalTokens = 0
  let totalCost = 0
  for (const models of Object.values(providers)) {
    for (const m of models) {
      totalRequests += m.requests
      totalTokens += m.total_tokens
      totalCost += m.total_cost
    }
  }

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Usage Analytics</h2>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
          <div className="text-3xl font-bold text-sky-400">{formatNumber(totalRequests)}</div>
          <div className="text-sm text-slate-400 mt-1">Total Requests</div>
        </div>
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
          <div className="text-3xl font-bold text-sky-400">{formatNumber(totalTokens)}</div>
          <div className="text-sm text-slate-400 mt-1">Total Tokens</div>
        </div>
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
          <div className="text-3xl font-bold text-sky-400">${totalCost.toFixed(2)}</div>
          <div className="text-sm text-slate-400 mt-1">Total Cost</div>
        </div>
      </div>

      {/* Per-Provider Sections */}
      {providerNames.length === 0 ? (
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-8 text-center text-slate-500">
          No usage data available yet.
        </div>
      ) : (
        <div className="space-y-4">
          {providerNames.map((provider) => {
            const models = providers[provider] ?? []
            let provTokens = 0
            let provCost = 0
            for (const m of models) {
              provTokens += m.total_tokens
              provCost += m.total_cost
            }
            const supportsBalance = providersData?.providers.find((p) => p.name === provider)?.supports_balance ?? false

            return (
              <details key={provider} className="bg-slate-800 border border-slate-700 rounded-lg group">
                <summary className="px-4 py-3 cursor-pointer hover:bg-slate-750 rounded-lg flex items-center gap-2 select-none [&::-webkit-details-marker]:hidden">
                  <span className="text-slate-400 group-open:rotate-90 transition-transform">{'>'}</span>
                  <span className="font-semibold text-lg capitalize">{provider}</span>
                  <span className="text-sm text-slate-400 ml-auto">
                    {models.length} model{models.length !== 1 ? 's' : ''}
                  </span>
                  <span className="text-sm text-slate-400 mx-3">|</span>
                  <span className="text-sm text-slate-400">{formatNumber(provTokens)} tokens</span>
                  <span className="text-sm text-slate-400 mx-3">|</span>
                  <span className="text-sm text-sky-400 font-mono">${provCost.toFixed(4)}</span>
                  {supportsBalance && balances[provider]?.balance && (
                    <>
                      <span className="text-sm text-slate-400 mx-3">|</span>
                      <span className="text-sm text-emerald-400 font-mono" title={`of $${(balances[provider].balance?.total_credits ?? 0).toFixed(2)} total`}>
                        ${(balances[provider].balance?.remaining_credits ?? 0).toFixed(2)} balance
                      </span>
                    </>
                  )}
                  {supportsBalance && balances[provider]?.loading && (
                    <span className="text-xs text-slate-500 ml-3">checking balance...</span>
                  )}
                  {supportsBalance && (
                    <button
                      onClick={(e) => {
                        e.preventDefault()
                        loadBalance(provider)
                      }}
                      title="Refresh balance"
                      className="ml-3 px-2 py-0.5 text-xs bg-slate-700 hover:bg-slate-600 rounded transition-colors"
                    >
                      ⟳
                    </button>
                  )}
                </summary>
                <div className="overflow-x-auto border-t border-slate-700">
                  {models.length === 0 ? (
                    <div className="px-4 py-6 text-sm text-slate-400">
                      No usage recorded yet for this provider.
                    </div>
                  ) : (
                    <table className="w-full">
                      <thead>
                        <tr className="text-left text-xs text-slate-400 uppercase">
                          <th className="px-4 py-3">Model</th>
                          <th className="px-4 py-3">Requests</th>
                          <th className="px-4 py-3">Tokens</th>
                          <th className="px-4 py-3">Cost</th>
                        </tr>
                      </thead>
                      <tbody>
                        {models.map((m) => (
                          <tr key={m.model} className="border-t border-slate-700">
                            <td className="px-4 py-3 font-medium">{m.model}</td>
                            <td className="px-4 py-3 text-slate-300">{formatNumber(m.requests)}</td>
                            <td className="px-4 py-3 text-slate-300">{formatNumber(m.total_tokens)}</td>
                            <td className="px-4 py-3 text-sky-400 font-mono">${(m.total_cost || 0).toFixed(6)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>
              </details>
            )
          })}
        </div>
      )}
    </div>
  )
}
