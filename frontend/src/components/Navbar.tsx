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
  ChevronDown
} from "lucide-react";

type NavItem = {
  path: string;
  label: string;
  icon: React.ElementType;
};

const globalNavItems: NavItem[] = [
  { path: "/", label: "总览", icon: LayoutDashboard },
  { path: "/projects", label: "项目", icon: Files },
  { path: "/engines", label: "搜索引擎", icon: Search },
  { path: "/workers", label: "Workers", icon: Cpu },
  { path: "/settings", label: "设置", icon: Settings },
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
        "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all",
        active
          ? "bg-secondary text-foreground shadow-sm"
          : "text-muted-foreground hover:bg-secondary/50 hover:text-foreground"
      )}
    >
      <item.icon className="h-4 w-4" strokeWidth={2} />
      <span>{item.label}</span>
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
        { path: `/projects/${currentProjectId}/targets`, label: "目标与 Scope", icon: Target },
        { path: `/projects/${currentProjectId}/assets`, label: "资产清单", icon: Box },
        { path: `/projects/${currentProjectId}/runs`, label: "扫描执行", icon: Play },
        { path: `/projects/${currentProjectId}/findings`, label: "发现审核", icon: AlertTriangle },
        { path: `/projects/${currentProjectId}/reports`, label: "报告交付", icon: FileText },
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
    <aside className="fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-right bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-r border-border">
      <div className="flex h-14 items-center gap-2 px-6 border-b border-border">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-foreground text-background">
           <Box className="h-4 w-4" />
        </div>
        <span className="font-bold tracking-tight text-foreground">Anchor</span>
      </div>

      <div className="flex-1 space-y-4 overflow-y-auto px-3 py-4">
        <div>
          <h2 className="mb-2 px-4 text-xs font-semibold tracking-tight text-muted-foreground uppercase">
            Workspace
          </h2>
          <div className="space-y-1">
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
          <div>
             <h2 className="mb-2 px-4 text-xs font-semibold tracking-tight text-muted-foreground uppercase">
              Current Project
            </h2>
            <div className="px-3 mb-2">
              <div className="relative">
                <select
                  value={currentProjectId ?? ""}
                  onChange={handleProjectSwitch}
                  className="w-full appearance-none rounded-md border border-input bg-background px-3 py-1.5 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="" disabled>选择项目</option>
                  {projects.map((project) => (
                    <option key={project.id} value={project.id}>
                      {project.name}
                    </option>
                  ))}
                </select>
                <ChevronDown className="absolute right-3 top-2.5 h-3.5 w-3.5 opacity-50" />
              </div>
            </div>
            
            <div className="space-y-1">
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
        <div className="rounded-lg border border-border bg-muted/50 p-3">
          <div className="text-[10px] font-medium uppercase text-muted-foreground mb-2">Workflow Status</div>
          <div className="flex gap-1">
             {[1,2,3,4,5,6].map(i => (
               <div key={i} className={cn(
                 "h-1.5 flex-1 rounded-full",
                 currentProjectId ? "bg-foreground/20" : "bg-muted"
               )} />
             ))}
          </div>
          <div className="mt-2 text-[11px] text-muted-foreground">
            {currentProjectId ? "Ready for audit" : "No project active"}
          </div>
        </div>
      </div>
    </aside>
  );
}
