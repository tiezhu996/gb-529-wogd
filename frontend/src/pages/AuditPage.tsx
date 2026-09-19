import { useEffect, useState } from 'react'
import { Button, Table, Tag } from 'antd'
import { RefreshCw } from 'lucide-react'
import { listAudits } from '../api/audits'
import { PageHeader } from '../components/common/PageHeader'
import type { AuditEvent } from '../types/audit'
import { dateTime } from '../utils/format'

export function AuditPage() {
  const [items, setItems] = useState<AuditEvent[]>([])
  const [loading, setLoading] = useState(false)
  const load = async () => {
    setLoading(true)
    try {
      const result = await listAudits()
      setItems(result.items)
    } finally {
      setLoading(false)
    }
  }
  useEffect(() => { void load() }, [])
  return (
    <>
      <PageHeader
        eyebrow="IMMUTABLE AUDIT TRAIL"
        title="审计复核"
        description="按 request ID 追溯参数、快照、转移、平衡运行及独立复核的前后摘要。"
        actions={<Button icon={<RefreshCw size={16} />} onClick={() => void load()}>刷新</Button>}
      />
      <section className="data-panel">
        <div className="section-heading"><h2>操作证据链</h2><span>{items.length} 个审计事件</span></div>
        <Table<AuditEvent>
          rowKey="id"
          loading={loading}
          dataSource={items}
          pagination={{ pageSize: 12, showSizeChanger: false }}
          scroll={{ x: 1000 }}
          expandable={{
            expandedRowRender: (item) => (
              <div className="audit-diff">
                <div><span>变更前</span><pre>{JSON.stringify(JSON.parse(item.before_json || '{}'), null, 2)}</pre></div>
                <div><span>变更后</span><pre>{JSON.stringify(JSON.parse(item.after_json || '{}'), null, 2)}</pre></div>
              </div>
            )
          }}
          columns={[
            { title: '时间', dataIndex: 'created_at', width: 132, render: dateTime },
            { title: '操作者', dataIndex: 'actor_email' },
            { title: '动作', dataIndex: 'action', render: (value: string) => <Tag color={value.includes('accepted') ? 'success' : value.includes('rejected') ? 'warning' : 'default'}>{value}</Tag> },
            { title: '实体', key: 'entity', render: (_, item) => item.entity_type + ' #' + item.entity_id },
            { title: 'Request ID', dataIndex: 'request_id', ellipsis: true, className: 'mono' }
          ]}
        />
      </section>
    </>
  )
}
