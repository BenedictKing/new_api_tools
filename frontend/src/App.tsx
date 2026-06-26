import { lazy, Suspense, useEffect, useMemo, useState, type ComponentType, type LazyExoticComponent } from 'react'
import { Login } from './components/Login'
import { Layout, type TabType } from './components/Layout'
import { useAuth } from './contexts/AuthContext'
import { WarmupScreen } from './components/WarmupScreen'

const Dashboard = lazy(() => import('./components/Dashboard').then(module => ({ default: module.Dashboard })))
const TopUps = lazy(() => import('./components/TopUps').then(module => ({ default: module.TopUps })))
const RedemptionCenter = lazy(() => import('./components/RedemptionCenter').then(module => ({ default: module.RedemptionCenter })))
const Analytics = lazy(() => import('./components/Analytics').then(module => ({ default: module.Analytics })))
const UserManagement = lazy(() => import('./components/UserManagement').then(module => ({ default: module.UserManagement })))
const RealtimeRanking = lazy(() => import('./components/RealtimeRanking').then(module => ({ default: module.RealtimeRanking })))
const IPAnalysis = lazy(() => import('./components/IPAnalysis').then(module => ({ default: module.IPAnalysis })))
const ModelStatusMonitor = lazy(() => import('./components/ModelStatusMonitor').then(module => ({ default: module.ModelStatusMonitor })))
const AutoGroup = lazy(() => import('./components/AutoGroup').then(module => ({ default: module.AutoGroup })))
const Tokens = lazy(() => import('./components/Tokens').then(module => ({ default: module.Tokens })))
const AbuseBroadcast = lazy(() => import('./components/AbuseBroadcast').then(module => ({ default: module.AbuseBroadcast })))

type TabComponent = ComponentType | LazyExoticComponent<ComponentType>

// Valid tabs
const validTabs: TabType[] = ['dashboard', 'topups', 'risk', 'abuse-broadcast', 'ip-analysis', 'analytics', 'model-status', 'users', 'tokens', 'auto-group', 'redemptions']

// 旧路径迁移：generator / history 现合并到 redemptions 内部 tab
const legacyRedirects: Record<string, string> = {
  generator: '/redemptions?view=generator',
  history: '/redemptions?view=history',
}

const tabComponents: Record<TabType, TabComponent> = {
  dashboard: Dashboard,
  topups: TopUps,
  risk: RealtimeRanking,
  'abuse-broadcast': AbuseBroadcast,
  'ip-analysis': IPAnalysis,
  analytics: Analytics,
  'model-status': ModelStatusMonitor,
  users: UserManagement,
  tokens: Tokens,
  'auto-group': AutoGroup,
  redemptions: RedemptionCenter,
}

// Get initial tab from URL pathname (supports sub-routes like /risk/ip)
const getInitialTab = (): TabType => {
  const pathname = window.location.pathname.slice(1) // Remove leading /
  const mainPath = pathname.split('/')[0] // Get first segment for main tab

  if (legacyRedirects[mainPath]) {
    window.history.replaceState(null, '', legacyRedirects[mainPath])
    return 'redemptions'
  }

  if (validTabs.includes(mainPath as TabType)) {
    return mainPath as TabType
  }
  // 兼容旧的 hash 路由，自动迁移
  const hash = window.location.hash.slice(1)
  // 处理 #risk/ip 等格式
  const hashMain = hash.split('/')[0].replace('risk-', 'risk/')
  if (legacyRedirects[hashMain]) {
    window.history.replaceState(null, '', legacyRedirects[hashMain])
    return 'redemptions'
  }
  if (validTabs.includes(hashMain as TabType)) {
    // 重定向到新路由
    const subPath = hash.includes('/') ? hash.split('/').slice(1).join('/') : ''
    const newPath = subPath ? `/${hashMain}/${subPath}` : `/${hashMain}`
    window.history.replaceState(null, '', newPath)
    return hashMain as TabType
  }
  return 'dashboard'
}

function App() {
  const { isAuthenticated, token, login, logout } = useAuth()
  const [activeTab, setActiveTab] = useState<TabType>(getInitialTab)
  const [warmupState, setWarmupState] = useState<'checking' | 'warming' | 'ready'>('checking')

  const apiUrl = import.meta.env.VITE_API_URL || ''

  // 检查后端预热状态
  useEffect(() => {
    if (!isAuthenticated || !token) return

    const checkWarmupStatus = async () => {
      try {
        const response = await fetch(`${apiUrl}/api/system/warmup-status`, {
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
        })

        // 处理 401 未授权错误 - token 失效，需要重新登录
        if (response.status === 401) {
          console.warn('Token invalid or expired, logging out...')
          logout()
          return
        }

        const data = await response.json()

        if (data.success && data.data.status === 'ready') {
          // 后端已预热完成，直接进入
          setWarmupState('ready')
        } else {
          // 后端还在预热中，显示预热界面
          setWarmupState('warming')
        }
      } catch {
        // 网络错误，显示预热界面让它处理
        setWarmupState('warming')
      }
    }

    checkWarmupStatus()
  }, [isAuthenticated, token, apiUrl, logout])

  // Sync tab with URL pathname (History API)
  // Only update if main path segment changes, preserve sub-routes
  useEffect(() => {
    const pathname = window.location.pathname.slice(1)
    const currentMainPath = pathname.split('/')[0]
    if (currentMainPath !== activeTab) {
      window.history.pushState(null, '', `/${activeTab}`)
    }
  }, [activeTab])

  // Listen for popstate (browser back/forward)
  useEffect(() => {
    const handlePopState = () => {
      const pathname = window.location.pathname.slice(1)
      const mainPath = pathname.split('/')[0] // Extract main tab from path
      if (validTabs.includes(mainPath as TabType)) {
        setActiveTab(mainPath as TabType)
      } else {
        setActiveTab('dashboard')
      }
    }
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  const handleWarmupReady = () => {
    setWarmupState('ready')
  }

  const ActiveTabComponent = useMemo(() => tabComponents[activeTab] ?? Dashboard, [activeTab])

  if (!isAuthenticated) {
    return <Login onLogin={login} />
  }

  // 正在检查预热状态
  if (warmupState === 'checking') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-4">
          <div className="relative">
            <div className="w-12 h-12 rounded-full border-4 border-primary/20 border-t-primary animate-spin" />
          </div>
          <p className="text-sm text-muted-foreground animate-pulse">正在连接服务器...</p>
        </div>
      </div>
    )
  }

  // 显示预热界面（后端还在预热中）
  if (warmupState === 'warming') {
    return <WarmupScreen onReady={handleWarmupReady} />
  }

  return (
    <Layout activeTab={activeTab} onTabChange={setActiveTab} onLogout={logout}>
      <Suspense
        fallback={
          <div className="min-h-[240px] flex items-center justify-center">
            <div className="flex flex-col items-center gap-4">
              <div className="w-10 h-10 rounded-full border-4 border-primary/20 border-t-primary animate-spin" />
              <p className="text-sm text-muted-foreground">正在加载页面...</p>
            </div>
          </div>
        }
      >
        <ActiveTabComponent />
      </Suspense>
    </Layout>
  )
}

export default App
