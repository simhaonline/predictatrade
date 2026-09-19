import { redirect } from "next/navigation";

/**
 * /admin/operations → /admin/risk-center (2026-09-19 merge).
 *
 * The two pages were near-duplicates (identical platform-state block, the
 * same 4 emergency controls, same confirm dialog). Risk Center now hosts the
 * union of both: emergency controls + risk guardrails + hard gates + the
 * active-operations audit trail. This page permanently redirects.
 */
export default function AdminOperationsRedirect() {
  redirect("/admin/risk-center");
}