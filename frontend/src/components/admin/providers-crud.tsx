"use client";
// Live AI Providers CRUD (check.md #17) — backed by /operations/ai/providers.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useState } from "react";
import { customInstance } from "@/lib/axios-instance";

interface Provider { id: string; name: string; provider: string; base_url: string; model?: string; enabled: boolean; api_key_ref?: string | null; }

export function ProvidersCRUD() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["ai-providers"], queryFn: async () => (await customInstance.get("/operations/ai/providers")).data });
  const [form, setForm] = useState({ name: "", provider: "ollama", base_url: "http://ollama:11434", model: "" });
  const add = useMutation({
    mutationFn: async () => (await customInstance.post("/operations/ai/providers", form)).data,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["ai-providers"] }); toast.success("Provider created"); },
    onError: () => toast.error("Create failed — check name/base_url"),
  });
  const togg = useMutation({
    mutationFn: async ({ id, enable }: { id: string; enable: boolean }) =>
      (await customInstance.post(`/operations/ai/providers/${id}/${enable ? "enable" : "disable"}`)).data,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ai-providers"] }),
  });
  const del = useMutation({
    mutationFn: async (id: string) => (await customInstance.delete(`/operations/ai/providers/${id}`)).data,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["ai-providers"] }); toast.success("Deleted"); },
  });

  return (
    <div className="space-y-3">
      <div className="rounded-lg border border-pat-border p-4">
        <h3 className="text-sm font-medium text-pat-text-primary mb-3">Add Provider</h3>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-2">
          <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="name" className="rounded border border-pat-border px-3 py-2 text-sm" />
          <select value={form.provider} onChange={(e) => setForm({ ...form, provider: e.target.value })} className="rounded border border-pat-border px-3 py-2 text-sm">
            <option value="ollama">ollama</option><option value="openai">openai</option><option value="custom">custom</option>
          </select>
          <input value={form.base_url} onChange={(e) => setForm({ ...form, base_url: e.target.value })} placeholder="base_url" className="rounded border border-pat-border px-3 py-2 text-sm" />
          <input value={form.model} onChange={(e) => setForm({ ...form, model: e.target.value })} placeholder="model (optional)" className="rounded border border-pat-border px-3 py-2 text-sm" />
        </div>
        <button onClick={() => { if (!form.name || !form.base_url) return toast.error("name and base_url required"); add.mutate(); }}
          className="mt-3 rounded bg-primary px-4 py-1.5 text-xs text-primary-foreground">+ Create Provider</button>
      </div>

      {q.isLoading && <div className="text-xs text-pat-text-muted">Loading providers…</div>}
      {(q.data ?? []).map((p: any) => (
        <div key={p.id} className="flex flex-wrap items-center justify-between gap-2 border border-pat-border rounded-lg p-3">
          <div>
            <div className="text-sm text-pat-text-primary">{p.name} <span className="text-xs text-pat-text-muted">({p.provider})</span></div>
            <div className="text-xs text-pat-text-muted font-mono">{p.base_url} {p.model ? `· ${p.model}` : ""}</div>
          </div>
          <div className="flex gap-2 items-center">
            <span className={`text-xs px-2 py-1 rounded ${p.enabled ? "bg-pat-success/15 text-pat-success" : "bg-pat-bg-surface-secondary text-pat-text-muted"}`}>
              {p.enabled ? "enabled" : "disabled"}
            </span>
            <button onClick={() => togg.mutate({ id: p.id, enable: !p.enabled })} className="rounded border border-pat-border px-3 py-1 text-xs">{p.enabled ? "Disable" : "Enable"}</button>
            <button onClick={() => { if (confirm(`Delete provider ${p.name}?`)) del.mutate(p.id); }} className="rounded border border-pat-danger/40 text-pat-danger px-3 py-1 text-xs">Delete</button>
          </div>
        </div>
      ))}
      {q.isError && <div className="text-xs text-pat-danger">Failed to load providers: {String((q.error as any)?.message || "")}</div>}
    </div>
  );
}
