import { create } from 'zustand'
import * as api from '../api/tanks'
import type { StorageTank, TankInput } from '../types/tank'

interface TankState {
  items: StorageTank[]
  selectedId: number | null
  loading: boolean
  load: () => Promise<void>
  select: (id: number) => void
  create: (input: TankInput) => Promise<StorageTank>
}

export const useTankStore = create<TankState>((set) => ({
  items: [],
  selectedId: null,
  loading: false,
  load: async () => {
    set({ loading: true })
    try {
      const result = await api.listTanks()
      set((state) => ({ items: result.items, selectedId: state.selectedId ?? result.items[0]?.id ?? null }))
    } finally {
      set({ loading: false })
    }
  },
  select: (id) => set({ selectedId: id }),
  create: async (input) => {
    const created = await api.createTank(input)
    set((state) => ({ items: [created, ...state.items], selectedId: created.id }))
    return created
  }
}))
