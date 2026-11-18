import { useEffect, useState } from 'react'
import { api, type HealthResponse, type StatsSummary, type ProviderListResponse } from '../api/client'

export default function Dashboard() {
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [stats, setStats] = useState<StatsSummary | null>(null)
  const [providers, setProviders] = useState<ProviderListResponse | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function load() {
      try {
        const [h, s, p] = await Promise.all([
          api.getHealth(),
          api.getStatsSummary().catch(() => null),
          api.listProviders(),
        ])
        setHealth(h)
        setStats(s)
        setProviders(p)
        console.log('Dashboard data loaded:', { health: h, stats: s, providers: p })
      } catch (err) {
        console.error('Failed to load dashboard:', err)
      } finally {
        setLoading(false)
      }
    }
    load()
    const interval = setInterval(load, 30000)
    return () => clearInterval(interval)
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-400">Loading dashboard...</div>
      </div>
    )
  }

  const uptimeHours = health ? Math.round(health.uptime_seconds / 3600) : 0

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Dashboard{health?.version ? ` (v${health.version})` : ''}</h2>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-8">
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
          <div className="text-3xl font-bold text-sky-400">{stats?.total_requests ?? 0}</div>
          <div className="text-sm text-slate-400 mt-1">Total Requests</div>
        </div>
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
          <div className="text-3xl font-bold text-sky-400">${(stats?.total_cost ?? 0).toFixed(2)}</div>
          <div className="text-sm text-slate-400 mt-1">Total Cost</div>
        </div>
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
          <div className="text-3xl font-bold text-sky-400">{(stats?.total_tokens ?? 0).toLocaleString('en-US')}</div>
          <div className="text-sm text-slate-400 mt-1">Total Tokens</div>
        </div>
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
          <div className="text-3xl font-bold text-green-400">{health?.providers_online ?? 0}</div>
          <div className="text-sm text-slate-400 mt-1">Providers Online</div>
        </div>
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
          <div className="text-3xl font-bold text-sky-400">{uptimeHours}h</div>
          <div className="text-sm text-slate-400 mt-1">Uptime</div>
        </div>
      </div>

      {/* Provider Status Table */}
      <div className="bg-slate-800 border border-slate-700 rounded-lg">
        <div className="px-4 py-3 border-b border-slate-700">
          <h3 className="font-semibold">Provider Status</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="text-left text-xs text-slate-400 uppercase">
                <th className="px-4 py-3">Name</th>
                <th className="px-4 py-3">Type</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Models</th>
                <th className="px-4 py-3">Cost</th>
              </tr>
            </thead>
            <tbody>
              {providers?.providers?.map((p) => (
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
                  <td className="px-4 py-3 text-sky-400 font-mono">${(p.total_cost || 0).toFixed(6)}</td>
                </tr>
              ))}
              {(!providers?.providers || providers.providers.length === 0) && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-slate-500">
                    No providers configured. Add your first provider to get started.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
