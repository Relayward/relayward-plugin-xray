import { useState } from "react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import type { Translator } from "@/i18n"
import type { ProxyService, ServiceType } from "@/types"

const fingerprints = ["chrome", "firefox", "safari", "ios", "android", "edge", "randomized"]

interface ServiceDialogProps {
  initial: ProxyService
  editingID: string | null
  serviceTypes: ServiceType[]
  existingIDs: string[]
  t: Translator
  onClose: () => void
  onApply: (service: ProxyService) => void
}

export function ServiceDialog({ initial, editingID, serviceTypes, existingIDs, t, onClose, onApply }: ServiceDialogProps) {
  const [value, setValue] = useState<ProxyService>(() => ({
    ...initial,
    vless_reality: { ...initial.vless_reality },
  }))
  const [duplicate, setDuplicate] = useState(false)
  const serviceType = serviceTypes.find((candidate) => candidate.id === value.type)

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const normalized = {
      ...value,
      service_id: value.service_id.trim(),
      display_name: value.display_name.trim(),
      listen: value.listen.trim(),
      vless_reality: {
        ...value.vless_reality,
        target: value.vless_reality.target.trim(),
        server_name: value.vless_reality.server_name.trim(),
      },
    }
    const duplicateID = existingIDs.some((id) => id === normalized.service_id && id !== editingID)
    setDuplicate(duplicateID)
    if (duplicateID || !event.currentTarget.reportValidity()) return
    onApply(normalized)
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-h-[calc(100vh-2rem)] overflow-y-auto sm:max-w-3xl" closeLabel={t("Close")}>
        <form className="grid gap-6" onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>{t(editingID == null ? "Add proxy service" : "Edit proxy service")}</DialogTitle>
            <DialogDescription>{t("Configure one independent VLESS REALITY listener")}</DialogDescription>
          </DialogHeader>

          <section className="grid gap-4" aria-labelledby="service-identity-title">
            <div className="flex items-center justify-between gap-4">
              <h3 id="service-identity-title" className="text-sm font-semibold">{t("Identity")}</h3>
              <div className="flex items-center gap-2">
                <Label htmlFor="service-enabled">{t("Enabled")}</Label>
                <Switch id="service-enabled" checked={value.enabled} onCheckedChange={(enabled) => setValue({ ...value, enabled })} />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-2">
                <Label htmlFor="service-id">{t("Service ID")}</Label>
                <Input
                  id="service-id"
                  value={value.service_id}
                  disabled={editingID != null}
                  pattern="[a-z0-9][a-z0-9._\-]{0,63}"
                  maxLength={64}
                  required
                  aria-invalid={duplicate}
                  onChange={(event) => { setValue({ ...value, service_id: event.target.value }); setDuplicate(false) }}
                />
                {duplicate ? <p className="text-sm text-destructive">{t("Service ID already exists.")}</p> : null}
              </div>
              <div className="grid gap-2">
                <Label htmlFor="service-display-name">{t("Display name")}</Label>
                <Input id="service-display-name" value={value.display_name} maxLength={100} required onChange={(event) => setValue({ ...value, display_name: event.target.value })} />
              </div>
              <div className="grid gap-2">
                <Label>{t("Service type")}</Label>
                <div className="flex h-9 items-center rounded-md border bg-muted/50 px-3 text-sm text-muted-foreground">
                  {serviceType?.display_name ?? value.type}
                </div>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="service-fingerprint">{t("Client fingerprint")}</Label>
                <Select value={value.vless_reality.fingerprint} onValueChange={(fingerprint) => setValue({ ...value, vless_reality: { ...value.vless_reality, fingerprint } })}>
                  <SelectTrigger id="service-fingerprint" className="w-full"><SelectValue /></SelectTrigger>
                  <SelectContent>{fingerprints.map((fingerprint) => <SelectItem key={fingerprint} value={fingerprint}>{fingerprint === "ios" ? "iOS" : fingerprint[0].toUpperCase() + fingerprint.slice(1)}</SelectItem>)}</SelectContent>
                </Select>
              </div>
            </div>
          </section>

          <section className="grid gap-4 border-t pt-6" aria-labelledby="service-network-title">
            <h3 id="service-network-title" className="text-sm font-semibold">{t("Listener and REALITY")}</h3>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-2">
                <Label htmlFor="service-listen">{t("Listen address")}</Label>
                <Input id="service-listen" value={value.listen} required onChange={(event) => setValue({ ...value, listen: event.target.value })} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="service-port">{t("Listen port")}</Label>
                <Input id="service-port" type="number" min={1} max={65535} value={value.port} required onChange={(event) => setValue({ ...value, port: Number(event.target.value) })} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="service-public-port">{t("Public port")}</Label>
                <Input id="service-public-port" type="number" min={1} max={65535} value={value.public_port} required onChange={(event) => setValue({ ...value, public_port: Number(event.target.value) })} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="service-target">{t("REALITY target")}</Label>
                <Input id="service-target" value={value.vless_reality.target} required onChange={(event) => setValue({ ...value, vless_reality: { ...value.vless_reality, target: event.target.value } })} />
              </div>
              <div className="grid gap-2 sm:col-span-2">
                <Label htmlFor="service-server-name">{t("Server name")}</Label>
                <Input id="service-server-name" value={value.vless_reality.server_name} required onChange={(event) => setValue({ ...value, vless_reality: { ...value.vless_reality, server_name: event.target.value } })} />
              </div>
            </div>
          </section>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>{t("Cancel")}</Button>
            <Button type="submit">{t("Apply service")}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
