import { Alert, Descriptions, Table, Tag } from 'antd'
import type { BalanceRun, UncertaintyComponent } from '../../types/balance'
import { deviationLabels } from '../../types/deviation'
import { kg, number } from '../../utils/format'

const sourceLabels: Record<string, string> = {
  opening_snapshot: '期初快照',
  closing_snapshot: '期末快照',
  transfer_inflow: '流入计量',
  transfer_outflow: '流出计量'
}

export function EvidenceBreakdownPanel({ run }: { run?: BalanceRun }) {
  if (!run) return null
  const evidence = run.evidence_json ?? {}
  const uncertainty = evidence.uncertainty
  return (
    <section className="evidence-panel" aria-label="平衡证据明细">
      <div className="section-heading">
        <div>
          <span className="eyebrow">EVIDENCE SNAPSHOT</span>
          <h2>证据与不确定度</h2>
        </div>
        <Tag color={run.deviation_level === 'investigate' ? 'warning' : 'success'}>
          {deviationLabels[run.deviation_level]}
        </Tag>
      </div>
      <Descriptions size="small" column={{ xs: 1, sm: 2, lg: 4 }} bordered>
        <Descriptions.Item label="算法版本">{evidence.algorithm_version ?? 'mass-balance-v1.0'}</Descriptions.Item>
        <Descriptions.Item label="系数版本">{run.coefficient_version}</Descriptions.Item>
        <Descriptions.Item label="合成不确定度">{kg(run.uncertainty_kg)}</Descriptions.Item>
        <Descriptions.Item label="偏差">{number.format(run.deviation_pct)}%</Descriptions.Item>
      </Descriptions>
      {uncertainty?.components?.length ? (
        <Table<UncertaintyComponent>
          className="evidence-table"
          rowKey={(item) => item.source + item.entity_id}
          size="small"
          pagination={false}
          dataSource={uncertainty.components}
          columns={[
            { title: '来源', dataIndex: 'source', render: (value: string) => sourceLabels[value] ?? value },
            { title: '证据 ID', dataIndex: 'entity_id' },
            { title: '质量', dataIndex: 'mass_kg', align: 'right', render: kg },
            { title: '不确定度', dataIndex: 'uncertainty_pct', align: 'right', render: (value: number) => number.format(value) + '%' },
            { title: '绝对贡献', dataIndex: 'absolute_kg', align: 'right', render: kg }
          ]}
        />
      ) : null}
      <Alert type="warning" showIcon message={evidence.safety_boundary ?? '未解释差异仅作工程分析，不直接认定为泄漏或安全事件。'} />
    </section>
  )
}
