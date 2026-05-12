import { useState, useEffect, useCallback, useRef } from "react";
import { api, type Dictionary } from "../lib/api";
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
  BookOpen,
  Upload,
  Trash2,
  FileText,
  Save,
  Settings2,
} from "lucide-react";

const CATEGORY_OPTIONS: { value: Dictionary["category"]; label: string }[] = [
  { value: "dirscan", label: "目录扫描" },
  { value: "subdomain", label: "子域名" },
  { value: "vhost", label: "虚拟主机" },
  { value: "custom", label: "自定义" },
];

function categoryBadge(category: Dictionary["category"]) {
  switch (category) {
    case "dirscan":
      return (
        <Badge className="bg-blue-500/10 text-blue-400 border-blue-500/20">
          目录扫描
        </Badge>
      );
    case "subdomain":
      return (
        <Badge className="bg-violet-500/10 text-violet-400 border-violet-500/20">
          子域名
        </Badge>
      );
    case "vhost":
      return (
        <Badge className="bg-amber-500/10 text-amber-400 border-amber-500/20">
          虚拟主机
        </Badge>
      );
    case "custom":
      return (
        <Badge className="bg-emerald-500/10 text-emerald-400 border-emerald-500/20">
          自定义
        </Badge>
      );
    default:
      return <Badge variant="outline">{category}</Badge>;
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

export default function DictionariesPage() {
  const toast = useToast();
  const [dictionaries, setDictionaries] = useState<Dictionary[]>([]);
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

  // Selected dictionary for edit/content operations
  const [selectedDict, setSelectedDict] = useState<Dictionary | null>(null);

  // Upload form
  const [uploadForm, setUploadForm] = useState({
    name: "",
    description: "",
    category: "custom" as Dictionary["category"],
    file: null as File | null,
  });

  // Edit metadata form
  const [editForm, setEditForm] = useState({
    name: "",
    description: "",
    category: "custom" as Dictionary["category"],
  });

  // Content editor
  const [editContent, setEditContent] = useState("");
  const [editContentLoading, setEditContentLoading] = useState(false);

  const abortRef = useRef<AbortController | null>(null);

  const loadDictionaries = useCallback(async () => {
    abortRef.current?.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    setLoading(true);
    try {
      const list = await api.listDictionaries(undefined, ctrl.signal);
      setDictionaries(list || []);
    } catch (err: any) {
      if (err.name === "AbortError") return;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadDictionaries();
    return () => abortRef.current?.abort();
  }, [loadDictionaries]);

  async function handleUpload() {
    if (!uploadForm.name.trim() || !uploadForm.file) {
      toast("名称和文件不能为空", "warning");
      return;
    }
    try {
      await api.createDictionary({
        name: uploadForm.name.trim(),
        description: uploadForm.description.trim() || undefined,
        category: uploadForm.category,
        file: uploadForm.file,
      });
      toast("字典上传成功", "success");
      setUploadModalOpen(false);
      setUploadForm({ name: "", description: "", category: "custom", file: null });
      loadDictionaries();
    } catch (err: any) {
    }
  }

  function openEditModal(dict: Dictionary) {
    setSelectedDict(dict);
    setEditForm({
      name: dict.name,
      description: dict.description || "",
      category: dict.category,
    });
    setEditModalOpen(true);
  }

  async function handleEditMetadata() {
    if (!selectedDict) return;
    if (!editForm.name.trim()) {
      toast("名称不能为空", "warning");
      return;
    }
    try {
      await api.patchDictionary(selectedDict.id, {
        name: editForm.name.trim(),
        description: editForm.description.trim() || undefined,
        category: editForm.category,
      });
      toast("字典信息更新成功", "success");
      setEditModalOpen(false);
      setSelectedDict(null);
      loadDictionaries();
    } catch (err: any) {
    }
  }

  async function openContentEditor(dict: Dictionary) {
    setSelectedDict(dict);
    setEditContent("");
    setEditContentLoading(true);
    setContentModalOpen(true);
    try {
      const text = await api.readDictionaryContent(dict.id);
      setEditContent(text);
    } catch (err: any) {
    } finally {
      setEditContentLoading(false);
    }
  }

  async function handleSaveContent() {
    if (!selectedDict) return;
    try {
      await api.writeDictionaryContent(selectedDict.id, editContent);
      toast("字典内容保存成功", "success");
      setContentModalOpen(false);
      setSelectedDict(null);
      loadDictionaries();
    } catch (err: any) {
    }
  }

  function handleDelete(dict: Dictionary) {
    setConfirmConfig({
      title: `删除字典 "${dict.name}"？`,
      onConfirm: async () => {
        try {
          await api.deleteDictionary(dict.id);
          toast("删除成功", "success");
          setConfirmOpen(false);
          loadDictionaries();
        } catch (err: any) {
        }
      },
    });
    setConfirmOpen(true);
  }

  return (
    <div className="space-y-8 animate-in fade-in duration-500 pb-20">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 text-cyan-500 font-bold text-xs uppercase tracking-widest mb-1.5">
            <BookOpen className="h-3.5 w-3.5" />
            Dictionary Management
          </div>
          <h1 className="text-3xl font-black tracking-tight text-foreground">
            字典管理
          </h1>
          <p className="text-muted-foreground mt-1">
            管理扫描使用的字典文件，支持目录扫描、子域名爆破等场景。
          </p>
        </div>
        <Button variant="secondary" onClick={() => setUploadModalOpen(true)}>
          <Upload className="mr-2 h-4 w-4" />
          上传
        </Button>
      </div>

      {/* Table */}
      <Card className="overflow-hidden">
        {loading && dictionaries.length === 0 ? (
          <div className="p-20 text-center text-muted-foreground">加载中...</div>
        ) : dictionaries.length === 0 ? (
          <div className="p-20 text-center">
            <EmptyState
              title="暂无字典"
              description="上传字典文件用于扫描任务中的目录、子域名等爆破场景"
            />
          </div>
        ) : (
          <Table>
            <TableHeader className="bg-white/5">
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>分类</TableHead>
                <TableHead>行数</TableHead>
                <TableHead>大小</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {dictionaries.map((dict) => (
                <TableRow
                  key={dict.id}
                  className="border-white/5 bg-white/[0.02] hover:bg-white/[0.04] transition-colors"
                >
                  <TableCell>
                    <div className="flex flex-col gap-0.5">
                      <span className="font-medium">{dict.name}</span>
                      {dict.description && (
                        <span className="text-xs text-muted-foreground">
                          {dict.description}
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>{categoryBadge(dict.category)}</TableCell>
                  <TableCell>
                    <span className="text-sm font-mono text-muted-foreground">
                      {dict.line_count.toLocaleString("zh-CN")}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-muted-foreground">
                      {formatSize(dict.size_bytes)}
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0"
                        onClick={() => openContentEditor(dict)}
                        title="编辑内容"
                      >
                        <FileText className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0"
                        onClick={() => openEditModal(dict)}
                        title="编辑信息"
                      >
                        <Settings2 className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0 text-rose-400 hover:text-rose-300"
                        onClick={() => handleDelete(dict)}
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
        title="上传字典"
        description="上传文本字典文件，每行一个条目"
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
              placeholder="字典名称"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">
              描述（可选）
            </label>
            <Input
              value={uploadForm.description}
              onChange={(e) =>
                setUploadForm((p) => ({ ...p, description: e.target.value }))
              }
              placeholder="字典用途描述"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">分类</label>
            <select
              value={uploadForm.category}
              onChange={(e) =>
                setUploadForm((p) => ({
                  ...p,
                  category: e.target.value as Dictionary["category"],
                }))
              }
              className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
            >
              {CATEGORY_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">文件</label>
            <input
              type="file"
              accept=".txt,.dict,.lst,text/plain"
              onChange={(e) =>
                setUploadForm((p) => ({
                  ...p,
                  file: e.target.files?.[0] || null,
                }))
              }
              className="block w-full text-sm text-muted-foreground file:mr-4 file:rounded-lg file:border-0 file:bg-primary/20 file:px-3 file:py-2 file:text-sm file:font-medium file:text-primary hover:file:bg-primary/30"
            />
            <p className="text-xs text-muted-foreground mt-1.5">
              支持 .txt / .dict / .lst 格式，最大 10MB
            </p>
          </div>
        </div>
      </Modal>

      {/* Edit Metadata Modal */}
      <Modal
        open={editModalOpen}
        onClose={() => {
          setEditModalOpen(false);
          setSelectedDict(null);
        }}
        title="编辑字典信息"
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => {
                setEditModalOpen(false);
                setSelectedDict(null);
              }}
            >
              取消
            </Button>
            <Button onClick={handleEditMetadata}>保存</Button>
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
              placeholder="字典名称"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">
              描述（可选）
            </label>
            <Input
              value={editForm.description}
              onChange={(e) =>
                setEditForm((p) => ({ ...p, description: e.target.value }))
              }
              placeholder="字典用途描述"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1.5 block">分类</label>
            <select
              value={editForm.category}
              onChange={(e) =>
                setEditForm((p) => ({
                  ...p,
                  category: e.target.value as Dictionary["category"],
                }))
              }
              className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
            >
              {CATEGORY_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </Modal>

      {/* Content Editor Modal */}
      <Modal
        open={contentModalOpen}
        onClose={() => {
          setContentModalOpen(false);
          setSelectedDict(null);
        }}
        title={selectedDict ? `编辑: ${selectedDict.name}` : "编辑字典内容"}
        size="lg"
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => {
                setContentModalOpen(false);
                setSelectedDict(null);
              }}
            >
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
            value={editContent}
            onChange={(e) => setEditContent(e.target.value)}
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
