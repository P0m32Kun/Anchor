import { useLocation, Link } from "react-router-dom";
import { useStore } from "../lib/store";
import { cn } from "../lib/utils";
import { ChevronRight, Home, Layout } from "lucide-react";

export function Header() {
  const location = useLocation();
  const currentProject = useStore((s) => s.currentProject);
  
  const pathnames = location.pathname.split("/").filter((x) => x);

  return (
    <header className="sticky top-0 z-40 w-full border-b border-white/5 bg-background/60 backdrop-blur-xl">
      <div className="flex h-16 items-center gap-4 px-8">
        <nav className="flex items-center space-x-2 text-sm font-medium text-muted-foreground">
          <Link
            to="/"
            className="flex items-center gap-1 hover:text-foreground transition-colors"
          >
            <Home className="h-4 w-4" />
            <span className="hidden sm:inline">Dashboard</span>
          </Link>
          
          {currentProject && pathnames.includes("projects") && (
            <>
              <ChevronRight className="h-4 w-4 opacity-50" />
              <Link
                to={`/projects/${currentProject.id}/targets`}
                className="flex items-center gap-1 text-primary font-bold hover:text-primary/80 transition-colors"
              >
                <Layout className="h-4 w-4" />
                <span>{currentProject.name}</span>
              </Link>
            </>
          )}
          
          {pathnames.length > 0 && !pathnames.includes("projects") && (
             <>
               <ChevronRight className="h-4 w-4 opacity-50" />
               <span className="text-foreground font-semibold capitalize">
                 {pathnames[pathnames.length - 1].replace(/-/g, " ")}
               </span>
             </>
          )}
        </nav>

        <div className="ml-auto flex items-center gap-4">
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-white/5 border border-white/10 text-[10px] font-black uppercase tracking-widest text-muted-foreground">
            <span className="h-2 w-2 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]" />
            Live
          </div>
        </div>
      </div>
    </header>
  );
}
