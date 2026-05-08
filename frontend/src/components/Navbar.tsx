import { Link, useLocation, useNavigate } from "react-router-dom";
import type React from "react";
import { useStore } from "../lib/store";
import { cn } from "../lib/utils";
import {
  LayoutDashboard,
  Files,
  Search,
  Cpu,
  Settings,
  Target,
  Box,
  Play,
  AlertTriangle,
  FileText,
  ChevronDown,
  Sparkles,
  Layers
} from "lucide-react";

type NavItem = {
  path: string;
  label: string;
  icon: React.ElementType;
  color: string; // 增加图标基础色
};

const globalNavItems: NavItem[] = [
  { path: "/", label: "总览", icon: LayoutDashboard, color: "text-blue-400" },
  { path: "/projects", label: "项目", icon: Files, color: "text-indigo-400" },
  { path: "/engines", label: "引擎", icon: Search, color: "text-cyan-400" },
  { path: "/templates", label: "模板", icon: Layers, color: "text-violet-400" },
  { path: "/workers", label: "Workers", icon: Cpu, color: "text-emerald-400" },
  { path: "/settings", label: "设置", icon: Settings, color: "text-slate-400" },
];

function isItemActive(locationPathname: string, itemPath: string): boolean {
  if (itemPath === "/") {
    return locationPathname === "/";
  }
  return locationPathname.startsWith(itemPath);
}

function NavLinkItem({ item, active }: { item: NavItem; active: boolean }) {
  return (
    <Link
      to={item.path}
      className={cn(
        "group relative flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-300",
        active
          ? "bg-primary/10 text-primary shadow-[inset_0_0_12px_rgba(0,212,255,0.1)] border border-primary/20"
          : "text-muted-foreground hover:bg-white/5 hover:text-foreground"
      )}
    >
      <item.icon className={cn(
        "h-4 w-4 transition-transform duration-300 group-hover:scale-110",
        active ? "text-primary" : item.color
      )} strokeWidth={2.5} />
      <span>{item.label}</span>
      {active && (
         <div className="absolute left-[-12px] top-1/4 h-1/2 w-1 rounded-r-full bg-primary shadow-[0_0_10px_rgba(0,212,255,0.8)]" />
      )}
    </Link>
  );
}

export function Navbar() {
  const location = useLocation();
  const navigate = useNavigate();
  const currentProjectId = useStore((s) => s.currentProjectId);
  const projects = useStore((s) => s.projects);
  const setCurrentProjectId = useStore((s) => s.setCurrentProjectId);

  const projectLinks: NavItem[] = currentProjectId
    ? [
        { path: `/projects/${currentProjectId}/targets`, label: "目标与 Scope", icon: Target, color: "text-emerald-400" },
        { path: `/projects/${currentProjectId}/assets`, label: "资产清单", icon: Box, color: "text-blue-400" },
        { path: `/projects/${currentProjectId}/runs`, label: "扫描执行", icon: Play, color: "text-violet-400" },
        { path: `/projects/${currentProjectId}/findings`, label: "发现审核", icon: AlertTriangle, color: "text-rose-400" },
        { path: `/projects/${currentProjectId}/reports`, label: "报告交付", icon: FileText, color: "text-amber-400" },
      ]
    : [];

  const handleProjectSwitch = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const id = e.target.value;
    if (id) {
      setCurrentProjectId(id);
      navigate(`/projects/${id}/targets`);
    }
  };

  return (
    <aside className="fixed inset-y-0 left-0 z-50 flex w-64 flex-col glass-panel border-r border-white/5 shadow-2xl">
      <div className="flex h-16 items-center gap-3 px-6 border-b border-white/5">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-primary to-violet-600 shadow-lg shadow-primary/20">
           <Box className="h-5 w-5 text-white" />
        </div>
        <span className="font-black text-lg tracking-tighter bg-clip-text text-transparent bg-gradient-to-r from-white to-white/60">
            ANCHOR
        </span>
      </div>

      <div className="flex-1 space-y-6 overflow-y-auto px-4 py-6 custom-scrollbar">
        <div>
          <h2 className="mb-3 px-2 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground/50">
            Workspace
          </h2>
          <div className="space-y-1.5">
            {globalNavItems.map((item) => (
              <NavLinkItem
                key={item.path}
                item={item}
                active={isItemActive(location.pathname, item.path)}
              />
            ))}
          </div>
        </div>

        {currentProjectId && (
          <div className="animate-in slide-in-from-left-4 duration-500">
             <h2 className="mb-3 px-2 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground/50 flex items-center gap-2">
              <Sparkles className="h-3 w-3" />
              Active Project
            </h2>
            <div className="px-1 mb-4">
              <div className="relative group">
                <select
                  value={currentProjectId ?? ""}
                  onChange={handleProjectSwitch}
                  className="w-full appearance-none rounded-xl border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold ring-offset-background focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all hover:bg-white/10"
                >
                  {projects.map((project) => (
                    <option key={project.id} value={project.id} className="bg-slate-900">
                      {project.name}
                    </option>
                  ))}
                </select>
                <ChevronDown className="absolute right-3 top-3 h-4 w-4 opacity-50 group-hover:text-primary transition-colors pointer-events-none" />
              </div>
            </div>
            
            <div className="space-y-1.5">
              {projectLinks.map((item) => (
                <NavLinkItem
                  key={item.path}
                  item={item}
                  active={isItemActive(location.pathname, item.path)}
                />
              ))}
            </div>
          </div>
        )}
      </div>

      <div className="mt-auto p-4">
        <div className="rounded-2xl border border-white/5 bg-white/5 p-4 shadow-inner">
          <div className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60 mb-3 flex justify-between items-center">
            Nodes Online
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
          </div>
          <div className="flex gap-1.5">
             {[1,2,3,4,5,6].map(i => (
               <div key={i} className={cn(
                 "h-1.5 flex-1 rounded-full",
                 currentProjectId ? "bg-primary/30" : "bg-white/5"
               )} />
             ))}
          </div>
          <div className="mt-3 text-[11px] font-semibold text-muted-foreground">
            System status nominal
          </div>
        </div>
      </div>
    </aside>
  );
}
