import { useEffect, useState } from 'react'
import { Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Table, Tag } from 'antd'
import { Check, Plus, RefreshCw, X } from 'lucide-react'
import { PageHeader } from '../components/common/PageHeader'
import { useAuth } from '../hooks/useAuth'
import { useTankStore } from '../stores/tankStore'
import { useTransferStore } from '../stores/transferStore'
import type { OperationStatus, OperationType, TransferInput, TransferOperation } from '../types/transfer'
import { dateTime, kg, localInputDate, number } from '../utils/format'

const statusLabels: Record<OperationStatus, string> = { draft: '草稿', confirmed: '已确认', cancelled: '已取消' }

export function TransfersPage() {
  const { can } = useAuth()
  const store = useTransferStore()
  const tanks = useTankStore()
  const [open, setOpen] = useState(false)
  const [form] = Form.useForm<TransferInput>()
  useEffect(() => { void Promise.all([store.load(), tanks.load()]) }, [])
  const openCreate = () => {
    form.setFieldsValue({ tank_id: tanks.items[0]?.id })
    setOpen(true)
  }
  const create = async (values: TransferInput) => {
    await store.create({
      ...values,
      start_at: new Date(values.start_at).toISOString(),
      end_at: new Date(values.end_at).toISOString()
    })
    setOpen(false)
    form.resetFields()
  }
  return (
    <>
      <PageHeader
        eyebrow="PHYSICAL MOVEMENTS"
        title="物理转移"
        description="记录平衡期间的实际流入与流出；同罐未取消时间段不允许重叠。"
        actions={
          <>
            <Button icon={<RefreshCw size={16} />} onClick={() => void store.load()}>刷新</Button>
            {can('process_analyst', 'admin') && <Button type="primary" icon={<Plus size={16} />} onClick={openCreate}>登记转移</Button>}
          </>
        }
      />
      <section className="data-panel">
        <div className="section-heading"><h2>期间流量证据</h2><span>{store.items.filter((item) => item.operation_status === 'confirmed').length} 条已确认</span></div>
        <Table<TransferOperation>
          rowKey="id"
          loading={store.loading}
          dataSource={store.items}
          pagination={{ pageSize: 12, showSizeChanger: false }}
          scroll={{ x: 1050 }}
          columns={[
            { title: '储罐', key: 'tank', fixed: 'left', width: 110, render: (_, item) => item.tank?.tank_code ?? item.tank_id },
            { title: '方向', dataIndex: 'operation_type', width: 90, render: (value: OperationType) => <Tag color={value === 'inflow' ? 'cyan' : 'gold'}>{value === 'inflow' ? '流入' : '流出'}</Tag> },
            { title: '开始', dataIndex: 'start_at', render: dateTime },
            { title: '结束', dataIndex: 'end_at', render: dateTime },
            { title: '计量质量', dataIndex: 'measured_mass_kg', align: 'right', render: kg },
            { title: '不确定度', dataIndex: 'measurement_uncertainty_pct', align: 'right', render: (value: number) => number.format(value) + '%' },
            { title: '物理参考', dataIndex: 'counterparty_ref' },
            { title: '状态', dataIndex: 'operation_status', render: (value: OperationStatus) => <Tag color={value === 'confirmed' ? 'success' : value === 'cancelled' ? 'default' : 'processing'}>{statusLabels[value]}</Tag> },
            {
              title: '操作', key: 'action', fixed: 'right', width: 150,
              render: (_, item) => can('process_analyst', 'admin') && item.operation_status !== 'cancelled' ? (
                <div className="table-actions">
                  {item.operation_status === 'draft' && <Button size="small" type="text" icon={<Check size={15} />} onClick={() => void store.confirm(item)} title="确认物理转移">确认</Button>}
                  <Popconfirm title="取消该物理转移？" onConfirm={() => void store.cancel(item, '人工复核后取消该物理转移')}><Button size="small" type="text" danger icon={<X size={15} />} title="取消物理转移">取消</Button></Popconfirm>
                </div>
              ) : null
            }
          ]}
        />
      </section>
      <Modal title="登记物理转移" open={open} onCancel={() => setOpen(false)} footer={null} destroyOnClose>
        <Form<TransferInput>
          form={form}
          layout="vertical"
          onFinish={create}
          requiredMark={false}
          initialValues={{
            operation_type: 'inflow', start_at: localInputDate(new Date(Date.now() - 3_600_000)),
            end_at: localInputDate(new Date()), measured_mass_kg: 100000,
            measurement_uncertainty_pct: 0.25, counterparty_ref: 'METER-REFERENCE',
            operation_status: 'draft'
          }}
        >
          <div className="form-grid">
            <Form.Item className="span-2" name="tank_id" label="储罐" rules={[{ required: true }]}><Select options={tanks.items.map((tank) => ({ value: tank.id, label: tank.tank_code + ' · ' + tank.name }))} /></Form.Item>
            <Form.Item name="operation_type" label="方向" rules={[{ required: true }]}><Select options={[{ value: 'inflow', label: '流入' }, { value: 'outflow', label: '流出' }]} /></Form.Item>
            <Form.Item name="operation_status" label="初始状态" rules={[{ required: true }]}><Select options={[{ value: 'draft', label: '草稿' }, { value: 'confirmed', label: '已确认' }]} /></Form.Item>
            <Form.Item name="start_at" label="开始时间" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item>
            <Form.Item name="end_at" label="结束时间" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item>
            <Form.Item name="measured_mass_kg" label="计量质量 (kg)" rules={[{ required: true }]}><InputNumber min={1} /></Form.Item>
            <Form.Item name="measurement_uncertainty_pct" label="不确定度 (%)" rules={[{ required: true }]}><InputNumber min={0.01} max={10} step={0.01} /></Form.Item>
            <Form.Item className="span-2" name="counterparty_ref" label="物理计量参考" rules={[{ required: true, min: 3 }]}><Input /></Form.Item>
          </div>
          <Button type="primary" htmlType="submit" block>保存物理转移</Button>
        </Form>
      </Modal>
    </>
  )
}
