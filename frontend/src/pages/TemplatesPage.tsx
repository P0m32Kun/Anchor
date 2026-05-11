import { useState, useEffect, useCallback, useRef, Fragment } from "react";
import {
  api,
  type NucleiCustomSource,
  type NucleiCustomFileEntry,
} from "../lib/api";
import {
  useToast,
  Button,
  Input,
  Card,
  Badge,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  Modal,
  ConfirmDialog,
  EmptyState,
} from "../components";
import { cn } from "../lib/utils";
import {
  FileCode,
  GitBranch,
  Upload,
  RefreshCw,
  Trash2,
  ChevronDown,
  ChevronRight,
  FileText,
  Folder,
  Save,
  Package,
  ToggleLeft,
  ToggleRight,
  Edit3,
  FileCheck,
  Layers,
} from "lucide-react";

type TabMode = "sources" | "manifest";

type TreeNode = {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
  children: TreeNode[];
};

function buildFileTree(entries: NucleiCustomFileEntry[]): TreeNode[] {
  const root: TreeNode = { name: "", path: "", isDir: true, size: 0, children: [] };
  const cache = new Map<string, TreeNode>();
  cache.set("", root);
  for (const entry of entries) {
    const clean = entry.path.replace(/\/+$/, "");
    if (!clean) continue;
    const parts = clean.split("/");
    let parent = root;
    let cum = "";
    for (let i = 0; i < parts.length; i++) {
      const name = parts[i];
      cum = cum ? `${cum}/${name}` : name;
      const isLast = i === parts.length - 1;
      let node = cache.get(cum);
      if (!node) {
        node = {
          name,
          path: cum,
          isDir: !isLast || entry.is_dir,
          size: isLast ? entry.size : 0,
          children: [],
        };
        cache.set(cum, node);
        parent.children.push(node);
      }
      parent = node;
    }
  }
  const sortFn = (n: TreeNode) => {
    n.children.sort((a, b) =>
      a.isDir !== b.isDir ? (a.isDir ? -1 : 1) : a.name.localeCompare(b.name)
    );
    n.children.forEach(sortFn);
  };
  sortFn(root);
  return root.children;
}

function flattenTree(
  nodes: TreeNode[],
  expanded: Set<string>,
  depth = 0
): Array<{ node: TreeNode; depth: number }> {
  const out: Array<{ node: TreeNode; depth: number }> = [];
  for (const n of nodes) {
    out.push({ node: n, depth });
    if (n.isDir && expanded.has(n.path)) {
      out.push(...flattenTree(n.children, expanded, depth + 1));
    }
  }
  return out;
}

