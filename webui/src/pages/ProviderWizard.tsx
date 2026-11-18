import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'

const PROVIDER_TYPES = [
  { id: 'openai', name: 'OpenAI', description: 'GPT-4, GPT-4o, GPT-4o-mini', icon: '\u{1F535}' },
  { id: 'anthropic', name: 'Anthropic', description: 'Claude Sonnet, Claude Haiku', icon: '\u{1F7E3}' },
  { id: 'google', name: 'Google', description: 'Gemini Pro, Gemini Flash', icon: '\u{1F7E1}' },
  { id: 'ollama', name: 'Ollama', description: 'Local LLMs', icon: '\u{1F999}' },
  { id: 'openrouter', name: 'OpenRouter', description: 'Allowed models via OpenRouter', icon: '\u{1F9ED}' },
  { id: 'openai-compatible', name: 'OpenAI Compatible', description: 'Groq, Together, Azure', icon: '\u{1F517}' },
  { id: 'chatgpt-plus', name: 'ChatGPT Plus', description: 'Via session cookie', icon: '\u{1F4AC}' },
]

const DEFAULT_URLS: Record<string, string> = {
  openai: 'https://api.openai.com/v1',
  anthropic: 'https://api.anthropic.com/v1',
  google: 'https://generativelanguage.googleapis.com/v1beta',
  ollama: 'http://ollama:11434',
  openrouter: 'https://openrouter.ai/api/v1',
  'openai-compatible': '',
  'chatgpt-plus': 'https://chatgpt.com',
}

