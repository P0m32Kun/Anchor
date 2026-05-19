import type { Finding } from "./api";

/** Host line from nuclei finding summary (`Host: …`). */
export function extractHostFromSummary(summary?: string): string {
  if (!summary) return "";
  const m = summary.match(/^Host:\s*(.+)$/m);
  return m?.[1]?.trim() ?? "";
}

export function extractMatcherFromSummary(summary?: string): string {
  if (!summary) return "";
  const m = summary.match(/^Matcher:\s*(.*)$/m);
  return m?.[1]?.trim() ?? "";
}

/** host:port for display dedup (paths stripped; default ports for http/https). */
export function scanOriginFromHost(host: string): string {
  const raw = host.trim();
  if (!raw) return "";
  try {
    if (raw.includes("://")) {
      const u = new URL(raw);
      const h = u.hostname;
      let port = u.port;
      if (!port) {
        if (u.protocol === "http:") port = "80";
        else if (u.protocol === "https:") port = "443";
      }
      return port ? `${h}:${port}`.toLowerCase() : h.toLowerCase();
    }
  } catch {
    /* fall through */
  }
  const slash = raw.indexOf("/");
  const base = slash > 0 ? raw.slice(0, slash) : raw;
  return base.toLowerCase();
}

/** Same IP:port + template + matcher → one row in the audit UI. */
export function findingDisplayKey(f: Finding): string {
  const rule = f.source_rule_id ?? f.title;
  const origin = scanOriginFromHost(extractHostFromSummary(f.summary));
  const matcher = extractMatcherFromSummary(f.summary);
  return `${rule}|${origin}|${matcher}`;
}

export function dedupeFindingsForDisplay(findings: Finding[]): Finding[] {
  const seen = new Set<string>();
  const out: Finding[] = [];
  for (const f of findings) {
    const key = findingDisplayKey(f);
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(f);
  }
  return out;
}
