import { Plus } from "lucide-react"

import { EmptyList } from "@/components/EmptyList"
import { EntityActions } from "@/components/EntityActions"
import { StatusBadge } from "@/components/StatusBadge"
import { Button } from "@/components/ui/button"
import type { Translator } from "@/i18n"
import type { ProxyService } from "@/types"

interface ServicesPanelProps {
  services: ProxyService[]
  busy: boolean
  t: Translator
  onAdd: () => void
  onEdit: (service: ProxyService) => void
  onDelete: (service: ProxyService) => void
}

export function ServicesPanel({ services, busy, t, onAdd, onEdit, onDelete }: ServicesPanelProps) {
  return (
    <div className="grid gap-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="grid gap-1">
          <h2 className="font-semibold">{t("Proxy services")}</h2>
          <p className="text-sm text-muted-foreground">{t("Independent listeners published by this node")}</p>
        </div>
        <Button type="button" disabled={busy} onClick={onAdd}><Plus />{t("Add service")}</Button>
      </div>

      {services.length === 0 ? (
        <EmptyList title={t("No services configured")} detail={t("Add a service to publish an Xray inbound")} />
      ) : (
        <div className="divide-y rounded-lg border">
          {services.map((service) => (
            <article key={service.service_id} className="grid gap-4 px-4 py-4 md:grid-cols-[minmax(0,1fr)_minmax(12rem,0.8fr)_auto] md:items-center">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <strong className="truncate text-sm font-medium">{service.display_name}</strong>
                  <StatusBadge enabled={service.enabled} t={t} />
                </div>
                <p className="mt-1 truncate font-mono text-xs text-muted-foreground">{service.service_id}</p>
              </div>
              <div className="grid gap-1 text-sm text-muted-foreground">
                <span>{t("Listen {address}:{port}", { address: service.listen, port: service.port })}</span>
                <span>{t("Public port {port}", { port: service.public_port })}</span>
              </div>
              <EntityActions busy={busy} t={t} onEdit={() => onEdit(service)} onDelete={() => onDelete(service)} />
            </article>
          ))}
        </div>
      )}
    </div>
  )
}
