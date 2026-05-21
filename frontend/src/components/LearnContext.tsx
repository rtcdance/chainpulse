interface LearnContextProps {
  title: string
  concept: string
  codePath: string
  debugTip?: string
  children?: React.ReactNode
}

export default function LearnContext({ title, concept, codePath, debugTip, children }: LearnContextProps) {
  return (
    <details className="group mb-6 overflow-hidden rounded-2xl border border-amber-300/20 bg-amber-300/5">
      <summary className="flex cursor-pointer items-center gap-2 px-5 py-3 text-sm font-medium text-amber-200 hover:text-amber-100">
        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-amber-300/15 text-xs">?</span>
        {title}
      </summary>
      <div className="border-t border-amber-300/10 px-5 py-4 text-sm leading-6 text-sand/80 space-y-3">
        <p>{concept}</p>
        <div className="flex flex-wrap gap-3 text-xs">
          <code className="rounded-md bg-ink/80 px-3 py-1.5 font-mono text-amber-200/90">{codePath}</code>
          {debugTip && (
            <span className="rounded-md bg-sky-400/10 px-3 py-1.5 text-sky-300">
              ▶ 调试: {debugTip}
            </span>
          )}
        </div>
        {children}
      </div>
    </details>
  )
}
