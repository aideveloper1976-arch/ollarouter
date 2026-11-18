import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, type ProviderBalance, type ProviderListResponse, type UsageRecord } from '../api/client'

export default function Providers() {
  const [data, setData] = useState<ProviderListResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [testResult, setTestResult] = useState<{name: string; status: string; latency_ms?: number} | null>(null)
  const [providerLogs, setProviderLogs] = useState<UsageRecord[]>([])
  const [logProvider, setLogProvider] = useState<string | null>(null)
  const [logLoading, setLogLoading] = useState(false)
  const [logError, setLogError] = useState('')
  const [balanceResult, setBalanceResult] = useState<{ name: string; type: string; balance?: ProviderBalance; error?: string } | null>(null)

  useEffect(() => {
    loadProviders()
  }, [])

  async function loadProviders() {
    try {
      const d = await api.listProviders()
      setData(d)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  async function handleTest(name: string) {
    setLogError('')
    setLogProvider(name)
    setLogLoading(true)
    try {
      const result = await api.testProvider(name)
      setTestResult({ name, status: result.status, latency_ms: result.latency_ms })
      const logsResp = await api.getLogs(name, 50)
      setProviderLogs(logsResp.logs)
      setTimeout(() => setTestResult(null), 3000)
    } catch (err) {
      setTestResult({ name, status: 'error' })
      const message = err instanceof Error ? err.message : 'Failed to fetch provider logs'
      setLogError(message)
    } finally {
      setLogLoading(false)
    }
  }

  async function handleDelete(name: string) {
    if (!window.confirm(`Delete provider "${name}"?`)) return
    try {
      await api.deleteProvider(name)
      loadProviders()
    } catch (err) {
      console.error(err)
    }
  }

  async function handleBalance(name: string) {
    try {
      const result = await api.getProviderBalance(name)
      setBalanceResult({ name: result.provider, type: result.type, balance: result.balance })
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch provider balance'
      const providerType = data?.providers.find((p) => p.name === name)?.type ?? 'unknown'
      setBalanceResult({ name, type: providerType, error: message })
    }
  }

  if (loading) {
    return <div className="text-slate-400">Loading providers...</div>
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">Providers</h2>
        <Link
          to="/providers/new"
          className="px-4 py-2 bg-sky-600 hover:bg-sky-500 rounded-lg text-sm font-medium transition-colors"
        >
          Add Provider
        </Link>
      </div>

      {testResult && (
        <div className={`mb-4 px-4 py-3 rounded-lg text-sm ${
          testResult.status === 'online' || testResult.status === 'ok'
            ? 'bg-green-900/50 text-green-300 border border-green-700'
            : 'bg-red-900/50 text-red-300 border border-red-700'
        }`}>
          {testResult.name}: {testResult.status}
          {testResult.latency_ms ? ` (${testResult.latency_ms}ms)` : ''}
        </div>
      )}

      {balanceResult && (
        <div className={`mb-4 px-4 py-3 rounded-lg text-sm ${
          balanceResult.error
            ? 'bg-red-900/50 text-red-300 border border-red-700'
            : 'bg-sky-900/40 text-sky-100 border border-sky-700'
        }`}>
          {balanceResult.error ? (
            <span>{balanceResult.name}: {balanceResult.error}</span>
          ) : (
            <span>
              {balanceResult.name} balance:
              {' '}remaining ${((balanceResult.balance?.remaining_credits ?? 0)).toFixed(2)}
              {' '}of ${(balanceResult.balance?.total_credits ?? 0).toFixed(2)}
              {' '}used ${(balanceResult.balance?.total_usage ?? 0).toFixed(2)}
            </span>
          )}
        </div>
      )}

      <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="text-left text-xs text-slate-400 uppercase">
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Type</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Models</th>
              <th className="px-4 py-3">Cost</th>
              <th className="px-4 py-3">Circuit Breaker</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {data?.providers?.map((p) => (
              <tr key={p.name} className="border-t border-slate-700">
                <td className="px-4 py-3 font-medium">{p.name}</td>
                <td className="px-4 py-3">
                  <span className="px-2 py-1 bg-slate-700 rounded text-xs">{p.type}</span>
                </td>
                <td className="px-4 py-3">
                  <span className={`inline-flex items-center gap-1.5`}>
                    <span className={`w-2 h-2 rounded-full ${
                      p.status === 'online' ? 'bg-green-400' :
                      p.status === 'degraded' ? 'bg-amber-400' : 'bg-red-400'
                    }`} />
                    <span className={
                      p.status === 'online' ? 'text-green-400' :
                      p.status === 'degraded' ? 'text-amber-400' : 'text-red-400'
                    }>
                      {p.status}
                    </span>
                  </span>
                </td>
                <td className="px-4 py-3 text-slate-300">{p.models_count}</td>
                <td className="px-4 py-3 text-sky-400 font-mono">${(p.total_cost || 0).toFixed(4)}</td>
                <td className="px-4 py-3 text-sm">
                  <span className={`px-2 py-0.5 rounded ${
                    p.circuit_breaker.state === 'closed' ? 'bg-green-900/30 text-green-400' : 'bg-red-900/30 text-red-400'
                  }`}>
                    {p.circuit_breaker.state} ({p.circuit_breaker.failures})
                  </span>
                </td>
                <td className="px-4 py-3">
                  <div className="flex gap-2 flex-wrap">
                    <Link
                      to={`/providers/${encodeURIComponent(p.name)}/edit`}
                      className="px-2 py-1 text-xs bg-slate-700 hover:bg-slate-600 rounded transition-colors"
                    >
                      Edit
                    </Link>
                    <button
                      onClick={() => handleTest(p.name)}
                      className="px-2 py-1 text-xs bg-slate-700 hover:bg-slate-600 rounded transition-colors"
                    >
                      Test
                    </button>
                    {p.supports_balance && (
                      <button
                        onClick={() => handleBalance(p.name)}
                        className="px-2 py-1 text-xs bg-slate-700 hover:bg-slate-600 rounded transition-colors"
                      >
                        Balance
                      </button>
                    )}
                    <button
                      onClick={() => handleDelete(p.name)}
                      className="px-2 py-1 text-xs bg-red-900/50 hover:bg-red-800/50 text-red-300 rounded transition-colors"
                    >
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {(!data?.providers || data.providers.length === 0) && (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                  No providers configured. Add your first provider to get started.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {logProvider && (
        <div className="mt-6 bg-slate-800 border border-slate-700 rounded-lg p-4">
          <div className="flex items-center justify-between gap-4 mb-4">
            <div>
              <h3 className="text-lg font-semibold">Logs for {logProvider}</h3>
              <p className="text-sm text-slate-400">Showing the latest provider activity after the test run.</p>
            </div>
            <button
              onClick={() => {
                setLogProvider(null)
                setProviderLogs([])
                setLogError('')
              }}
              className="px-3 py-2 text-xs bg-slate-700 hover:bg-slate-600 rounded transition-colors"
            >
              Close
            </button>
          </div>

          {logLoading ? (
            <div className="text-slate-400">Loading logs...</div>
          ) : logError ? (
            <div className="text-red-400">{logError}</div>
          ) : providerLogs.length === 0 ? (
            <div className="text-slate-400">No recent provider logs found.</div>
          ) : (
            <div className="space-y-3 max-h-72 overflow-y-auto">
              {providerLogs.map((log, idx) => (
                <div key={`${log.timestamp}-${idx}`} className="rounded-lg border border-slate-700 bg-slate-900/80 p-3 text-sm">
                  <div className="flex flex-wrap items-center justify-between gap-2 text-slate-300">
                    <span>{new Date(log.timestamp).toLocaleString()}</span>
                    <span className="text-xs uppercase tracking-widest text-slate-500">{log.request_type}</span>
                  </div>
                  <div className="mt-2 text-slate-200">{log.provider}/{log.model}</div>
                  <div className="mt-1 text-xs text-slate-500">
                    Status: {log.status} | Latency: {log.latency_ms}ms | Tokens: {log.total_tokens} | Cost: ${log.cost_usd.toFixed(4)}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
