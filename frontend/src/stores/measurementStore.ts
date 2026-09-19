import { create } from 'zustand'
import * as api from '../api/measurements'
import type { MeasurementInput, MeasurementSnapshot } from '../types/measurement'

interface MeasurementState {
  items: MeasurementSnapshot[]
  loading: boolean
  load: (tankId?: number) => Promise<void>
  create: (input: MeasurementInput) => Promise<MeasurementSnapshot>
}

export const useMeasurementStore = create<MeasurementState>((set) => ({
  items: [],
  loading: false,
  load: async (tankId) => {
    set({ loading: true })
    try {
      const result = await api.listMeasurements(tankId)
      set({ items: result.items })
    } finally {
      set({ loading: false })
    }
  },
  create: async (input) => {
    const created = await api.createMeasurement(input)
    set((state) => ({ items: [created, ...state.items] }))
    return created
  }
}))
