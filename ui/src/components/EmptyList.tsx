export function EmptyList({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="grid min-h-32 place-content-center gap-1 rounded-lg border border-dashed px-4 text-center">
      <strong className="text-sm font-medium">{title}</strong>
      <span className="text-sm text-muted-foreground">{detail}</span>
    </div>
  )
}
