import ReactECharts from 'echarts-for-react'
import { Empty } from 'antd'
import type { BalanceRun } from '../../types/balance'
import { compactNumber } from '../../utils/format'

interface Props {
  run?: BalanceRun
  height?: number
}

export function MassBalanceWaterfall({ run, height = 310 }: Props) {
  if (!run) return <div className="chart-empty"><Empty description="选择一条平衡运行查看质量边界" /></div>
  const values = [run.opening_mass_kg, run.net_transfer_kg, -run.closing_mass_kg, run.estimated_bog_kg]
  const colors = ['#147d75', run.net_transfer_kg >= 0 ? '#3389a5' : '#b9771f', '#6c7a82', Math.abs(run.estimated_bog_kg) <= run.uncertainty_kg ? '#2d8a60' : '#c47822']
  return (
    <ReactECharts
      style={{ height, width: '100%' }}
      option={{
        animationDuration: 420,
        grid: { left: 72, right: 24, top: 30, bottom: 58 },
        tooltip: { trigger: 'axis', valueFormatter: (value: number) => compactNumber.format(value) + ' kg' },
        xAxis: {
          type: 'category',
          data: ['期初液相', '净转移', '期末扣减', 'BOG / 未解释项'],
          axisLabel: { color: '#45545b', interval: 0 }
        },
        yAxis: {
          type: 'value',
          name: '质量 / kg',
          nameTextStyle: { color: '#66767c' },
          axisLabel: { formatter: (value: number) => compactNumber.format(value), color: '#66767c' },
          splitLine: { lineStyle: { color: '#e3e9e9' } }
        },
        series: [{
          type: 'bar',
          data: values.map((value, index) => ({ value, itemStyle: { color: colors[index], borderRadius: [3, 3, 0, 0] } })),
          barMaxWidth: 58,
          label: { show: true, position: 'top', formatter: ({ value }: { value: number }) => compactNumber.format(value), color: '#26343a' },
          markArea: {
            silent: true,
            itemStyle: { color: 'rgba(45, 138, 96, 0.08)' },
            data: [[
              { xAxis: 'BOG / 未解释项', yAxis: run.interval_lower_kg },
              { xAxis: 'BOG / 未解释项', yAxis: run.interval_upper_kg }
            ]]
          }
        }]
      }}
      opts={{ renderer: 'canvas' }}
      notMerge
    />
  )
}
