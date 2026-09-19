export type DeviationLevel = 'within_uncertainty' | 'watch' | 'investigate' | 'invalid'

export const deviationLabels: Record<DeviationLevel, string> = {
  within_uncertainty: '不确定度内',
  watch: '关注',
  investigate: '需调查',
  invalid: '无效'
}
