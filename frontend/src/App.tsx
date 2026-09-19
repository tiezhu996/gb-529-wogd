import { Component, useEffect, type ErrorInfo, type ReactNode } from 'react'
import { App as AntApp, ConfigProvider, Result, message } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { RouterProvider } from 'react-router-dom'
import { router } from './router'

interface ApiErrorDetail {
  message?: string
  code?: string
  details?: Record<string, unknown>
  requestId?: string
}

function describeApiError(detail: string | ApiErrorDetail): string {
  if (typeof detail === 'string') return detail
  const parts = [detail.message]
  if (detail.details) {
    const transferID = detail.details.transfer_id
    const crossedAt = detail.details.crossed_at
    if (transferID !== undefined) parts.push(`转移编号 #${String(transferID)}`)
    if (typeof crossedAt === 'string') parts.push(`越界时刻 ${crossedAt}`)
  }
  if (detail.requestId) parts.push(`请求 ${detail.requestId}`)
  return parts.filter(Boolean).join(' · ')
}

function ApiMessages() {
  const [messageApi, contextHolder] = message.useMessage()
  useEffect(() => {
    const listener = (event: Event) => {
      const detail = (event as CustomEvent<string | ApiErrorDetail>).detail
      void messageApi.error(describeApiError(detail), 6)
    }
    window.addEventListener('api:error', listener)
    return () => window.removeEventListener('api:error', listener)
  }, [messageApi])
  return <>{contextHolder}<RouterProvider router={router} /></>
}

export function App() {
  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: '#147d75',
          colorInfo: '#277f99',
          colorSuccess: '#2d8a60',
          colorWarning: '#b9771f',
          colorError: '#b6473e',
          colorText: '#26343a',
          colorTextSecondary: '#66767c',
          colorBgBase: '#f7f9f9',
          colorBorder: '#d8e0e1',
          borderRadius: 4,
          fontFamily: 'Inter, "Noto Sans SC", "Microsoft YaHei", sans-serif'
        },
        components: {
          Button: { controlHeight: 38, primaryShadow: 'none' },
          Table: { headerBg: '#eef3f3', headerColor: '#3d4d53', rowHoverBg: '#edf7f5' },
          Modal: { borderRadiusLG: 6 },
          Tag: { borderRadiusSM: 3 }
        }
      }}
    >
      <AntApp><ErrorBoundary><ApiMessages /></ErrorBoundary></AntApp>
    </ConfigProvider>
  )
}

class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false }
  static getDerivedStateFromError() { return { failed: true } }
  componentDidCatch(error: Error, info: ErrorInfo) { console.error('interface_error', error, info.componentStack) }
  render() {
    if (!this.state.failed) return this.props.children
    return <Result status="error" title="界面无法继续渲染" subTitle="数据未被修改，请重新加载分析工作台。" extra={<button onClick={() => window.location.reload()}>重新加载</button>} />
  }
}
