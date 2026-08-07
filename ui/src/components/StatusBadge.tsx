import { Badge } from "@/components/ui/badge"
import type { Translator } from "@/i18n"

export function StatusBadge({ enabled, t }: { enabled: boolean; t: Translator }) {
  return (
    <Badge
      variant="outline"
      className={enabled ? "border-success/30 bg-success-soft text-success" : "text-muted-foreground"}
    >
      {t(enabled ? "Enabled status" : "Disabled status")}
    </Badge>
  )
}
