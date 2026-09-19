import { useEffect, useMemo, useState } from 'react'
import { Button, Form, Input, InputNumber, Modal, Select, Table, Tag } from 'antd'
import { Plus, RefreshCw } from 'lucide-react'
import { MassBalanceWaterfall } from '../components/common/MassBalanceWaterfall'
import { PageHeader } from '../components/common/PageHeader'
import { useAuth } from '../hooks/useAuth'
import { useBalanceStore } from '../stores/balanceStore'
import { useTankStore } from '../stores/tankStore'
import type { StorageTank, TankInput, TankStatus } from '../types/tank'
import { cubicMeters, number } from '../utils/format'

interface TankForm extends Omit<TankInput, 'capacity_curve'> {
  capacity_curve_text: string
}

const statusLabel: Record<TankStatus, string> = { active: '启用', calibration_due: '待校准', inactive: '停用' }

export function TanksPage() {
  const { can } = useAuth()
  const tankStore = useTankStore()
  const balanceStore = useBalanceStore()
  const [open, setOpen] = useState(false)
  const [form] = Form.useForm<TankForm>()
  useEffect(() => { void Promise.all([tankStore.load(), balanceStore.load()]) }, [])
  const selected = tankStore.items.find((item) => item.id === tankStore.selectedId)
  const selectedRun = useMemo(
    () => balanceStore.items.find((item) => item.tank_id === tankStore.selectedId),
    [balanceStore.items, tankStore.selectedId]
  )
  const create = async (values: TankForm) => {
    const coefficients = values.capacity_curve_text.split(',').map((value) => Number(value.trim()))
    await tankStore.create({ ...values, capacity_curve: coefficients })
    setOpen(false)
    form.resetFields()
  }
  return (
    <>
      <PageHeader
        eyebrow="CALCULATION BOUNDARIES"
        title="储罐参数"
        description="维护罐容曲线、温度修正和有效液位边界；版本变化不会覆盖既有运行快照。"
        actions={
          <>
            <Button icon={<RefreshCw size={16} />} onClick={() => void tankStore.load()}>刷新</Button>
            {can('process_analyst', 'admin') && <Button type="primary" icon={<Plus size={16} />} onClick={() => setOpen(true)}>登记储罐</Button>}
          </>
        }
      />
      <section className="split-workspace">
        <div className="data-panel">
          <div className="section-heading"><h2>计算边界</h2><span>{tankStore.items.length} 座储罐</span></div>
          <Table<StorageTank>
            rowKey="id"
            loading={tankStore.loading}
            dataSource={tankStore.items}
            pagination={false}
            scroll={{ x: 920 }}
            rowClassName={(item) => item.id === tankStore.selectedId ? 'selected-row' : ''}
            onRow={(item) => ({ onClick: () => tankStore.select(item.id) })}
            columns={[
              { title: '储罐', key: 'tank', fixed: 'left', width: 190, render: (_, item) => <><strong>{item.tank_code}</strong><div className="secondary">{item.name}</div></> },
              { title: '状态', dataIndex: 'tank_status', width: 96, render: (value: TankStatus) => <Tag color={value === 'active' ? 'success' : value === 'calibration_due' ? 'warning' : 'default'}>{statusLabel[value]}</Tag> },
              { title: '名义容积', dataIndex: 'nominal_capacity_m3', align: 'right', render: cubicMeters },
              { title: '液位范围', key: 'level', align: 'right', render: (_, item) => number.format(item.min_level_m) + '–' + number.format(item.max_level_m) + ' m' },
              { title: '参考密度', dataIndex: 'reference_density_kgm3', align: 'right', render: (value: number) => number.format(value) + ' kg/m³' },
              { title: '系数版本', dataIndex: 'coefficient_version' },
              { title: '版本', dataIndex: 'version', align: 'center', width: 72 }
            ]}
          />
        </div>
        <aside className="insight-panel">
          <div className="section-heading"><h2>{selected?.tank_code ?? '质量边界'}</h2><span>{selectedRun ? '最近运行' : '暂无运行'}</span></div>
          <MassBalanceWaterfall run={selectedRun} height={300} />
          {selected && (
            <dl className="metric-list">
              <div><dt>参考温度</dt><dd>{number.format(selected.reference_temperature_c)} °C</dd></div>
              <div><dt>体膨胀系数</dt><dd>{selected.thermal_expansion_per_c}</dd></div>
              <div><dt>曲线阶数</dt><dd>{selected.capacity_curve_json.coefficients.length - 1}</dd></div>
            </dl>
          )}
        </aside>
      </section>
      <Modal title="登记计算边界" open={open} onCancel={() => setOpen(false)} footer={null} destroyOnClose>
        <Form<TankForm>
          form={form}
          layout="vertical"
          onFinish={create}
          requiredMark={false}
          initialValues={{
            nominal_capacity_m3: 180000, min_level_m: 0, max_level_m: 12,
            reference_density_kgm3: 450, reference_temperature_c: -160,
            thermal_expansion_per_c: 0.0035, capacity_curve_text: '0,15000',
            coefficient_version: 'CV-2026.08', tank_status: 'active'
          }}
        >
          <div className="form-grid">
            <Form.Item name="tank_code" label="储罐编号" rules={[{ required: true }]}><Input /></Form.Item>
            <Form.Item name="name" label="名称" rules={[{ required: true }]}><Input /></Form.Item>
            <Form.Item name="nominal_capacity_m3" label="名义容积 (m³)" rules={[{ required: true }]}><InputNumber min={1} /></Form.Item>
            <Form.Item name="max_level_m" label="最高液位 (m)" rules={[{ required: true }]}><InputNumber min={0.1} /></Form.Item>
            <Form.Item name="min_level_m" label="最低液位 (m)" rules={[{ required: true }]}><InputNumber min={0} /></Form.Item>
            <Form.Item name="reference_density_kgm3" label="参考密度 (kg/m³)" rules={[{ required: true }]}><InputNumber min={350} max={550} /></Form.Item>
            <Form.Item name="reference_temperature_c" label="参考温度 (°C)" rules={[{ required: true }]}><InputNumber min={-200} max={-100} /></Form.Item>
            <Form.Item name="thermal_expansion_per_c" label="体膨胀系数" rules={[{ required: true }]}><InputNumber min={0.0001} max={0.01} step={0.0001} /></Form.Item>
            <Form.Item className="span-2" name="capacity_curve_text" label="罐容多项式系数（常数项起，逗号分隔）" rules={[{ required: true }]}><Input /></Form.Item>
            <Form.Item name="coefficient_version" label="系数版本" rules={[{ required: true }]}><Input /></Form.Item>
            <Form.Item name="tank_status" label="状态" rules={[{ required: true }]}><Select options={Object.entries(statusLabel).map(([value, label]) => ({ value, label }))} /></Form.Item>
          </div>
          <Button type="primary" htmlType="submit" block>保存计算边界</Button>
        </Form>
      </Modal>
    </>
  )
}
