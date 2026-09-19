import { useCallback, useState } from 'react'
import { getBalance } from '../api/balances'
import { useBalanceStore } from '../stores/balanceStore'
import type { BalanceRun, BalanceRunInput, BalanceStatus } from '../types/balance'

const pause = (milliseconds: number) => new Promise((resolve) => window.setTimeout(resolve, milliseconds))

export function useBalanceRun() {
  const store = useBalanceStore()
  const [working, setWorking] = useState(false)

  const pollUntilCalculated = useCallback(async (id: number) => {
    for (let attempt = 0; attempt < 6; attempt += 1) {
      const current = await getBalance(id)
      if (current.balance_status !== 'queued') {
        await store.load()
        return current
      }
      await pause(800)
    }
    await store.load()
    return getBalance(id)
  }, [store.load])

  const run = useCallback(async (input: BalanceRunInput) => {
    setWorking(true)
    try {
      const created = await store.run(input)
      return await pollUntilCalculated(created.id)
    } finally {
      setWorking(false)
    }
  }, [pollUntilCalculated, store.run])

  const submit = useCallback(async (item: BalanceRun) => {
    setWorking(true)
    try {
      return await store.submit(item)
    } finally {
      setWorking(false)
    }
  }, [store.submit])

  const review = useCallback(async (item: BalanceRun, target: Extract<BalanceStatus, 'accepted' | 'rejected'>, note: string) => {
    setWorking(true)
    try {
      return await store.review(item, target, note)
    } finally {
      setWorking(false)
    }
  }, [store.review])

  return { ...store, working, run, submit, review }
}
