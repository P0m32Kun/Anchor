import { useState, useEffect, useCallback, useRef } from "react";
import { api, type HttpxFingerprint } from "../lib/api";
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
import {
  Upload,
  Trash2,
  ToggleLeft,
  ToggleRight,
  Edit3,
  Save,
  FileText,
  Fingerprint,
} from "lucide-react";

export default function HttpxFingerprintsPage() {
  const toast = useToast();
  const [fingerprints, setFingerprints] = useState<HttpxFingerprint[]>([]);
  const [loading, setLoading] = useState(false);

  // Modals
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [contentModalOpen, setContentModalOpen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [confirmConfig, setConfirmConfig] = useState({
    title: "",
    onConfirm: () => {},
  });

  // Upload form state
  const [uploadForm, setUploadForm] = useState({
    name: "",
    description: "",
    type: "tech_detect" as "favicon" | "tech_detect",
    file: null as File | null,
  });

  // Edit metadata form state
  const [editForm, setEditForm] = useState<{
    id: string;
    name: string;
    description: string;
  }>({ id: "", name: "", description: "" });

  // Content editing state
  const [editContentId, setEditContentId] = useState("");
  const [editContentName, setEditContentName] = useState("");
  const [editFileContent, setEditFileContent] = useState("");
  const [editContentLoading, setEditContentLoading] = useState(false);

  const abortRef = useRef<AbortController | null>(null);

  const loadFingerprints = useCallback(async () => {
    abortRef.current?.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    setLoading(true);
    try {
      const list = await api.listHttpxFingerprints(undefined, ctrl.signal);
      setFingerprints(list || []);
    } catch (err: any) {
      if (err.name === "AbortError") return;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadFingerprints();
    return () => abortRef.current?.abort();
  }, [loadFingerprints]);

  async function handleUpload() {
    if (!uploadForm.name.trim() || !uploadForm.file) {
      toast("名称和文件不能为空", "warning");
      return;
    }
    try {
      await api.createHttpxFingerprint({
        name: uploadForm.name.trim(),
        description: uploadForm.description.trim() || undefined,
        type: uploadForm.type,
        file: uploadForm.file,
      });
      toast("指纹文件上传成功", "success");
      setUploadModalOpen(false);
      setUploadForm({ name: "", description: "", type: "tech_detect", file: null });
      loadFingerprints();
    } catch (err: any) {
      // errors handled by global handler
    }
  }

  async function handleToggleEnabled(fp: HttpxFingerprint) {
    try {
      await api.patchHttpxFingerprint(fp.id, { enabled: !fp.enabled });
      toast(fp.enabled ? "已禁用" : "已启用", "success");
      loadFingerprints();
    } catch (err: any) {
      // errors handled by global handler
    }
  }

  function openEditModal(fp: HttpxFingerprint) {
    setEditForm({
      id: fp.id,
      name: fp.name,
      description: fp.description || "",
    });
    setEditModalOpen(true);
  }

  async function handleSaveMetadata() {
    if (!editForm.name.trim()) {
      toast("名称不能为空", "warning");
      return;
    }
    try {
      await api.patchHttpxFingerprint(editForm.id, {
        name: editForm.name.trim(),
        description: editForm.description.trim() || undefined,
      });
      toast("保存成功", "success");
      setEditModalOpen(false);
      loadFingerprints();
    } catch (err: any) {
      // errors handled by global handler
    }
  }

  async function openContentEditor(fp: HttpxFingerprint) {
    setEditContentId(fp.id);
    setEditContentName(fp.name);
    setEditFileContent("");
    setEditContentLoading(true);
    setContentModalOpen(true);
    try {
      const text = await api.readHttpxFingerprintContent(fp.id);
      setEditFileContent(text);
    } catch (err: any) {
      // errors handled by global handler
    } finally {
      setEditContentLoading(false);
    }
  }

  async function handleSaveContent() {
    try {
      await api.writeHttpxFingerprintContent(editContentId, editFileContent);
      toast("内容保存成功", "success");
      setContentModalOpen(false);
      loadFingerprints();
    } catch (err: any) {
      // errors handled by global handler
    }
  }

  function handleDelete(id: string, name: string) {
    setConfirmConfig({
      title: `删除指纹文件 "${name}"？`,
      onConfirm: async () => {
        try {
          await api.deleteHttpxFingerprint(id);
          toast("删除成功", "success");
          setConfirmOpen(false);
          loadFingerprints();
        } catch (err: any) {
          // errors handled by global handler
        }
      },
    });
    setConfirmOpen(true);
  }

  function typeBadge(type: string) {
    switch (type) {
      case "favicon":
        return (
          <Badge className="bg-amber-500/10 text-amber-400 border-amber-500/20">
            Favicon
          </Badge>
        );
      case "tech_detect":
        return (
          <Badge className="bg-blue-500/10 text-blue-400 border-blue-500/20">
            技术检测
          </Badge>
        );
      default:
        return <Badge variant="outline">{type}</Badge>;
    }
  }

  return (
    <div className="space-y-8 animate-in fade-in duration-500 pb-20">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 text-cyan-500 font-bold text-xs uppercase tracking-widest mb-1.5">
            <Fingerprint className="h-3.5 w-3.5" />
            Fingerprint Management
          </div>
          <h1 className="text-3xl font-black tracking-tight text-foreground">
            HTTPX 指纹管理
          </h1>
          <p className="text-muted-foreground mt-1">
            上传并管理 HTTPX 自定义指纹文件，用于 favicon 识别和技术栈检测。
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" onClick={() => setUploadModalOpen(true)}>
            <Upload className="mr-2 h-4 w-4" />
            上传
          </Button>
        </div>
      </div>

      {/* Table */}
      <Card className="overflow-hidden">
        {loading && fingerprints.length === 0 ? (
          <div className="p-20 text-center text-muted-foreground">加载中...</div>
        ) : fingerprints.length === 0 ? (
          <div className="p-20 text-center">
            <EmptyState
              title="暂无指纹文件"
              description="上传自定义 HTTPX 指纹文件以启用 favicon 识别或技术栈检测"
            />
          </div>
        ) : (
          <Table>
            <TableHeader className="bg-white/5">
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>描述</TableHead>
                <TableHead>状态</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {fingerprints.map((fp) => (
                <TableRow
                  key={fp.id}
                  className="group border-white/5 bg-white/[0.02]"
                >
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <FileText className="h-4 w-4 text-muted-foreground" />
                      <span className="font-medium">{fp.name}</span>
                      {!fp.enabled && (
                        <Badge variant="outline" className="text-xs">
                          已禁用
                        </Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>{typeBadge(fp.type)}</TableCell>
                  <TableCell>
                    <span className="text-sm text-muted-foreground">
                      {fp.description || "—"}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 w-8 p-0"
                      onClick={() => handleToggleEnabled(fp)}
                      title={fp.enabled ? "禁用" : "启用"}
                    >
                      {fp.enabled ? (
                        <ToggleRight className="h-3.5 w-3.5 text-emerald-400" />
                      ) : (
                        <ToggleLeft className="h-3.5 w-3.5 text-muted-foreground" />
                      )}
                    </Button>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0"
                        onClick={() => openContentEditor(fp)}
                        title="编辑内容"
                      >
                        <Edit3 className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0"
                        onClick={() => openEditModal(fp)}
                        title="编辑信息"
                      >
                        <Fingerprint className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0 text-rose-400 hover:text-rose-300"
                        onClick={() => handleDelete(fp.id, fp.name)}
                        title="删除"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {/* Upload Modal */}
      <Modal
        open={uploadModalOpen}
        onClose={() => setUploadModalOpen(false)}
        title="上传指纹文件"
        description="上传自定义 HTTPX 指纹文件"
        footer={
          <>
            <Button variant="secondary" onClick={() => setUploadModalOpen(false)}>
              取消
            </Button>
            <Button onClick={handleUpload}>上传</Button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium mb-1.5 block">名称</label>
            <Input
              value={uploadForm.name}
              onChange={(e) =>
                setUploadForm((p) => ({ ...p, name: e.target.value }))
              }
              placeholder="指纹文件名称"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">描述（可选）</label>
            <Input
              value={uploadForm.description}
              onChange={(e) =>
                setUploadForm((p) => ({ ...p, description: e.target.value }))
              }
              placeholder="简短描述"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">类型</label>
            <select
              value={uploadForm.type}
              onChange={(e) =>
                setUploadForm((p) => ({
                  ...p,
                  type: e.target.value as "favicon" | "tech_detect",
                }))
              }
              className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
            >
              <option value="tech_detect">tech_detect — 技术栈检测 (Wappalyzer JSON)</option>
              <option value="favicon">favicon — Favicon 哈希</option>
            </select>
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">文件</label>
            <input
              type="file"
              onChange={(e) =>
                setUploadForm((p) => ({
                  ...p,
                  file: e.target.files?.[0] || null,
                }))
              }
              className="block w-full text-sm text-muted-foreground file:mr-4 file:rounded-lg file:border-0 file:bg-primary/20 file:px-3 file:py-2 file:text-sm file:font-medium file:text-primary hover:file:bg-primary/30"
            />
          </div>
        </div>
      </Modal>

      {/* Edit Metadata Modal */}
      <Modal
        open={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        title="编辑指纹信息"
        footer={
          <>
            <Button variant="secondary" onClick={() => setEditModalOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSaveMetadata}>
              <Save className="mr-2 h-4 w-4" />
              保存
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium mb-1.5 block">名称</label>
            <Input
              value={editForm.name}
              onChange={(e) =>
                setEditForm((p) => ({ ...p, name: e.target.value }))
              }
              placeholder="指纹文件名称"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">描述（可选）</label>
            <Input
              value={editForm.description}
              onChange={(e) =>
                setEditForm((p) => ({ ...p, description: e.target.value }))
              }
              placeholder="简短描述"
            />
          </div>
        </div>
      </Modal>

      {/* Content Editor Modal */}
      <Modal
        open={contentModalOpen}
        onClose={() => setContentModalOpen(false)}
        title={`编辑内容 — ${editContentName}`}
        size="lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setContentModalOpen(false)}>
              关闭
            </Button>
            <Button onClick={handleSaveContent} loading={editContentLoading}>
              <Save className="mr-2 h-4 w-4" />
              保存
            </Button>
          </>
        }
      >
        {editContentLoading ? (
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
