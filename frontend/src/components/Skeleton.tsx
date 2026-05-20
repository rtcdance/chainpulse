export function Skeleton({ className = '' }: { className?: string }) {
  return (
    <div className={`animate-pulse rounded-2xl bg-white/5 ${className}`} />
  )
}

export function PageSkeleton() {
  return (
    <div className="space-y-6">
      <div className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <Skeleton className="mb-3 h-4 w-40" />
        <Skeleton className="mb-3 h-8 w-80" />
        <Skeleton className="h-4 w-full max-w-2xl" />
        <div className="mt-6 grid gap-4 md:grid-cols-3">
          <Skeleton className="h-16" />
          <Skeleton className="h-16" />
          <Skeleton className="h-16" />
        </div>
      </div>
      <div className="grid gap-6 xl:grid-cols-2">
        <div className="rounded-[28px] border border-white/10 bg-white/5 p-6">
          <Skeleton className="mb-4 h-5 w-32" />
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="mb-3 h-12" />
          ))}
        </div>
        <div className="rounded-[28px] border border-white/10 bg-white/5 p-6">
          <Skeleton className="mb-4 h-5 w-48" />
          <Skeleton className="h-64" />
        </div>
      </div>
    </div>
  )
}