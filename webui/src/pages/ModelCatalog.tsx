import { useEffect, useState } from 'react'

interface ModelInfo {
  name: string
  modified_at: string
  size: number
  digest: string
  details: {
    format: string
    family: string
    parameter_size: string
    quantization_level: string
  }
}

export default function ModelCatalog() {
  const [models, setModels] = useState<ModelInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null)
  const [selectedModel, setSelectedModel] = useState<ModelInfo | null>(null)

  useEffect(() => {
    async function load() {
      try {
        const resp = await fetch('/api/tags')
        const data = await resp.json()
        setModels(data.models || [])
      } catch (err) {
        console.error(err)
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  const providers = [...new Set(models.map(m => m.name.split('/')[0]))]

  const filtered = models.filter(m => {
    if (search && !m.name.toLowerCase().includes(search.toLowerCase())) return false
    if (selectedProvider && !m.name.startsWith(selectedProvider + '/')) return false
    return true
  })

  if (loading) return <div className="text-slate-400">Loading...</div>

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Model Catalog</h2>

      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-6">
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search models..."
          className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm focus:border-sky-500 focus:outline-none w-64"
        />
        <div className="flex gap-2">
          <button
            onClick={() => setSelectedProvider(null)}
            className={`px-3 py-1.5 rounded-full text-xs font-medium transition-colors ${
              selectedProvider === null
                ? 'bg-sky-600 text-white'
                : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
            }`}
          >
            All
          </button>
          {providers.map(p => (
            <button
              key={p}
              onClick={() => setSelectedProvider(selectedProvider === p ? null : p)}
              className={`px-3 py-1.5 rounded-full text-xs font-medium transition-colors ${
                selectedProvider === p
                  ? 'bg-sky-600 text-white'
                  : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
              }`}
            >
              {p}
            </button>
          ))}
        </div>
      </div>

      {/* Model Table */}
      <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="text-left text-xs text-slate-400 uppercase">
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Size</th>
              <th className="px-4 py-3">Provider</th>
              <th className="px-4 py-3">Family</th>
              <th className="px-4 py-3">Details</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((m) => {
              const provider = m.name.split('/')[0]
              return (
                <tr
                  key={m.name}
                  className="border-t border-slate-700 hover:bg-slate-700/50 cursor-pointer"
                  onClick={() => setSelectedModel(selectedModel?.name === m.name ? null : m)}
                >
                  <td className="px-4 py-3 font-medium">{m.name}</td>
                  <td className="px-4 py-3 text-sm text-slate-300">
                    {m.size > 0 ? `${(m.size / 1024 / 1024 / 1024).toFixed(1)} GB` : '☁️ cloud'}
                  </td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-1 bg-slate-700 rounded text-xs">{provider}</span>
                  </td>
                  <td className="px-4 py-3 text-sm text-slate-300">{m.details.family || '-'}</td>
                  <td className="px-4 py-3 text-xs text-slate-400">
                    {m.details.parameter_size ? `${m.details.parameter_size}` : '-'}
                  </td>
                </tr>
              )
            })}
            {filtered.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-slate-500">
                  {models.length === 0
                    ? 'No models available. Add a provider to see available models.'
                    : 'No models match your search criteria.'}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Model Detail Panel */}
      {selectedModel && (
        <div className="fixed inset-y-0 right-0 w-96 bg-slate-800 border-l border-slate-700 p-6 overflow-y-auto shadow-2xl">
          <div className="flex justify-between items-start mb-6">
            <h3 className="text-lg font-semibold">{selectedModel.name}</h3>
            <button
              onClick={() => setSelectedModel(null)}
              className="text-slate-400 hover:text-slate-200 text-xl"
            >
              ✕
            </button>
          </div>
          <div className="space-y-4">
            <div>
              <label className="text-xs text-slate-400 uppercase">Provider</label>
              <p className="text-sm mt-1">{selectedModel.name.split('/')[0]}</p>
            </div>
            <div>
              <label className="text-xs text-slate-400 uppercase">Size</label>
              <p className="text-sm mt-1">
                {selectedModel.size > 0 ? `${(selectedModel.size / 1024 / 1024 / 1024).toFixed(2)} GB` : 'Cloud (no size info)'}
              </p>
            </div>
            <div>
              <label className="text-xs text-slate-400 uppercase">Format</label>
              <p className="text-sm mt-1">{selectedModel.details.format || 'N/A'}</p>
            </div>
            <div>
              <label className="text-xs text-slate-400 uppercase">Family</label>
              <p className="text-sm mt-1">{selectedModel.details.family || 'N/A'}</p>
            </div>
            <div>
              <label className="text-xs text-slate-400 uppercase">Parameter Size</label>
              <p className="text-sm mt-1">{selectedModel.details.parameter_size || 'N/A'}</p>
            </div>
            <div>
              <label className="text-xs text-slate-400 uppercase">Quantization</label>
              <p className="text-sm mt-1">{selectedModel.details.quantization_level || 'N/A'}</p>
            </div>
            <div>
              <label className="text-xs text-slate-400 uppercase">Digest</label>
              <p className="text-xs mt-1 font-mono text-slate-400 break-all">{selectedModel.digest || 'N/A'}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
