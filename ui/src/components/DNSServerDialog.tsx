import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { lines } from "@/configuration"
import type { Translator } from "@/i18n"
import type { DNSServer, DNSTransport } from "@/types"

interface DNSServerDialogProps {
  initial: DNSServer
  editingID: string | null
  existingIDs: string[]
  t: Translator
  onClose: () => void
  onApply: (server: DNSServer) => void
}

export function DNSServerDialog({ initial, editingID, existingIDs, t, onClose, onApply }: DNSServerDialogProps) {
  const [value, setValue] = useState<DNSServer>(() => ({ ...initial, domains: [...initial.domains] }))
  const [domains, setDomains] = useState(() => initial.domains.join("\n"))
  const [duplicate, setDuplicate] = useState(false)

  function changeTransport(transport: DNSTransport) {
    if (transport === "system") {
      setValue({ ...value, transport, address: "", port: 0 })
    } else if (transport === "doh") {
      setValue({ ...value, transport, address: value.address.startsWith("https://") ? value.address : "https://1.1.1.1/dns-query", port: 0 })
    } else {
      setValue({ ...value, transport, address: value.address === "" || value.address.startsWith("https://") ? "1.1.1.1" : value.address, port: value.port || 53 })
    }
  }

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const candidate: DNSServer = {
      ...value,
      server_id: value.server_id.trim(),
      display_name: value.display_name.trim(),
      address: value.transport === "system" ? "" : value.address.trim(),
      port: value.transport === "system" || value.transport === "doh" ? 0 : value.port,
      domains: lines(domains),
    }
    const duplicateID = existingIDs.some((id) => id === candidate.server_id && id !== editingID)
    setDuplicate(duplicateID)
    if (duplicateID || !event.currentTarget.reportValidity()) return
    onApply(candidate)
  }

  const endpointVisible = value.transport !== "system"
  const portVisible = value.transport === "udp" || value.transport === "tcp"

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-h-[calc(100vh-2rem)] overflow-y-auto sm:max-w-2xl" closeLabel={t("Close")}>
        <form className="grid gap-6" onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>{t(editingID == null ? "Add DNS server" : "Edit DNS server")}</DialogTitle>
            <DialogDescription>{t("Configure one ordered DNS resolver")}</DialogDescription>
          </DialogHeader>

          <section className="grid gap-4" aria-labelledby="dns-identity-title">
            <div className="flex items-center justify-between gap-4">
              <h3 id="dns-identity-title" className="text-sm font-semibold">{t("Identity")}</h3>
              <div className="flex items-center gap-2">
                <Label htmlFor="dns-server-enabled">{t("Enabled")}</Label>
                <Switch id="dns-server-enabled" checked={value.enabled} onCheckedChange={(enabled) => setValue({ ...value, enabled })} />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-2">
                <Label htmlFor="dns-server-id">{t("Server ID")}</Label>
                <Input id="dns-server-id" value={value.server_id} disabled={editingID != null} pattern="[a-z0-9][a-z0-9._\-]{0,63}" maxLength={64} required aria-invalid={duplicate} onChange={(event) => { setValue({ ...value, server_id: event.target.value }); setDuplicate(false) }} />
                {duplicate ? <p className="text-sm text-destructive">{t("Server ID already exists.")}</p> : null}
              </div>
              <div className="grid gap-2">
                <Label htmlFor="dns-server-name">{t("Display name")}</Label>
                <Input id="dns-server-name" value={value.display_name} maxLength={100} required onChange={(event) => setValue({ ...value, display_name: event.target.value })} />
              </div>
              <div className="grid gap-2 sm:col-span-2">
                <Label htmlFor="dns-transport">{t("Transport")}</Label>
                <Select value={value.transport} onValueChange={(transport: DNSTransport) => changeTransport(transport)}>
                  <SelectTrigger id="dns-transport" className="w-full"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="system">{t("System resolver")}</SelectItem>
                    <SelectItem value="udp">UDP</SelectItem>
                    <SelectItem value="tcp">TCP</SelectItem>
                    <SelectItem value="doh">{t("DNS over HTTPS")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </section>

          {endpointVisible ? (
            <section className="grid gap-4 border-t pt-6" aria-labelledby="dns-endpoint-title">
              <h3 id="dns-endpoint-title" className="text-sm font-semibold">{t("Resolver endpoint")}</h3>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className={`grid gap-2 ${portVisible ? "" : "sm:col-span-2"}`}>
                  <Label htmlFor="dns-address">{t("Address")}</Label>
                  <Input id="dns-address" value={value.address} required placeholder={value.transport === "doh" ? "https://1.1.1.1/dns-query" : "1.1.1.1"} onChange={(event) => setValue({ ...value, address: event.target.value })} />
                </div>
                {portVisible ? (
                  <div className="grid gap-2">
                    <Label htmlFor="dns-port">{t("Port")}</Label>
                    <Input id="dns-port" type="number" min={1} max={65535} value={value.port} required onChange={(event) => setValue({ ...value, port: Number(event.target.value) })} />
                  </div>
                ) : null}
              </div>
            </section>
          ) : null}

          <section className="grid gap-2 border-t pt-6" aria-labelledby="dns-domains-title">
            <h3 id="dns-domains-title" className="text-sm font-semibold">{t("Domain selection")}</h3>
            <Textarea id="dns-domains" rows={5} maxLength={8192} value={domains} onChange={(event) => setDomains(event.target.value)} />
            <p className="text-xs text-muted-foreground">{t("Leave empty to use this server as a general resolver")}</p>
          </section>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>{t("Cancel")}</Button>
            <Button type="submit">{t("Apply server")}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
