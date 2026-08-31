import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import AppLayout from './components/Layout/AppLayout'
import Login from './pages/Login'
import Overview from './pages/Overview'
import Alerts from './pages/Alerts'
import Rules from './pages/Rules'
import Channels from './pages/Channels'
import History from './pages/History'
import Process from './pages/Process'
import Service from './pages/Service'
import Settings from './pages/Settings'
import Realtime from './pages/Realtime'
import { useAuthStore } from './store/auth'

export default function App() {
  const token = useAuthStore((s) => s.token)
  const location = useLocation()

  if (!token) {
    return (
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="*" element={<Navigate to="/login" state={{ from: location }} replace />} />
      </Routes>
    )
  }

  return (
    <Routes>
      <Route path="/login" element={<Navigate to="/overview" replace />} />
      <Route element={<AppLayout />}>
        <Route path="/" element={<Navigate to="/overview" replace />} />
        <Route path="/overview" element={<Overview />} />
        <Route path="/alerts" element={<Alerts />} />
        <Route path="/rules" element={<Rules />} />
        <Route path="/channels" element={<Channels />} />
        <Route path="/history" element={<History />} />
        <Route path="/process" element={<Process />} />
        <Route path="/service" element={<Service />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/realtime" element={<Realtime />} />
        <Route path="*" element={<Navigate to="/overview" replace />} />
      </Route>
    </Routes>
  )
}
