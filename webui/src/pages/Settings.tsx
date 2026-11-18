import { useEffect, useState } from 'react'
import { api } from '../api/client'

export default function Settings() {
  const [settings, setSettings] = useState<any>(null)
  const [yaml, setYaml] = useState('')
  const [loading, setLoading] = useState(true)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    async function load() {
      try {
        const [s, y] = await Promise.all([
          api.getSettings(),
          api.getConfigYAML(),
        ])
        setSettings(s)
        setYaml(y)
      } catch (err) {
        console.error(err)
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  async function handleSave() {
    try {
      await api.updateSettings(settings)
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    } catch (err) {
      console.error(err)
    }
  }

  if (loading) return <div className="text-slate-400">Loading...</div>

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Settings</h2>

      {saved && (
        <div className="mb-4 px-4 py-3 bg-green-900/50 border border-green-700 rounded-lg text-sm text-green-300">
          Settings saved successfully
        </div>
      )}

      {/* General Settings */}
      <div className="bg-slate-800 border border-slate-700 rounded-lg p-4 mb-6">
        <h3 className="font-semibold mb-4">General Settings</h3>
        <div className="space-y-4 max-w-md">
          <div>
            <label className="block text-sm text-slate-400 mb-1">Port</label>
            <input
              type="number"
              value={settings?.port ?? 11435}
              disabled
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm opacity-50"
            />
          </div>
          <div>
            <label className="block text-sm text-slate-400 mb-1">Log Level</label>
            <select
              value={settings?.log_level ?? 'info'}
              onChange={(e) => setSettings({ ...settings, log_level: e.target.value })}
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm"
            >
              <option value="debug">Debug</option>
              <option value="info">Info</option>
              <option value="warn">Warn</option>
              <option value="error">Error</option>
            </select>
          </div>
          <div>
            <label className="block text-sm text-slate-400 mb-1">Prometheus Metrics</label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={settings?.prometheus_enabled ?? true}
                onChange={(e) => setSettings({ ...settings, prometheus_enabled: e.target.checked })}
                className="w-4 h-4"
              />
              <span className="text-sm">Enabled</span>
            </label>
          </div>
          <div>
            <label className="block text-sm text-slate-400 mb-1">Max Retries</label>
            <input
              type="number"
              value={settings?.max_retries ?? 3}
              onChange={(e) => setSettings({ ...settings, max_retries: parseInt(e.target.value) || 0 })}
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm"
            />
          </div>
          <div>
            <label className="block text-sm text-slate-400 mb-1">Timeout (seconds)</label>
            <input
              type="number"
              value={settings?.timeout_seconds ?? 30}
              onChange={(e) => setSettings({ ...settings, timeout_seconds: parseInt(e.target.value) || 0 })}
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm"
            />
          </div>
          <button
            onClick={handleSave}
            className="px-4 py-2 bg-sky-600 hover:bg-sky-500 rounded-lg text-sm font-medium transition-colors"
          >
            Save All
          </button>
        </div>
      </div>

      {/* YAML Preview */}
      <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
        <h3 className="font-semibold mb-4">Runtime Configuration (YAML)</h3>
        <pre className="text-xs text-slate-300 font-mono bg-slate-900 p-4 rounded-lg overflow-x-auto max-h-96 overflow-y-auto">
          {yaml}
        </pre>
      </div>
    </div>
  )
}