export default function TemplatesPage() {
  const toast = useToast();
  const [sources, setSources] = useState<NucleiCustomSource[]>([]);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<TabMode>("sources");
  const [expandedSource, setExpandedSource] = useState<string | null>(null);
  const [filesMap, setFilesMap] = useState<Record<string, NucleiCustomFileEntry[]>>({});
  const [filesLoading, setFilesLoading] = useState<Record<string, boolean>>({});
  const [expandedFolders, setExpandedFolders] = useState<Record<string, Set<string>>>({});

  // Modals
  const [gitModalOpen, setGitModalOpen] = useState(false);
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [fileModalOpen, setFileModalOpen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [confirmConfig, setConfirmConfig] = useState({ title: "", onConfirm: () => {} });

  // Form state
  const [gitForm, setGitForm] = useState({ name: "", uri: "", branch: "", routing_policy: "manual" });
  const [uploadForm, setUploadForm] = useState({ name: "", routing_policy: "manual", file: null as File | null });

  // File editing
  const [editSourceId, setEditSourceId] = useState("");
  const [editFilePath, setEditFilePath] = useState("");
  const [editFileContent, setEditFileContent] = useState("");
  const [editFileLoading, setEditFileLoading] = useState(false);

  // Manifest
  const [manifest, setManifest] = useState<{ version: string; sources: { id: string; name: string; files: string[]; checksum: string }[]; created_at: string } | null>(null);

  const abortRef = useRef<AbortController | null>(null);

  const loadSources = useCallback(async () => {
    abortRef.current?.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    setLoading(true);
    try {
      const list = await api.listNucleiCustomSources(ctrl.signal);
      setSources(list || []);
    } catch (err: any) {
      if (err.name === "AbortError") return;
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    loadSources();
    return () => abortRef.current?.abort();
  }, [loadSources]);

  async function loadFiles(sourceId: string) {
    if (filesMap[sourceId]) {
      setExpandedSource(expandedSource === sourceId ? null : sourceId);
      return;
    }
    setFilesLoading((p) => ({ ...p, [sourceId]: true }));
    try {
      const files = await api.listNucleiCustomFiles(sourceId);
      setFilesMap((p) => ({ ...p, [sourceId]: files || [] }));
      setExpandedSource(sourceId);
    } catch (err: any) {
    } finally {
      setFilesLoading((p) => ({ ...p, [sourceId]: false }));
    }
  }

  function toggleFolder(sid: string, folderPath: string) {
    setExpandedFolders((prev) => {
      const cur = new Set(prev[sid] || []);
      if (cur.has(folderPath)) cur.delete(folderPath);
      else cur.add(folderPath);
      return { ...prev, [sid]: cur };
    });
  }

  async function handleCreateGit() {
    if (!gitForm.name.trim() || !gitForm.uri.trim()) {
      toast("名称和仓库地址不能为空", "warning");
      return;
    }
    try {
      await api.createNucleiCustomGitSource({
        name: gitForm.name.trim(),
        uri: gitForm.uri.trim(),
        branch: gitForm.branch.trim() || undefined,
        routing_policy: gitForm.routing_policy,
      });
      toast("Git模板源创建成功", "success");
      setGitModalOpen(false);
      setGitForm({ name: "", uri: "", branch: "", routing_policy: "manual" });
      loadSources();
    } catch (err: any) {
    }
  }

  async function handleUpload() {
    if (!uploadForm.name.trim() || !uploadForm.file) {
      toast("名称和文件不能为空", "warning");
      return;
    }
    try {
      await api.createNucleiCustomUploadSource(
        uploadForm.name.trim(),
        uploadForm.routing_policy,
        uploadForm.file
      );
      toast("模板上传成功", "success");
      setUploadModalOpen(false);
      setUploadForm({ name: "", routing_policy: "manual", file: null });
      loadSources();
    } catch (err: any) {
    }
  }

  async function handleRefresh(id: string) {
    try {
      await api.refreshNucleiCustomSource(id);
      toast("刷新成功", "success");
      loadSources();
    } catch (err: any) {
    }
  }

  async function handleToggleEnabled(src: NucleiCustomSource) {
    try {
      await api.patchNucleiCustomSource(src.id, { enabled: !src.enabled });
      toast(src.enabled ? "已禁用" : "已启用", "success");
      loadSources();
    } catch (err: any) {
      toast("操作失败: " + (err.message || String(err)), "error");
    }
  }

  function handleDelete(id: string, name: string) {
    setConfirmConfig({
      title: `删除模板源 "${name}"？`,
      onConfirm: async () => {
        try {
          await api.deleteNucleiCustomSource(id);
          toast("删除成功", "success");
          setConfirmOpen(false);
          loadSources();
        } catch (err: any) {
          toast("删除失败: " + (err.message || String(err)), "error");
        }
      },
    });
    setConfirmOpen(true);
  }

  async function handleValidate(id: string) {
    try {
      const result = await api.validateNucleiCustomSource(id);
      if (result.ok) {
        toast("验证通过", "success");
      } else {
        toast("验证失败: " + (result.errors?.[0] || "未知错误"), "error");
      }
      loadSources();
    } catch (err: any) {
      toast("验证失败: " + (err.message || String(err)), "error");
    }
  }

  async function handlePublish() {
    try {
      const res = await api.publishNucleiCustom();
      toast(`Bundle 发布成功: ${res.version.slice(0, 16)}...`, "success");
      loadManifest();
    } catch (err: any) {
      toast("发布失败: " + (err.message || String(err)), "error");
    }
  }

  async function loadManifest() {
    try {
      const m = await api.getNucleiCustomManifest();
      setManifest(m);
    } catch {
      setManifest(null);
    }
  }

  async function openFileEditor(sourceId: string, path: string) {
    setEditSourceId(sourceId);
    setEditFilePath(path);
    setEditFileContent("");
    setEditFileLoading(true);
    setFileModalOpen(true);
    try {
      const blob = await api.readNucleiCustomFile(sourceId, path);
      const text = await new Response(blob).text();
      setEditFileContent(text);
    } catch (err: any) {
      toast("读取文件失败: " + (err.message || String(err)), "error");
    } finally {
      setEditFileLoading(false);
    }
  }

  async function handleSaveFile() {
    try {
      await api.writeNucleiCustomFile(editSourceId, editFilePath, editFileContent);
      toast("文件保存成功", "success");
      setFileModalOpen(false);
    } catch (err: any) {
      toast("保存失败: " + (err.message || String(err)), "error");
    }
  }

  async function handleDeleteFile(sourceId: string, path: string) {
    setConfirmConfig({
      title: `删除文件 "${path}"？`,
      onConfirm: async () => {
        try {
          await api.deleteNucleiCustomFile(sourceId, path);
          toast("删除成功", "success");
          setConfirmOpen(false);
          // Refresh files
          const files = await api.listNucleiCustomFiles(sourceId);
          setFilesMap((p) => ({ ...p, [sourceId]: files || [] }));
        } catch (err: any) {
          toast("删除失败: " + (err.message || String(err)), "error");
        }
      },
    });
    setConfirmOpen(true);
  }

  function statusBadge(status: string) {
    switch (status) {
      case "ready":
        return <Badge className="bg-emerald-500/10 text-emerald-400 border-emerald-500/20">就绪</Badge>;
      case "draft":
        return <Badge className="bg-amber-500/10 text-amber-400 border-amber-500/20">草稿</Badge>;
      case "error":
        return <Badge className="bg-rose-500/10 text-rose-400 border-rose-500/20">错误</Badge>;
      default:
        return <Badge variant="outline">{status}</Badge>;
    }
  }

  function typeIcon(type: string) {
    switch (type) {
      case "git":
        return <GitBranch className="h-4 w-4 text-blue-400" />;
      case "upload":
        return <Upload className="h-4 w-4 text-violet-400" />;
      default:
        return <FileCode className="h-4 w-4 text-muted-foreground" />;
    }
  }

  return (
    <div className="space-y-8 animate-in fade-in duration-500 pb-20">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 text-cyan-500 font-bold text-xs uppercase tracking-widest mb-1.5">
            <Layers className="h-3.5 w-3.5" />
            Template Management
          </div>
          <h1 className="text-3xl font-black tracking-tight text-foreground">Nuclei 模板管理</h1>
          <p className="text-muted-foreground mt-1">
            导入、验证并发布自定义 Nuclei 模板源，Worker 将自动同步并使用。
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" onClick={() => setGitModalOpen(true)}>
            <GitBranch className="mr-2 h-4 w-4" />
            Git 导入
          </Button>
          <Button variant="secondary" onClick={() => setUploadModalOpen(true)}>
            <Upload className="mr-2 h-4 w-4" />
            上传
          </Button>
          <Button onClick={handlePublish}>
            <Package className="mr-2 h-4 w-4" />
            发布 Bundle
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-white/5 pb-0">
        {[
          { key: "sources" as TabMode, label: "模板源", icon: FileCode },
          { key: "manifest" as TabMode, label: "已发布 Bundle", icon: Package },
        ].map((t) => (
          <button
            key={t.key}
            onClick={() => {
              setActiveTab(t.key);
              if (t.key === "manifest") loadManifest();
            }}
            className={cn(
              "flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors",
              activeTab === t.key
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            )}
          >
            <t.icon className="h-4 w-4" />
            {t.label}
          </button>
        ))}
      </div>

      {/* Sources Tab */}
      {activeTab === "sources" && (
        <Card className="overflow-hidden">
          {loading && sources.length === 0 ? (
            <div className="p-20 text-center text-muted-foreground">加载中...</div>
          ) : sources.length === 0 ? (
            <div className="p-20 text-center">
              <EmptyState
                title="暂无模板源"
                description="通过 Git 导入或文件上传添加自定义 Nuclei 模板"
              />
            </div>
          ) : (
            <Table>
              <TableHeader className="bg-white/5">
                <TableRow>
                  <TableHead className="w-8"></TableHead>
                  <TableHead>名称</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>策略</TableHead>
                  <TableHead>同步时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sources.map((src) => (
                  <Fragment key={src.id}>
                    <TableRow className="group cursor-pointer" onClick={() => loadFiles(src.id)}>
                      <TableCell>
                        {expandedSource === src.id ? (
                          <ChevronDown className="h-4 w-4 text-muted-foreground" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-muted-foreground" />
                        )}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          {typeIcon(src.type)}
                          <span className="font-medium">{src.name}</span>
                          {!src.enabled && (
                            <Badge variant="outline" className="text-xs">已禁用</Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs text-muted-foreground capitalize">{src.type}</span>
                      </TableCell>
                      <TableCell>{statusBadge(src.status)}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className="font-mono text-xs">
                          {src.routing_policy}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs text-muted-foreground">
                          {src.last_sync_at
                            ? new Date(src.last_sync_at).toLocaleString("zh-CN")
                            : "—"}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center justify-end gap-1">
                          {src.type === "git" && (
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-8 w-8 p-0"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleRefresh(src.id);
                              }}
                              title="刷新"
                            >
                              <RefreshCw className="h-3.5 w-3.5" />
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 w-8 p-0"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleValidate(src.id);
                            }}
                            title="验证"
                          >
                            <FileCheck className="h-3.5 w-3.5" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 w-8 p-0"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleToggleEnabled(src);
                            }}
                            title={src.enabled ? "禁用" : "启用"}
                          >
                            {src.enabled ? (
                              <ToggleRight className="h-3.5 w-3.5 text-emerald-400" />
                            ) : (
                              <ToggleLeft className="h-3.5 w-3.5 text-muted-foreground" />
                            )}
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 w-8 p-0 text-rose-400 hover:text-rose-300"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDelete(src.id, src.name);
                            }}
                            title="删除"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                    {expandedSource === src.id && (
                      <TableRow className="bg-white/[0.02]">
                        <TableCell colSpan={7} className="p-0">
                          {filesLoading[src.id] ? (
                            <div className="p-8 text-center text-sm text-muted-foreground">加载文件...</div>
                          ) : (
                            <div className="p-4 space-y-1">
                              {(filesMap[src.id] || []).length === 0 ? (
                                <div className="text-sm text-muted-foreground py-4 text-center">暂无文件</div>
                              ) : (
                                <div className="grid gap-0.5">
                                  {flattenTree(
                                    buildFileTree(filesMap[src.id] || []),
                                    expandedFolders[src.id] || new Set()
                                  ).map(({ node, depth }) => {
                                    const isExpanded = (expandedFolders[src.id] || new Set()).has(node.path);
                                    const editable =
                                      !node.isDir &&
                                      (node.path.endsWith(".yaml") ||
                                        node.path.endsWith(".yml") ||
                                        node.path.startsWith("payloads/"));
                                    return (
                                      <div
                                        key={node.path}
                                        className={cn(
                                          "flex items-center justify-between rounded-lg pr-3 py-1.5 transition-colors group/file",
                                          node.isDir ? "cursor-pointer hover:bg-white/5" : "hover:bg-white/5"
                                        )}
                                        style={{ paddingLeft: `${12 + depth * 16}px` }}
                                        onClick={() => node.isDir && toggleFolder(src.id, node.path)}
                                      >
                                        <div className="flex items-center gap-2 min-w-0">
                                          {node.isDir ? (
                                            <>
                                              {isExpanded ? (
                                                <ChevronDown className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                                              ) : (
                                                <ChevronRight className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                                              )}
                                              <Folder className="h-3.5 w-3.5 text-amber-400 shrink-0" />
                                              <span className="text-sm font-mono truncate">{node.name}</span>
                                            </>
                                          ) : (
                                            <>
                                              <span className="w-3.5 shrink-0" />
                                              <FileText className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                                              <span className="text-sm font-mono truncate">{node.name}</span>
                                              <span className="text-xs text-muted-foreground shrink-0">
                                                {(node.size / 1024).toFixed(1)} KB
                                              </span>
                                            </>
                                          )}
                                        </div>
                                        {!node.isDir && (
                                          <div className="flex gap-1 opacity-0 group-hover/file:opacity-100 transition-opacity shrink-0">
                                            {editable && (
                                              <Button
                                                variant="ghost"
                                                size="sm"
                                                className="h-7 w-7 p-0"
                                                onClick={(e) => {
                                                  e.stopPropagation();
                                                  openFileEditor(src.id, node.path);
                                                }}
                                                title="编辑"
                                              >
                                                <Edit3 className="h-3 w-3" />
                                              </Button>
                                            )}
                                            <Button
                                              variant="ghost"
                                              size="sm"
                                              className="h-7 w-7 p-0 text-rose-400"
                                              onClick={(e) => {
                                                e.stopPropagation();
                                                handleDeleteFile(src.id, node.path);
                                              }}
                                              title="删除"
                                            >
                                              <Trash2 className="h-3 w-3" />
                                            </Button>
                                          </div>
                                        )}
                                      </div>
                                    );
                                  })}
                                </div>
                              )}
                            </div>
                          )}
                        </TableCell>
                      </TableRow>
                    )}
                  </Fragment>
                ))}
              </TableBody>
            </Table>
          )}
        </Card>
      )}

      {/* Manifest Tab */}
      {activeTab === "manifest" && (
        <Card className="p-8">
          {!manifest ? (
            <EmptyState
              title="暂无已发布的 Bundle"
              description="点击「发布 Bundle」将已启用的模板源打包并激活"
            />
          ) : (
            <div className="space-y-6">
              <div className="flex items-center gap-3">
                <Package className="h-8 w-8 text-primary" />
                <div>
                  <div className="text-lg font-bold">已激活的 Bundle</div>
                  <div className="text-xs font-mono text-muted-foreground">{manifest.version}</div>
                </div>
              </div>
              <div className="text-sm text-muted-foreground">
                发布时间: {new Date(manifest.created_at).toLocaleString("zh-CN")}
              </div>
              <div className="space-y-3">
                <div className="text-sm font-bold">包含的源:</div>
                {manifest.sources.map((s) => (
                  <div key={s.id} className="rounded-lg border border-white/5 bg-white/[0.02] p-3">
                    <div className="flex items-center justify-between">
                      <span className="font-medium text-sm">{s.name}</span>
                      <Badge variant="outline" className="font-mono text-xs">{s.files.length} files</Badge>
                    </div>
                    <div className="text-xs font-mono text-muted-foreground mt-1">{s.checksum.slice(0, 16)}...</div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </Card>
      )}

      {/* Git Import Modal */}
      <Modal
        open={gitModalOpen}
        onClose={() => setGitModalOpen(false)}
        title="从 Git 导入模板源"
        description="克隆一个公开的 Git 仓库作为模板源"
        footer={
          <>
            <Button variant="secondary" onClick={() => setGitModalOpen(false)}>取消</Button>
            <Button onClick={handleCreateGit}>创建</Button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium mb-1.5 block">名称</label>
            <Input
              value={gitForm.name}
              onChange={(e) => setGitForm((p) => ({ ...p, name: e.target.value }))}
              placeholder="例如: RBKD-SEC Templates"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">仓库地址 (HTTPS)</label>
            <Input
              value={gitForm.uri}
              onChange={(e) => setGitForm((p) => ({ ...p, uri: e.target.value }))}
              placeholder="https://github.com/owner/templates.git"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">分支 (可选)</label>
            <Input
              value={gitForm.branch}
              onChange={(e) => setGitForm((p) => ({ ...p, branch: e.target.value }))}
              placeholder="main"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">路由策略</label>
            <select
              value={gitForm.routing_policy}
              onChange={(e) => setGitForm((p) => ({ ...p, routing_policy: e.target.value }))}
              className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
            >
              <option value="manual">manual — 仅手动指定使用</option>
              <option value="auto">auto — 自动包含在所有扫描中</option>
            </select>
          </div>
        </div>
      </Modal>

      {/* Upload Modal */}
      <Modal
        open={uploadModalOpen}
        onClose={() => setUploadModalOpen(false)}
        title="上传模板源"
        description="上传单个 YAML 文件或 ZIP 压缩包"
        footer={
          <>
            <Button variant="secondary" onClick={() => setUploadModalOpen(false)}>取消</Button>
            <Button onClick={handleUpload}>上传</Button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium mb-1.5 block">名称</label>
            <Input
              value={uploadForm.name}
              onChange={(e) => setUploadForm((p) => ({ ...p, name: e.target.value }))}
              placeholder="模板源名称"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">路由策略</label>
            <select
              value={uploadForm.routing_policy}
              onChange={(e) => setUploadForm((p) => ({ ...p, routing_policy: e.target.value }))}
              className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
            >
              <option value="manual">manual — 仅手动指定使用</option>
              <option value="auto">auto — 自动包含在所有扫描中</option>
            </select>
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">文件</label>
            <input
              type="file"
              accept=".yaml,.yml,.zip"
              onChange={(e) => setUploadForm((p) => ({ ...p, file: e.target.files?.[0] || null }))}
              className="block w-full text-sm text-muted-foreground file:mr-4 file:rounded-lg file:border-0 file:bg-primary/20 file:px-3 file:py-2 file:text-sm file:font-medium file:text-primary hover:file:bg-primary/30"
            />
          </div>
        </div>
      </Modal>

      {/* File Editor Modal */}
      <Modal
        open={fileModalOpen}
        onClose={() => setFileModalOpen(false)}
        title={editFilePath}
        size="lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setFileModalOpen(false)}>关闭</Button>
            <Button onClick={handleSaveFile} loading={editFileLoading}>
              <Save className="mr-2 h-4 w-4" />
              保存
            </Button>
          </>
        }
      >
        {editFileLoading ? (
          <div className="py-12 text-center text-muted-foreground">加载中...</div>
        ) : (
          <textarea
            value={editFileContent}
            onChange={(e) => setEditFileContent(e.target.value)}
            className="w-full h-96 rounded-lg border border-white/10 bg-card px-4 py-3 text-sm font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-primary/50 resize-none"
            spellCheck={false}
          />
        )}
      </Modal>

      {/* Confirm Dialog */}
      <ConfirmDialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={confirmConfig.onConfirm}
        title={confirmConfig.title}
        variant="danger"
      />
    </div>
  );
}
