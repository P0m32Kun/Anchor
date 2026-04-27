import { Link, useLocation } from "react-router-dom";

const navItems = [
  { path: "/", label: "项目" },
  { path: "/runs", label: "运行" },
];

export function Navbar() {
  const location = useLocation();

  return (
    <nav className="nav-glass-dark sticky top-0 z-50">
      <div className="flex items-center h-14 px-5 gap-1 max-w-6xl mx-auto">
        <Link
          to="/"
          className="flex items-center gap-2.5 mr-8 text-text-primary font-semibold text-[15px] tracking-tight group"
        >
          <div className="p-1.5 rounded-lg bg-brand-primary/10 border border-brand-primary/20 transition-all duration-300 group-hover:bg-brand-primary/20 group-hover:shadow-[0_0_15px_rgba(47,129,247,0.25)]">
            <svg
              className="w-4.5 h-4.5 text-brand-primary"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M12 2L2 7l10 5 10-5-10-5z" />
              <path d="M2 17l10 5 10-5" />
              <path d="M2 12l10 5 10-5" />
            </svg>
          </div>
          Anchor
        </Link>

        {navItems.map((item) => {
          const isActive =
            item.path === "/"
              ? location.pathname === "/"
              : location.pathname.startsWith(item.path);
          return (
            <Link
              key={item.path}
              to={item.path}
              aria-current={isActive ? "page" : undefined}
              className={`relative px-3.5 py-2 text-[13px] font-medium rounded-lg transition-all duration-200 ${
                isActive
                  ? "text-text-primary bg-white/[0.06]"
                  : "text-text-tertiary hover:text-text-secondary hover:bg-white/[0.04]"
              }`}
            >
              {item.label}
              {isActive && (
                <span className="absolute bottom-0 left-1/2 -translate-x-1/2 w-4 h-[2px] bg-brand-primary rounded-full shadow-[0_0_8px_rgba(47,129,247,0.6)]" />
              )}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
