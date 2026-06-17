import { Link, useLocation, useNavigate } from "react-router-dom";
import type React from "react";
import { useEffect, useRef } from "react";
import { useStore } from "../lib/store";
import { api, PAGE_ALL } from "../lib/api";
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
  Layers,
  BookOpen,
  Fingerprint,
  ShieldAlert,
  Shield,
  PanelLeftClose,
  PanelLeftOpen
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
  { path: "/dictionaries", label: "字典", icon: BookOpen, color: "text-orange-400" },
  { path: "/httpx-fingerprints", label: "指纹", icon: Fingerprint, color: "text-pink-400" },
  { path: "/vuln-templates", label: "漏洞模板", icon: ShieldAlert, color: "text-rose-400" },
  { path: "/excluded-domains", label: "域名排除", icon: Shield, color: "text-teal-400" },
  { path: "/workers", label: "Workers", icon: Cpu, color: "text-emerald-400" },
  { path: "/settings", label: "设置", icon: Settings, color: "text-slate-400" },
];

function isItemActive(locationPathname: string, itemPath: string): boolean {
  if (itemPath === "/") {
    return locationPathname === "/";
  }
  return locationPathname.startsWith(itemPath);
}

function NavLinkItem({ item, active, collapsed }: { item: NavItem; active: boolean; collapsed: boolean }) {
  return (
    <Link
      to={item.path}
      title={collapsed ? item.label : undefined}
      className={cn(
        "group relative flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-300",
        active
          ? "bg-primary/10 text-primary shadow-[inset_0_0_12px_rgba(0,212,255,0.1)] border border-primary/20"
          : "text-muted-foreground hover:bg-white/5 hover:text-foreground",
        collapsed && "justify-center px-2"
      )}
    >
      <item.icon className={cn(
        "h-4 w-4 transition-transform duration-300 group-hover:scale-110 shrink-0",
        active ? "text-primary" : item.color
      )} strokeWidth={2.5} />
      {!collapsed && <span className="truncate">{item.label}</span>}
      {active && (
         <div className={cn(
           "absolute left-[-12px] top-1/4 h-1/2 w-1 rounded-r-full bg-primary shadow-[0_0_10px_rgba(0,212,255,0.8)]",
           collapsed && "left-0"
         )} />
      )}
    </Link>
  );
}

export function Navbar() {
  const location = useLocation();
  const navigate = useNavigate();
  const currentProjectId = useStore((s) => s.currentProjectId);
  const projects = useStore((s) => s.projects);
  const setProjects = useStore((s) => s.setProjects);
  const setCurrentProjectId = useStore((s) => s.setCurrentProjectId);
  const collapsed = useStore((s) => s.sidebarCollapsed);
  const toggleCollapsed = useStore((s) => s.toggleSidebarCollapsed);
  const loadRef = useRef(false);

  // 自动加载 projects 列表（仅一次）
  useEffect(() => {
    if (loadRef.current) return;
    loadRef.current = true;

    if (projects.length === 0) {
      const ctrl = new AbortController();
      api.listProjects(PAGE_ALL, ctrl.signal)
        .then((res) => {
          setProjects(res.data ?? []);
        })
        .catch((err) => {
          if (err instanceof DOMException && err.name === "AbortError") return;
          console.error("Failed to load projects:", err);
        });
      return () => ctrl.abort();
    }
  }, [projects, setProjects]);

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
    <aside className={cn(
      "fixed inset-y-0 left-0 z-50 flex flex-col glass-panel border-r border-white/5 shadow-2xl transition-all duration-300",
      collapsed ? "w-20" : "w-64"
    )}>
      <div className={cn(
        "flex h-16 items-center border-b border-white/5 px-4",
        collapsed ? "justify-center" : "justify-between gap-3 px-6"
      )}>
        <div className="flex items-center gap-3 overflow-hidden">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-primary to-violet-600 shadow-lg shadow-primary/20">
             <Box className="h-5 w-5 text-white" />
          </div>
          {!collapsed && (
            <span className="font-black text-lg tracking-tighter bg-clip-text text-transparent bg-gradient-to-r from-white to-white/60 truncate">
                ANCHOR
            </span>
          )}
        </div>
        
        {!collapsed && (
          <button 
            onClick={toggleCollapsed}
            className="p-1.5 rounded-lg hover:bg-white/10 text-muted-foreground transition-colors"
          >
            <PanelLeftClose className="h-4 w-4" />
          </button>
        )}
      </div>

      <div className="flex-1 space-y-6 overflow-y-auto px-3 py-6 custom-scrollbar">
        {collapsed && (
          <div className="flex justify-center mb-4">
            <button 
              onClick={toggleCollapsed}
              className="p-2 rounded-xl bg-primary/10 text-primary hover:bg-primary/20 transition-all border border-primary/20 shadow-[0_0_15px_rgba(0,212,255,0.1)]"
            >
              <PanelLeftOpen className="h-5 w-5" />
            </button>
          </div>
        )}

        <div>
          {!collapsed && (
            <h2 className="mb-3 px-2 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground/50">
              Workspace
            </h2>
          )}
          <div className="space-y-1.5">
            {globalNavItems.map((item) => (
              <NavLinkItem
                key={item.path}
                item={item}
                active={isItemActive(location.pathname, item.path)}
                collapsed={collapsed}
              />
            ))}
          </div>
        </div>

        {currentProjectId && (
          <div className="animate-in slide-in-from-left-4 duration-500">
             {!collapsed && (
               <h2 className="mb-3 px-2 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground/50 flex items-center gap-2">
                <Sparkles className="h-3 w-3" />
                Active Project
              </h2>
             )}
            <div className={cn("mb-4", collapsed ? "px-0" : "px-1")}>
              <div className="relative group">
                <select
                  value={currentProjectId ?? ""}
                  onChange={handleProjectSwitch}
                  className={cn(
                    "w-full appearance-none rounded-xl border border-white/10 bg-white/5 text-sm font-semibold ring-offset-background focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all hover:bg-white/10",
                    collapsed ? "px-2 py-2" : "px-4 py-2"
                  )}
                >
                  {projects.map((project) => (
                    <option key={project.id} value={project.id} className="bg-slate-900">
                      {collapsed ? project.name.substring(0, 2) : project.name}
                    </option>
                  ))}
                </select>
                {!collapsed && <ChevronDown className="absolute right-3 top-3 h-4 w-4 opacity-50 group-hover:text-primary transition-colors pointer-events-none" />}
              </div>
            </div>
            
            <div className="space-y-1.5">
              {projectLinks.map((item) => (
                <NavLinkItem
                  key={item.path}
                  item={item}
                  active={isItemActive(location.pathname, item.path)}
                  collapsed={collapsed}
                />
              ))}
            </div>
          </div>
        )}
      </div>

      {!collapsed && (
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
      )}
    </aside>
  );
}
