import { useEffect, useState } from 'react'
import { finalityLabel } from '../lib/finality'

export default function FinalityStatus({ raw }: { raw: Record<string, unknown> }) {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [])
  return <>{finalityLabel(raw, now)}</>
}
