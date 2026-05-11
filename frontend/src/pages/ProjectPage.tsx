import { useEffect, useState } from "react";
import type React from "react";
import { useNavigate } from "react-router-dom";
import { api, PAGE_ALL } from "../lib/api";
import { useStore } from "../lib/store";
import { 
  Button, 
  Input, 
  Card, 
  CardHeader, 
  CardTitle, 
  CardDescription, 
  CardContent,
  useToast, 
  ConfirmDialog, 
  EmptyState, 
  SkeletonList,
  Badge
} from "../components";
import { Plus, Trash2, Folder, ExternalLink, Calendar, Users, Target } from "lucide-react";

export default function ProjectPage() {
  const navigate = useNavigate();
  const projects = useStore((state) => state.projects) ?? [];
  const setProjects = useStore((state) => state.setProjects);
  const setCurrentProject = useStore((state) => state.setCurrentProject);
  const [name, setName] = useState("");
  const [org, setOrg] = useState("");
  const [purpose, setPurpose] = useState("");
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const projectsLoading = useStore((state) => state.projectsLoading);
  const setProjectsLoading = useStore((state) => state.setProjectsLoading);
  const setProjectsError = useStore((state) => state.setProjectsError);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<typeof projects[0] | null>(null);
  const toast = useToast();

  useEffect(() => {
    const ctrl = new AbortController();
    setProjectsLoading(true);
    setProjectsError(null);
    api.listProjects(PAGE_ALL, ctrl.signal)
      .then((res) => {
        setProjects(res.data ?? []);
        setProjectsError(null);
      })
      .catch((err) => {
        if (err instanceof DOMException && err.name === "AbortError") return;
        const msg = err instanceof Error ? err.message : String(err);
        setProjectsError(msg);
        toast("加载项目列表失败: " + msg, "error");
        console.error(err);
      })
      .finally(() => setProjectsLoading(false));
    return () => ctrl.abort();
  }, [setProjects, setProjectsLoading, setProjectsError, toast]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      toast("项目名称不能为空", "warning");
      return;
    }
    if (creating) return;
    setCreating(true);
    try {
      const p = await api.createProject({
        name,
        organization: org || undefined,
        purpose: purpose || undefined,
      });
      setProjects([p, ...projects]);
      setCurrentProject(p);
      setName("");
      setOrg("");
      setPurpose("");
      toast("项目创建成功", "success");
    } catch (err) {
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-foreground">项目与授权边界</h1>
        <p className="text-muted-foreground mt-1">每个项目承载一次授权测试交付，后续所有的安全操作都围绕项目展开。</p>
      </div>

      <div className="grid gap-8 lg:grid-cols-[360px_1fr]">
        <aside className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">新建项目</CardTitle>
              <CardDescription>先建立授权容器，再导入目标范围。</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleCreate} className="space-y-4">
                <div className="space-y-2">
                  <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">项目名称 *</label>
                  <Input
                    placeholder="例如：2024 Q2 外部红队评估"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">组织 / 客户</label>
                  <Input
                    placeholder="客户名称或部门"
                    value={org}
                    onChange={(e) => setOrg(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">目的描述</label>
                  <Input
                    placeholder="测试目的或项目背景"
                    value={purpose}
                    onChange={(e) => setPurpose(e.target.value)}
                  />
                </div>
                <Button type="submit" variant="primary" loading={creating} className="w-full mt-2">
                  <Plus className="mr-2 h-4 w-4" />
                  创建项目
                </Button>
              </form>
            </CardContent>
          </Card>

          <Card className="bg-muted/30 border-dashed">
            <CardContent className="p-6 text-center">
               <Folder className="h-8 w-8 text-muted-foreground mx-auto mb-3 opacity-50" />
               <p className="text-xs text-muted-foreground">
                 需要导入大量项目？<br />请联系管理员通过命令行工具导入。
               </p>
            </CardContent>
          </Card>
        </aside>

        <section className="space-y-4">
          <div className="flex items-center justify-between">
             <h2 className="text-xl font-semibold tracking-tight flex items-center gap-2">
               所有项目
               <Badge variant="secondary" className="ml-2">{projects.length}</Badge>
             </h2>
          </div>

          {projectsLoading ? (
            <div className="grid gap-4">
              <SkeletonList count={3} />
            </div>
          ) : projects.length === 0 ? (
            <Card className="p-12 text-center border-dashed">
              <EmptyState
                title="暂无项目"
                description="创建第一个项目后即可导入目标并启动扫描"
              />
            </Card>
          ) : (
            <div className="grid gap-4">
              {projects.map((p) => (
                <Card key={p.id} className="group overflow-hidden transition-all hover:border-primary/50 hover:shadow-md">
                  <div className="flex flex-col sm:flex-row">
                    <div className="flex-1 p-5">
                      <div className="flex items-start justify-between">
                        <div className="space-y-1">
                          <h3 
                            className="text-lg font-bold text-foreground group-hover:text-primary transition-colors cursor-pointer flex items-center gap-2"
                            onClick={() => {
                              setCurrentProject(p);
                              navigate(`/projects/${p.id}/targets`);
                            }}
                          >
                            {p.name}
                            <ExternalLink className="h-3.5 w-3.5 opacity-0 group-hover:opacity-100 transition-opacity" />
                          </h3>
                          <div className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted-foreground mt-2">
                             <div className="flex items-center gap-1.5">
                                <Users className="h-3.5 w-3.5" />
                                <span>{p.organization || "未指定组织"}</span>
                             </div>
                             <div className="flex items-center gap-1.5 border-l pl-4 border-border">
                                <Calendar className="h-3.5 w-3.5" />
                                <span>{new Date(p.created_at).toLocaleDateString()}</span>
                             </div>
                             {p.purpose && (
                               <div className="flex items-center gap-1.5 border-l pl-4 border-border">
                                  <Target className="h-3.5 w-3.5" />
                                  <span className="truncate max-w-[200px]">{p.purpose}</span>
                               </div>
                             )}
                          </div>
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setDeleteTarget(p);
                            setDeleteDialogOpen(true);
                          }}
                          className="text-muted-foreground hover:text-destructive hover:bg-destructive/10 -mt-2 -mr-2"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>

                      <div className="mt-6 flex flex-wrap gap-2">
                        {[
                          ["目标", "targets"],
                          ["资产", "assets"],
                          ["扫描", "runs"],
                          ["发现", "findings"],
                          ["报告", "reports"],
                        ].map(([label, route]) => (
                          <Button
                            key={route}
                            variant="secondary"
                            size="sm"
                            onClick={() => {
                              setCurrentProject(p);
                              navigate(`/projects/${p.id}/${route}`);
                            }}
                            className="h-7 text-xs font-medium bg-muted/50 hover:bg-primary hover:text-primary-foreground transition-all"
                          >
                            {label}
                          </Button>
                        ))}
                      </div>
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </section>
      </div>

      <ConfirmDialog
        open={deleteDialogOpen}
        onClose={() => {
          setDeleteDialogOpen(false);
          setDeleteTarget(null);
        }}
        onConfirm={async () => {
          if (!deleteTarget) return;
          setDeletingId(deleteTarget.id);
          try {
            await api.deleteProject(deleteTarget.id);
            setProjects(projects.filter((proj) => proj.id !== deleteTarget.id));
            setDeleteDialogOpen(false);
            setDeleteTarget(null);
            toast("项目已删除", "success");
          } catch (err) {
            toast("删除失败: " + (err instanceof Error ? err.message : String(err)), "error");
          } finally {
            setDeletingId(null);
          }
        }}
        title="删除项目"
        description={deleteTarget ? `确认删除项目 "${deleteTarget.name}"？此操作不可恢复，所有关联的目标和扫描记录将被清空。` : ""}
        confirmText="彻底删除"
        cancelText="取消"
        variant="danger"
        loading={deletingId !== null}
      />
    </div>
  );
}
