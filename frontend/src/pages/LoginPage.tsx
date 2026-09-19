import { useEffect, useState } from 'react'
import { Alert, Button, Form, Input, Segmented, Typography } from 'antd'
import { LockKeyhole, Snowflake, UserRound } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'

const accounts = {
  process_analyst: 'analyst@lng.local',
  reviewer: 'reviewer@lng.local',
  admin: 'admin@lng.local'
}

export function LoginPage() {
  const { user, login, loading } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState(accounts.process_analyst)
  useEffect(() => {
    if (user) navigate('/balances', { replace: true })
  }, [navigate, user])
  const submit = async (values: { email: string; password: string }) => {
    await login(values.email, values.password)
    navigate('/balances', { replace: true })
  }
  return (
    <main className="login-shell">
      <section className="login-context" aria-label="LNG 物理质量平衡">
        <div className="cryogenic-mark"><Snowflake size={38} /></div>
        <div className="login-title">
          <span className="eyebrow">OFFLINE ENGINEERING ANALYSIS</span>
          <h1>LNG 物理质量平衡</h1>
          <p>液位、温度、密度与物理转移构成同一条可复核证据链。</p>
        </div>
        <div className="process-rule">
          <span>期初质量</span><b>+</b><span>流入</span><b>−</b><span>流出</span><b>−</b><span>期末质量</span>
        </div>
      </section>
      <section className="login-form-wrap">
        <div className="login-form">
          <Typography.Title level={2}>进入分析工作台</Typography.Title>
          <Typography.Paragraph type="secondary">选择职责身份并使用本地验证账号。</Typography.Paragraph>
          <Segmented
            block
            value={email}
            onChange={(value) => setEmail(String(value))}
            options={[
              { label: '工艺分析', value: accounts.process_analyst },
              { label: '独立复核', value: accounts.reviewer },
              { label: '管理', value: accounts.admin }
            ]}
          />
          <Form layout="vertical" onFinish={submit} key={email} initialValues={{ email, password: 'LngBalance!2026' }} requiredMark={false}>
            <Form.Item name="email" label="邮箱" rules={[{ required: true, type: 'email' }]}>
              <Input size="large" prefix={<UserRound size={17} />} autoComplete="username" />
            </Form.Item>
            <Form.Item name="password" label="密码" rules={[{ required: true, min: 8 }]}>
              <Input.Password size="large" prefix={<LockKeyhole size={17} />} autoComplete="current-password" />
            </Form.Item>
            <Button size="large" type="primary" htmlType="submit" loading={loading} block>登录</Button>
          </Form>
          <Alert className="login-boundary" type="info" showIcon message="结果仅作离线工程分析，不连接控制系统，也不用于贸易结算。" />
        </div>
      </section>
    </main>
  )
}
