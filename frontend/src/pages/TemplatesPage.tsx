import { useState, useEffect, useCallback, useRef } from "react";
import { api, type NucleiCustomSource } from "../lib/api";
import {
  useToast,
  Card,
  Badge,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  EmptyState,
  Switch,
} from "../components";
import { Layers } from "lucide-react";

const RBKD_MOUNT_PATH = "/opt/rbkd-templates";
const BUILTIN_READONLY_TITLE = "内置只读，通过目录挂载与 Git 同步更新";

export default function TemplatesPage() {
  const toast = useToast();
  const [sources, setSources] = useState<NucleiCustomSource[]>([]);
  const [loading, setLoading] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  const loadSources = useCallback(async () => {
    abortRef.current?.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    setLoading(true);
    try {
      const list = await api.listNucleiCustomSources(ctrl.signal);
      setSources((list || []).filter((s) => s.builtin));
    } catch (err: unknown) {
      if (err instanceof DOMException && err.name === "AbortError") return;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadSources();
    return () => abortRef.current?.abort();
  }, [loadSources]);

  async function handleToggleEnabled(src: NucleiCustomSource) {
    try {
      const updated = await api.patchNucleiCustomSourceEnabled(src.id, !src.enabled);
      setSources((prev) => prev.map((s) => (s.id === src.id ? updated : s)));
      toast(updated.enabled ? "RBKD 模板已启用" : "RBKD 模板已禁用", "success");
    } catch {
      // global error handler
    }
  }

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-3">
          <div className="h-10 w-10 rounded-xl bg-primary/10 flex items-center justify-center border border-primary/20">
            <Layers className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">Nuclei 模板</h1>
            <p className="text-sm text-muted-foreground">
              团队 RBKD 模板库 — 启动时 Git 同步到 <code className="text-xs">{RBKD_MOUNT_PATH}</code>，Worker 通过 symlink 加载
            </p>
          </div>
        </div>
      </div>

      <Card className="border-white/5 bg-white/[0.02] p-4 text-sm text-muted-foreground leading-relaxed">
        <p className="font-medium text-foreground mb-2">更新模板</p>
        <ul className="list-disc list-inside space-y-1 text-xs">
          <li>Server / Worker 启动时通过 <code>ANCHOR_BUILTIN_SYNC</code> 从 GitHub 拉取 RBKD-templates</li>
          <li>自定义模板请直接挂载或更新 <code>{RBKD_MOUNT_PATH}</code> 目录（Docker volume / 运维脚本）</li>
          <li>禁用下方开关后，Worker 会移除 nuclei 搜索路径中的 RBKD symlink</li>
        </ul>
      </Card>

      {loading ? (
        <div className="py-16 text-center text-muted-foreground text-sm">加载中...</div>
      ) : sources.length === 0 ? (
        <EmptyState
          title="暂无内置模板源"
          description="Server 启动后应自动 seed RBKD 模板行；请检查 ANCHOR_BUILTIN_TEMPLATES_REPO 配置"
        />
      ) : (
        <Card className="border-white/5 overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent border-white/5">
                <TableHead>名称</TableHead>
                <TableHead>安装路径</TableHead>
                <TableHead>来源</TableHead>
                <TableHead>状态</TableHead>
                <TableHead className="text-right">启用</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sources.map((src) => (
                <TableRow key={src.id} className="border-white/5">
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{src.name}</span>
                      <Badge variant="outline" className="text-[10px] uppercase">
                        内置
                      </Badge>
                    </div>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {src.install_path || "RBKD-templates"}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground max-w-xs truncate" title={src.uri}>
                    {src.uri ? `${src.uri}${src.branch ? `@${src.branch}` : ""}` : RBKD_MOUNT_PATH}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant="outline"
                      className={
                        src.status === "ready"
                          ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20"
                          : "bg-amber-500/10 text-amber-400 border-amber-500/20"
                      }
                    >
                      {src.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Switch
                      checked={src.enabled}
                      onCheckedChange={() => handleToggleEnabled(src)}
                      aria-label={`启用 ${src.name}`}
                      title={BUILTIN_READONLY_TITLE}
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}
    </div>
  );
}