export default function ProviderWizard() {
  const { name: providerName } = useParams()
  const editMode = typeof providerName === 'string' && providerName.length > 0
  const navigate = useNavigate()
  const [step, setStep] = useState(1)
  const [saving, setSaving] = useState(false)
  const [loading, setLoading] = useState(editMode)
  const [form, setForm] = useState({
    name: '',
    type: '',
    baseUrl: '',
    apiKey: '',
    autoDiscover: false,
    inputPricePer1K: '',
    outputPricePer1K: '',
    models: [] as string[],
  })
  const [error, setError] = useState('')

  useEffect(() => {
    if (!editMode) {
      return
    }

    async function loadProvider() {
      try {
        const response = await api.listProviders()
        const provider = response.providers.find((p) => p.name === providerName)
        if (!provider) {
          setError('Provider not found.')
          return
        }

        setForm({
          name: provider.name,
          type: provider.type,
          baseUrl: provider.base_url || '',
          apiKey: '',
          autoDiscover: provider.auto_discover || false,
          inputPricePer1K: provider.pricing ? String(provider.pricing.input_per_1k) : '',
          outputPricePer1K: provider.pricing ? String(provider.pricing.output_per_1k) : '',
          models: provider.models || [],
        })
      } catch (err) {
        console.error(err)
        setError('Failed to load provider.')
      } finally {
        setLoading(false)
      }
    }

    loadProvider()
  }, [editMode, providerName])

  const [discoveredModels, setDiscoveredModels] = useState<string[]>([])
  const [discoveringModels, setDiscoveringModels] = useState(false)
  const [discoveryError, setDiscoveryError] = useState('')

  useEffect(() => {
    if (!form.autoDiscover || step !== 3 || !form.type || !form.baseUrl) {
      return
    }

    if (discoveredModels.length > 0 || discoveringModels) {
      return
    }

    async function fetchDiscoveredModels() {
      setDiscoveryError('')
      setDiscoveringModels(true)
      try {
        const response = await api.discoverProviderModels({
          name: form.name,
          type: form.type,
          base_url: form.baseUrl,
          api_key: form.apiKey,
          auto_discover: true,
        })
        setDiscoveredModels(response.models)
        if (response.models.length === 0) {
          setDiscoveryError('No models were discovered for this provider. You can still enter model names manually.')
        }
      } catch (err) {
        console.error(err)
        setDiscoveryError((err as Error).message || 'Failed to discover models from provider')
        setDiscoveredModels([])
      } finally {
        setDiscoveringModels(false)
      }
    }

    fetchDiscoveredModels()
  }, [step, form.autoDiscover, form.type, form.baseUrl, form.apiKey, discoveredModels.length, discoveringModels])

  useEffect(() => {
    if (!form.autoDiscover) {
      setDiscoveredModels([])
      setDiscoveryError('')
      return
    }

    setDiscoveredModels([])
    setDiscoveryError('')
  }, [form.autoDiscover, form.type, form.baseUrl, form.apiKey])

  function selectType(type: string) {
    setForm({ ...form, type, baseUrl: DEFAULT_URLS[type] || '' })
  }

  function openProviderUrl() {
    let url = (form.baseUrl || '').trim()
    if (!url) return
    if (!/^https?:\/\//i.test(url)) {
      url = 'https://' + url
    }
    try {
      window.open(url, '_blank')
    } catch (e) {
      console.error('failed to open url', e)
    }
  }

  async function handleSave() {
    setSaving(true)
    try {
      const pricing = form.inputPricePer1K !== '' || form.outputPricePer1K !== ''
        ? {
            input_per_1k: Number(form.inputPricePer1K) || 0,
            output_per_1k: Number(form.outputPricePer1K) || 0,
          }
        : undefined

      if (editMode && providerName) {
        await api.updateProvider(providerName, {
          name: form.name,
          type: form.type,
          base_url: form.baseUrl,
          api_key: form.apiKey,
          auto_discover: form.autoDiscover,
          pricing,
          models: form.models.map((m) => ({ name: m })),
        })
      } else {
        await api.addProvider({
          name: form.name,
          type: form.type,
          base_url: form.baseUrl,
          api_key: form.apiKey,
          auto_discover: form.autoDiscover,
          pricing,
          models: form.models.map((m) => ({ name: m })),
        })
      }
      navigate('/providers')
    } catch (err) {
      console.error(err)
      alert('Failed to save provider')
    } finally {
      setSaving(false)
    }
  }

  const canNextStep1 = form.type !== ''
  const canNextStep2 = form.name !== ''
  const canNextStep3 = form.models.length > 0 || form.autoDiscover
  const activeModels = form.autoDiscover && discoveredModels.length > 0 ? discoveredModels : form.models

  if (loading) {
    return <div className="text-slate-400">Loading provider...</div>
  }

  if (error) {
    return (
      <div className="text-slate-300">
        <p className="text-lg font-semibold mb-4">Edit Provider</p>
        <p className="text-red-400">{error}</p>
      </div>
    )
  }

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">{editMode ? 'Edit Provider' : 'Add Provider'}</h2>

      <div className="flex gap-2 mb-8">
        {[1, 2, 3, 4].map((s) => (
          <div key={s} className={`flex-1 h-1 rounded ${s <= step ? 'bg-sky-500' : 'bg-slate-700'}`} />
        ))}
      </div>

      {step === 1 && (
        <div>
          <h3 className="text-lg font-semibold mb-4">Select Provider Type</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {PROVIDER_TYPES.map((pt) => (
              <button
                key={pt.id}
                onClick={() => selectType(pt.id)}
                className={`p-4 rounded-lg border text-left transition-all ${
                  form.type === pt.id
                    ? 'border-sky-500 bg-sky-900/30'
                    : 'border-slate-700 bg-slate-800 hover:border-slate-600'
                }`}
              >
                <div className="text-2xl mb-2">{pt.icon}</div>
                <div className="font-medium">{pt.name}</div>
                <div className="text-sm text-slate-400 mt-1">{pt.description}</div>
              </button>
            ))}
          </div>
          <div className="mt-6 flex justify-end">
            <button
              onClick={() => setStep(2)}
              disabled={!canNextStep1}
              className="px-4 py-2 bg-sky-600 hover:bg-sky-500 disabled:bg-slate-700 disabled:text-slate-500 rounded-lg text-sm font-medium transition-colors"
            >
              Next
            </button>
          </div>
        </div>
      )}

      {step === 2 && (
        <div>
          <h3 className="text-lg font-semibold mb-4">Connection Details</h3>
          <div className="space-y-4 max-w-md">
            <div>
              <label className="block text-sm text-slate-400 mb-1">Provider Name</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="my-openai-provider"
                disabled={editMode}
                className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed disabled:opacity-60"
              />
            </div>
            <div>
              <label className="block text-sm text-slate-400 mb-1">Base URL</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={form.baseUrl}
                  onChange={(e) => setForm({ ...form, baseUrl: e.target.value })}
                  placeholder="https://api.openai.com/v1"
                  className="flex-1 px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm focus:border-sky-500 focus:outline-none"
                />
                <button
                  type="button"
                  onClick={openProviderUrl}
                  className="px-3 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-sm font-medium transition-colors"
                >
                  Open
                </button>
              </div>
            </div>
            <div>
              <label className="block text-sm text-slate-400 mb-1">API Key</label>
              <input
                type="password"
                value={form.apiKey}
                onChange={(e) => setForm({ ...form, apiKey: e.target.value })}
                placeholder="sk-..."
                className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm focus:border-sky-500 focus:outline-none"
              />
            </div>
            <div className="flex items-center gap-3">
              <input
                id="auto-discover"
                type="checkbox"
                checked={form.autoDiscover}
                onChange={(e) => setForm({ ...form, autoDiscover: e.target.checked })}
                className="h-4 w-4 rounded border-slate-600 bg-slate-900 text-sky-500 focus:ring-sky-500"
              />
              <label htmlFor="auto-discover" className="text-sm text-slate-300">
                Automatically retrieve model list from provider
              </label>
            </div>
            <div className="rounded-lg border border-slate-700 bg-slate-850 p-4">
              <p className="text-sm font-medium text-slate-200 mb-3">Pricing (optional)</p>
              <p className="text-xs text-slate-400 mb-3">
                Used for OllaRouter's estimated cost tracking. Enter USD cost per 1K input and output tokens.
              </p>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Input / 1K tokens</label>
                  <input
                    type="number"
                    min="0"
                    step="0.000001"
                    value={form.inputPricePer1K}
                    onChange={(e) => setForm({ ...form, inputPricePer1K: e.target.value })}
                    placeholder="0.000000"
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm focus:border-sky-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Output / 1K tokens</label>
                  <input
                    type="number"
                    min="0"
                    step="0.000001"
                    value={form.outputPricePer1K}
                    onChange={(e) => setForm({ ...form, outputPricePer1K: e.target.value })}
                    placeholder="0.000000"
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm focus:border-sky-500 focus:outline-none"
                  />
                </div>
              </div>
            </div>
          </div>
          <div className="mt-6 flex justify-between">
            <button
              onClick={() => setStep(1)}
              className="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-sm font-medium transition-colors"
            >
              Back
            </button>
            <button
              onClick={() => setStep(3)}
              disabled={!canNextStep2}
              className="px-4 py-2 bg-sky-600 hover:bg-sky-500 disabled:bg-slate-700 disabled:text-slate-500 rounded-lg text-sm font-medium transition-colors"
            >
              Next
            </button>
          </div>
        </div>
      )}

      {step === 3 && (
        <div>
          <h3 className="text-lg font-semibold mb-4">Configure Models</h3>
          <div className="flex items-center gap-3 mb-4">
            <input
              id="auto-discover-step3"
              type="checkbox"
              checked={form.autoDiscover}
              onChange={(e) => setForm({ ...form, autoDiscover: e.target.checked })}
              className="h-4 w-4 rounded border-slate-600 bg-slate-900 text-sky-500 focus:ring-sky-500"
            />
            <label htmlFor="auto-discover-step3" className="text-sm text-slate-300">
              Automatically retrieve model list from provider
            </label>
          </div>
          <p className="text-sm text-slate-400 mb-4">
            {form.autoDiscover
              ? 'Auto-discover is enabled. Models will be retrieved automatically from the provider.'
              : 'Enter model names (one per line).'}
          </p>

          {form.autoDiscover && (
            <div className="mb-4 rounded-lg border border-slate-700 bg-slate-850 p-4 text-sm text-slate-300">
              {discoveringModels ? (
                <p>Discovering models from the provider...</p>
              ) : discoveryError ? (
                <p className="text-red-400">{discoveryError}</p>
              ) : discoveredModels.length > 0 ? (
                <>
                  <p className="mb-2 text-slate-400">Discovered models:</p>
                  <ul className="list-disc list-inside space-y-1">
                    {discoveredModels.map((model) => (
                      <li key={model} className="text-slate-200">
                        {model}
                      </li>
                    ))}
                  </ul>
                </>
              ) : (
                <p>No models were discovered yet. You may still enter model names manually.</p>
              )}
            </div>
          )}

          <textarea
            value={form.models.join('\n')}
            onChange={(e) => setForm({ ...form, models: e.target.value.split('\n').filter((m) => m.trim()) })}
            placeholder="gpt-4o&#10;gpt-4o-mini&#10;gpt-4-turbo"
            rows={6}
            disabled={form.autoDiscover && discoveredModels.length > 0}
            className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm font-mono focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed disabled:opacity-70"
          />
          <div className="mt-6 flex justify-between">
            <button
              onClick={() => setStep(2)}
              className="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-sm font-medium transition-colors"
            >
              Back
            </button>
            <button
              onClick={() => setStep(4)}
              disabled={!canNextStep3}
              className="px-4 py-2 bg-sky-600 hover:bg-sky-500 disabled:bg-slate-700 disabled:text-slate-500 rounded-lg text-sm font-medium transition-colors"
            >
              Next
            </button>
          </div>
        </div>
      )}

      {step === 4 && (
        <div>
          <h3 className="text-lg font-semibold mb-4">Review & Save</h3>
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4 space-y-3 max-w-md">
            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Type:</span>
              <span>{PROVIDER_TYPES.find((t) => t.id === form.type)?.name || form.type}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Name:</span>
              <span>{form.name}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Base URL:</span>
              <span className="text-xs truncate max-w-[200px]">{form.baseUrl}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Models:</span>
              <span>{activeModels.length}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Pricing:</span>
              <span>
                {form.inputPricePer1K || form.outputPricePer1K
                  ? `$${form.inputPricePer1K || '0'} in / $${form.outputPricePer1K || '0'} out`
                  : 'Not set'}
              </span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Auto-discover:</span>
              <span>{form.autoDiscover ? 'Enabled' : 'Disabled'}</span>
            </div>
          </div>
          <div className="mt-6 flex justify-between">
            <button
              onClick={() => setStep(3)}
              className="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-sm font-medium transition-colors"
            >
              Back
            </button>
            <button
              onClick={handleSave}
              disabled={saving}
              className="px-4 py-2 bg-sky-600 hover:bg-sky-500 disabled:bg-slate-700 rounded-lg text-sm font-medium transition-colors"
            >
              {saving ? (editMode ? 'Updating...' : 'Saving...') : editMode ? 'Update Provider' : 'Save Provider'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
