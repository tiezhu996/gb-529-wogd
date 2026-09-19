import { Tag } from 'antd'
import { AlertTriangle, CheckCircle2, CircleX } from 'lucide-react'
import type { QualityFlag } from '../../types/measurement'

const labels: Record<QualityFlag, string> = { good: '良好', suspect: '可疑', invalid: '无效' }
const colors: Record<QualityFlag, string> = { good: 'success', suspect: 'warning', invalid: 'error' }
const icons = {
  good: <CheckCircle2 size={14} />,
  suspect: <AlertTriangle size={14} />,
  invalid: <CircleX size={14} />
}

export function QualityFlagBadge({ value }: { value: QualityFlag }) {
  return <Tag className="status-tag" color={colors[value]} icon={icons[value]}>{labels[value]}</Tag>
}
