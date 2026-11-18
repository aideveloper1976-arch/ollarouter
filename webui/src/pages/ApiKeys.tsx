import { useEffect, useState } from 'react'
import { api } from '../api/client'

export default function ApiKeys() {
  const [keys, setKeys] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [newKey, setNewKey] = useState<{ key: string; name: string } | null>(null)
  const [form, setForm] = useState({ name: '', rateLimit: 100, monthlyBudget: 100 })

  useEffect(() => { loadKeys() }, [])

  async function loadKeys() {
    try {
      const resp = await api.listApiKeys()
      setKeys(resp.api_keys)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  async function handleCreate() {
    try {
      const resp = await api.createApiKey({
        name: form.name,
        rate_limit: form.rateLimit,
        monthly_budget_usd: form.monthlyBudget,
      })
      setNewKey({ key: resp.key, name: resp.name })
      setShowCreate(false)
      loadKeys()
    } catch (err) {
      console.error(err)
    }
  }

  async function handleRevoke(id: string) {
    if (!window.confirm('Revoke this API key?')) return
    try {
      await api.revokeApiKey(id)
      loadKeys()
    } catch (err) {
      console.error(err)
    }
  }

  if (loading) return <div className="text-slate-400">Loading...</div>

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">API Keys</h2>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 bg-sky-600 hover:bg-sky-500 rounded-lg text-sm font-medium transition-colors"
        >
          New Key
        </button>
      </div>

      {/* New Key Created Dialog */}
      {newKey && (
        <div className="mb-4 p-4 bg-yellow-900/30 border border-yellow-700 rounded-lg">
          <h4 className="font-semibold text-yellow-300 mb-2">Key Created: {newKey.name}</h4>
          <code className="block p-2 bg-slate-900 rounded text-sm mb-2 break-all">{newKey.key}</code>
          <p className="text-sm text-yellow-400 font-medium">⚠️ Save this key - it will not be shown again</p>
          <button onClick={() => setNewKey(null)} className="mt-2 text-sm text-sky-400 hover:text-sky-300">Dismiss</button>
        </div>
      )}

      {/* Create Dialog */}
      {showCreate && (
        <div className="mb-4 p-4 bg-slate-800 border border-slate-700 rounded-lg">
          <h4 className="font-semibold mb-4">Create API Key</h4>
          <div className="space-y-3 max-w-sm">
            <div>
              <label className="block text-sm text-slate-400 mb-1">Key Name</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="dev-key"
                className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm focus:border-sky-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="block text-sm text-slate-400 mb-1">Rate Limit (req/min)</label>
              <input
                type="number"
                value={form.rateLimit}
                onChange={(e) => setForm({ ...form, rateLimit: parseInt(e.target.value) || 0 })}
                className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm focus:border-sky-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="block text-sm text-slate-400 mb-1">Monthly Budget ($USD)</label>
              <input
                type="number"
                value={form.monthlyBudget}
                onChange={(e) => setForm({ ...form, monthlyBudget: parseFloat(e.target.value) || 0 })}
                className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm focus:border-sky-500 focus:outline-none"
              />
            </div>
            <div className="flex gap-2">
              <button onClick={handleCreate} className="px-4 py-2 bg-sky-600 hover:bg-sky-500 rounded text-sm">Create</button>
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded text-sm">Cancel</button>
            </div>
          </div>
        </div>
      )}

      {/* Keys Table */}
      <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="text-left text-xs text-slate-400 uppercase">
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Key</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {keys.map((k: any) => (
              <tr key={k.id} className="border-t border-slate-700">
                <td className="px-4 py-3 font-medium">{k.name}</td>
                <td className="px-4 py-3">
                  <code className="text-xs text-slate-400">{k.key_masked || k.key?.substring(0, 12) + '...'}</code>
                </td>
                <td className="px-4 py-3">
                  <span className="text-green-400">active</span>
                </td>
                <td className="px-4 py-3">
                  <button
                    onClick={() => handleRevoke(k.id)}
                    className="px-2 py-1 text-xs bg-red-900/50 hover:bg-red-800/50 text-red-300 rounded"
                  >
                    Revoke
                  </button>
                </td>
              </tr>
            ))}
            {keys.length === 0 && (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-slate-500">
                  No API keys configured
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
