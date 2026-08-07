import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { lines } from "@/configuration"
import type { Translator } from "@/i18n"
import type { RoutingAction, RoutingProtocol, RoutingRule } from "@/types"

const protocols: RoutingProtocol[] = ["http", "tls", "quic", "bittorrent"]

interface RoutingRuleDialogProps {
  initial: RoutingRule
  editingID: string | null
  existingIDs: string[]
  t: Translator
  onClose: () => void
  onApply: (rule: RoutingRule) => void
}

export function RoutingRuleDialog({ initial, editingID, existingIDs, t, onClose, onApply }: RoutingRuleDialogProps) {
  const [value, setValue] = useState<RoutingRule>(() => ({
    ...initial,
    domains: [...initial.domains],
    ip_cidrs: [...initial.ip_cidrs],
    protocols: [...initial.protocols],
  }))
  const [domains, setDomains] = useState(() => initial.domains.join("\n"))
  const [cidrs, setCIDRs] = useState(() => initial.ip_cidrs.join("\n"))
  const [error, setError] = useState("")

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const candidate: RoutingRule = {
      ...value,
      rule_id: value.rule_id.trim(),
      display_name: value.display_name.trim(),
      domains: lines(domains),
      ip_cidrs: lines(cidrs),
      protocols: [...value.protocols],
    }
    if (existingIDs.some((id) => id === candidate.rule_id && id !== editingID)) {
      setError(t("Rule ID already exists."))
      return
    }
    if (candidate.domains.length + candidate.ip_cidrs.length + candidate.protocols.length === 0) {
      setError(t("No match values configured."))
      return
    }
    setError("")
    if (!event.currentTarget.reportValidity()) return
    onApply(candidate)
  }

  function toggleProtocol(protocol: RoutingProtocol, checked: boolean) {
    const next = new Set(value.protocols)
    if (checked) next.add(protocol); else next.delete(protocol)
    setValue({ ...value, protocols: protocols.filter((candidate) => next.has(candidate)) })
    setError("")
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-h-[calc(100vh-2rem)] overflow-y-auto sm:max-w-3xl" closeLabel={t("Close")}>
        <form className="grid gap-6" onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>{t(editingID == null ? "Add routing rule" : "Edit routing rule")}</DialogTitle>
            <DialogDescription>{t("All populated match categories must match")}</DialogDescription>
          </DialogHeader>

          <section className="grid gap-4" aria-labelledby="routing-identity-title">
            <div className="flex items-center justify-between gap-4">
              <h3 id="routing-identity-title" className="text-sm font-semibold">{t("Identity and action")}</h3>
              <div className="flex items-center gap-2">
                <Label htmlFor="routing-enabled">{t("Enabled")}</Label>
                <Switch id="routing-enabled" checked={value.enabled} onCheckedChange={(enabled) => setValue({ ...value, enabled })} />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-2">
                <Label htmlFor="routing-id">{t("Rule ID")}</Label>
                <Input id="routing-id" value={value.rule_id} disabled={editingID != null} pattern="[a-z0-9][a-z0-9._\-]{0,63}" maxLength={64} required aria-invalid={error === t("Rule ID already exists.")} onChange={(event) => { setValue({ ...value, rule_id: event.target.value }); setError("") }} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="routing-name">{t("Display name")}</Label>
                <Input id="routing-name" value={value.display_name} maxLength={100} required onChange={(event) => setValue({ ...value, display_name: event.target.value })} />
              </div>
              <div className="grid gap-2 sm:col-span-2">
                <Label htmlFor="routing-action">{t("Action")}</Label>
                <Select value={value.action} onValueChange={(action: RoutingAction) => setValue({ ...value, action })}>
                  <SelectTrigger id="routing-action" className="w-full"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="blocked">{t("Block")}</SelectItem>
                    <SelectItem value="direct">{t("Direct")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </section>

          <section className="grid gap-4 border-t pt-6" aria-labelledby="destination-title">
            <h3 id="destination-title" className="text-sm font-semibold">{t("Destination matches")}</h3>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-2">
                <Label htmlFor="routing-domains">{t("Domain suffixes")}</Label>
                <Textarea id="routing-domains" rows={5} maxLength={8192} value={domains} onChange={(event) => { setDomains(event.target.value); setError("") }} />
                <p className="text-xs text-muted-foreground">{t("One lowercase domain per line; subdomains are included")}</p>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="routing-cidrs">{t("IP CIDRs")}</Label>
                <Textarea id="routing-cidrs" rows={5} maxLength={8192} value={cidrs} onChange={(event) => { setCIDRs(event.target.value); setError("") }} />
                <p className="text-xs text-muted-foreground">{t("One canonical IPv4 or IPv6 CIDR per line")}</p>
              </div>
            </div>
          </section>

          <section className="grid gap-4 border-t pt-6" aria-labelledby="protocol-title">
            <h3 id="protocol-title" className="text-sm font-semibold">{t("Sniffed protocols")}</h3>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {protocols.map((protocol) => (
                <Label key={protocol} className="rounded-md border px-3 py-3">
                  <Checkbox checked={value.protocols.includes(protocol)} onCheckedChange={(checked) => toggleProtocol(protocol, checked === true)} />
                  {protocol === "bittorrent" ? "BitTorrent" : protocol.toUpperCase()}
                </Label>
              ))}
            </div>
            {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
          </section>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>{t("Cancel")}</Button>
            <Button type="submit">{t("Apply rule")}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
