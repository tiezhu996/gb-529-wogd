import { useEffect, type ReactNode } from 'react'
import { Button, Result, Spin, Tooltip } from 'antd'
import { ArrowLeftRight, Database, FileClock, LogOut, Scale, Snowflake, Thermometer } from 'lucide-react'
import { NavLink, Navigate, Outlet, createBrowserRouter, useLocation } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import { AuditPage } from '../pages/AuditPage'
import { BalancesPage } from '../pages/BalancesPage'
import { LoginPage } from '../pages/LoginPage'
import { MeasurementsPage } from '../pages/MeasurementsPage'
import { TanksPage } from '../pages/TanksPage'
import { TransfersPage } from '../pages/TransfersPage'
import type { UserRole } from '../types/auth'

const roleLabels: Record<UserRole, string> = {
  process_analyst: '工艺分析员',
  reviewer: '独立复核员',
  admin: '管理员'
}

function Protected({ children, roles }: { children: ReactNode; roles?: UserRole[] }) {
  const { initialized, loading, user, bootstrap } = useAuth()
  const location = useLocation()
  useEffect(() => { void bootstrap() }, [bootstrap])
  if (!initialized || loading) return <div className="route-loading"><Spin size="large" /></div>
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  if (roles && !roles.includes(user.role)) {
    return <Result status="403" title="无权访问" subTitle="该页面仅向独立复核员与管理员开放。" extra={<Button href="/balances">返回平衡工作台</Button>} />
  }
  return <>{children}</>
}

function WorkspaceLayout() {
  const { user, logout } = useAuth()
  const links = [
    { to: '/tanks', label: '储罐', icon: Database },
    { to: '/measurements', label: '计量', icon: Thermometer },
    { to: '/transfers', label: '转移', icon: ArrowLeftRight },
    { to: '/balances', label: '平衡', icon: Scale },
    ...(user?.role === 'reviewer' || user?.role === 'admin' ? [{ to: '/audit', label: '审计', icon: FileClock }] : [])
  ]
  return (
    <div className="workspace-shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-symbol"><Snowflake size={24} /></span>
          <div><strong>LNG Balance</strong><span>物理质量分析</span></div>
        </div>
        <nav aria-label="业务导航">
          {links.map(({ to, label, icon: Icon }) => (
            <NavLink key={to} to={to} className={({ isActive }) => isActive ? 'active' : undefined}>
              <Icon size={19} /><span>{label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-boundary">OFFLINE<br />NO CONTROL OUTPUT</div>
      </aside>
      <div className="workspace-body">
        <header className="topbar">
          <div className="system-state"><span className="state-dot" />离线计算环境</div>
          <div className="identity">
            <div><strong>{user?.display_name}</strong><span>{user ? roleLabels[user.role] : ''}</span></div>
            <Tooltip title="退出登录"><Button type="text" shape="circle" icon={<LogOut size={17} />} onClick={logout} aria-label="退出登录" /></Tooltip>
          </div>
        </header>
        <main className="workspace-content"><Outlet /></main>
      </div>
    </div>
  )
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/',
    element: <Protected><WorkspaceLayout /></Protected>,
    children: [
      { index: true, element: <Navigate to="/balances" replace /> },
      { path: 'tanks', element: <TanksPage /> },
      { path: 'measurements', element: <MeasurementsPage /> },
      { path: 'transfers', element: <TransfersPage /> },
      { path: 'balances', element: <BalancesPage /> },
      { path: 'audit', element: <Protected roles={['reviewer', 'admin']}><AuditPage /></Protected> }
    ]
  },
  { path: '*', element: <Navigate to="/" replace /> }
])
