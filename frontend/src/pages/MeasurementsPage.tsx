import { useEffect, useState } from 'react'
import { Button, Form, Input, InputNumber, Modal, Select, Table } from 'antd'
import { Plus, RefreshCw } from 'lucide-react'
import { PageHeader } from '../components/common/PageHeader'
import { QualityFlagBadge } from '../components/common/QualityFlagBadge'
import { useAuth } from '../hooks/useAuth'
import { useMeasurementStore } from '../stores/measurementStore'
import { useTankStore } from '../stores/tankStore'
import type { MeasurementInput, MeasurementSnapshot, QualityFlag } from '../types/measurement'
import { cubicMeters, dateTime, kg, localInputDate, number } from '../utils/format'

export function MeasurementsPage() {
  const { can } = useAuth()
  const measurementStore = useMeasurementStore()
  const tankStore = useTankStore()
  const [open, setOpen] = useState(false)
  const [form] = Form.useForm<MeasurementInput>()
  useEffect(() => { void Promise.all([measurementStore.load(), tankStore.load()]) }, [])
  const create = async (values: MeasurementInput) => {
    await measurementStore.create({ ...values, measured_at: new Date(values.measured_at).toISOString() })
    setOpen(false)
    form.resetFields()
  }
  return (
    <>
      <PageHeader
        eyebrow="IMMUTABLE MEASUREMENTS"
        title="计量快照"
        description="原始液位、温度、压力和密度只增不改，计算质量与数据质量标记一并固化。"
        actions={
          <>
            <Button icon={<RefreshCw size={16} />} onClick={() => void measurementStore.load()}>刷新</Button>
            {can('process_analyst', 'admin') && <Button type="primary" icon={<Plus size={16} />} onClick={() => setOpen(true)}>录入快照</Button>}
          </>
        }
      />
      <section className="data-panel">
        <div className="section-heading"><h2>离线计量序列</h2><span>{measurementStore.items.length} 条不可变记录</span></div>
        <Table<MeasurementSnapshot>
          rowKey="id"
          loading={measurementStore.loading}
          dataSource={measurementStore.items}
          pagination={{ pageSize: 12, showSizeChanger: false }}
          scroll={{ x: 1180 }}
          columns={[
            { title: '时间', dataIndex: 'measured_at', fixed: 'left', width: 132, render: dateTime },
            { title: '储罐', key: 'tank', width: 110, render: (_, item) => item.tank?.tank_code ?? item.tank_id },
            { title: '质量', dataIndex: 'quality_flag', width: 94, render: (value: QualityFlag) => <QualityFlagBadge value={value} /> },
            { title: '液位', dataIndex: 'liquid_level_m', align: 'right', render: (value: number) => number.format(value) + ' m' },
            { title: '液温', dataIndex: 'liquid_temp_c', align: 'right', render: (value: number) => number.format(value) + ' °C' },
            { title: '汽相压力', dataIndex: 'vapor_pressure_kpa', align: 'right', render: (value: number) => number.format(value) + ' kPa' },
            { title: '修正密度', dataIndex: 'temperature_density_kgm3', align: 'right', render: (value: number) => number.format(value) + ' kg/m³' },
            { title: '罐容', dataIndex: 'calculated_volume_m3', align: 'right', render: cubicMeters },
            { title: '液相质量', dataIndex: 'calculated_liquid_mass_kg', align: 'right', render: kg },
            { title: '不确定度', dataIndex: 'measurement_uncertainty_pct', align: 'right', render: (value: number) => number.format(value) + '%' },
            { title: '来源说明', dataIndex: 'source_note', ellipsis: true }
          ]}
        />
      </section>
      <Modal title="录入不可变计量快照" open={open} onCancel={() => setOpen(false)} footer={null} destroyOnClose>
        <Form<MeasurementInput>
          form={form}
          layout="vertical"
          onFinish={create}
          requiredMark={false}
          initialValues={{
            tank_id: tankStore.items[0]?.id,
            measured_at: localInputDate(new Date()),
            liquid_level_m: 8,
            liquid_temp_c: -160,
            vapor_pressure_kpa: 110,
            density_kgm3: 450,
            measurement_uncertainty_pct: 0.3,
            quality_flag: 'good',
            source_note: '离线计量经班次复核'
          }}
        >
          <div className="form-grid">
            <Form.Item className="span-2" name="tank_id" label="储罐" rules={[{ required: true }]}><Select options={tankStore.items.map((tank) => ({ value: tank.id, label: tank.tank_code + ' · ' + tank.name }))} /></Form.Item>
            <Form.Item className="span-2" name="measured_at" label="计量时间" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item>
            <Form.Item name="liquid_level_m" label="液位 (m)" rules={[{ required: true }]}><InputNumber min={0} step={0.01} /></Form.Item>
            <Form.Item name="liquid_temp_c" label="液温 (°C)" rules={[{ required: true }]}><InputNumber min={-200} max={-100} step={0.1} /></Form.Item>
            <Form.Item name="vapor_pressure_kpa" label="汽相压力 (kPa)" rules={[{ required: true }]}><InputNumber min={0} step={0.1} /></Form.Item>
            <Form.Item name="density_kgm3" label="密度 (kg/m³)" rules={[{ required: true }]}><InputNumber min={350} max={550} step={0.1} /></Form.Item>
            <Form.Item name="measurement_uncertainty_pct" label="不确定度 (%)" rules={[{ required: true }]}><InputNumber min={0.01} max={10} step={0.01} /></Form.Item>
            <Form.Item name="quality_flag" label="质量标记" rules={[{ required: true }]}><Select options={[{ value: 'good', label: '良好' }, { value: 'suspect', label: '可疑' }, { value: 'invalid', label: '无效' }]} /></Form.Item>
            <Form.Item className="span-2" name="source_note" label="来源说明" rules={[{ required: true, min: 3 }]}><Input.TextArea rows={2} /></Form.Item>
          </div>
          <Button type="primary" htmlType="submit" block>写入快照</Button>
        </Form>
      </Modal>
    </>
  )
}
