import { create } from 'zustand'
import * as api from '../api/transfers'
import type { TransferInput, TransferOperation } from '../types/transfer'

interface TransferState {
  items: TransferOperation[]
  loading: boolean
  load: (tankId?: number) => Promise<void>
  create: (input: TransferInput) => Promise<TransferOperation>
  confirm: (item: TransferOperation) => Promise<void>
  cancel: (item: TransferOperation, reason: string) => Promise<void>
}

export const useTransferStore = create<TransferState>((set) => ({
  items: [],
  loading: false,
  load: async (tankId) => {
    set({ loading: true })
    try {
      const result = await api.listTransfers(tankId)
      set({ items: result.items })
    } finally {
      set({ loading: false })
    }
  },
  create: async (input) => {
    const created = await api.createTransfer(input)
    set((state) => ({ items: [created, ...state.items] }))
    return created
  },
  confirm: async (item) => {
    const updated = await api.transitionTransfer(item.id, item.version, 'confirmed')
    set((state) => ({ items: state.items.map((candidate) => candidate.id === updated.id ? updated : candidate) }))
  },
  cancel: async (item, reason) => {
    const updated = await api.transitionTransfer(item.id, item.version, 'cancelled', reason)
    set((state) => ({ items: state.items.map((candidate) => candidate.id === updated.id ? updated : candidate) }))
  }
}))
