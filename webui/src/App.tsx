import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Providers from './pages/Providers'
import ProviderWizard from './pages/ProviderWizard'
import Settings from './pages/Settings'
import ModelCatalog from './pages/ModelCatalog'
import UsageAnalytics from './pages/UsageAnalytics'

function App() {
  return (
    <BrowserRouter basename="/admin">
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/providers" element={<Providers />} />
          <Route path="/providers/new" element={<ProviderWizard />} />
          <Route path="/providers/:name/edit" element={<ProviderWizard />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="/models" element={<ModelCatalog />} />
          <Route path="/usage" element={<UsageAnalytics />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App
