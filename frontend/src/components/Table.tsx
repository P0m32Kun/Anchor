import React from "react";
import { Skeleton } from "./Skeleton";

export interface TableColumn<T> {
  key: string;
  header: string;
  render?: (row: T) => React.ReactNode;
  width?: string;
}

export interface TableProps<T> {
  columns: TableColumn<T>[];
  data: T[];
  loading?: boolean;
  emptyText?: string;
  onRowClick?: (row: T) => void;
  className?: string;
  maxHeight?: number | string;
}

export function Table<T extends Record<string, unknown>>({
  columns,
  data,
  loading = false,
  emptyText = "暂无数据",
  onRowClick,
  className = "",
  maxHeight,
}: TableProps<T>) {
  const hasData = data.length > 0;
  const isClickable = !!onRowClick;

  const containerStyle = maxHeight
    ? { maxHeight: typeof maxHeight === "number" ? `${maxHeight}px` : maxHeight }
    : undefined;

  return (
    <div className={`panel overflow-auto ${className}`} style={containerStyle}>
      <table className="w-full text-left border-collapse">
        <thead className="sticky top-0 z-10">
          <tr className="border-b border-white/[0.06] bg-white/[0.03]">
            {columns.map((col) => (
              <th
                key={col.key}
                className="px-4 py-3 text-xs font-semibold text-text-tertiary uppercase tracking-wider whitespace-nowrap"
                style={col.width ? { width: col.width } : undefined}
              >
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {loading && (
            <>
              {Array.from({ length: 3 }).map((_, rowIdx) => (
                <tr key={`loading-${rowIdx}`}>
                  {columns.map((col) => (
                    <td key={col.key} className="px-4 py-3">
                      <Skeleton className="h-4 w-3/4" />
                    </td>
                  ))}
                </tr>
              ))}
            </>
          )}

          {!loading && hasData &&
            data.map((row, rowIdx) => (
              <tr
                key={rowIdx}
                onClick={isClickable ? () => onRowClick!(row) : undefined}
                className={`
                  border-b border-white/[0.04] relative
                  transition-colors duration-150
                  ${isClickable ? "cursor-pointer hover:bg-white/[0.06] hover:shadow-[inset_3px_0_0_0_#00d4ff]" : "hover:bg-white/[0.04]"}
                `}
              >
                {columns.map((col) => (
                  <td
                    key={col.key}
                    className="px-4 py-3 text-sm text-text-primary whitespace-nowrap"
                  >
                    {col.render
                      ? col.render(row)
                      : String((row as Record<string, unknown>)[col.key] ?? "-")}
                  </td>
                ))}
              </tr>
            ))}

          {!loading && !hasData && (
            <tr>
              <td colSpan={columns.length}>
                <div className="flex flex-col items-center justify-center py-16 text-center animate-fade-in">
                  <svg
                    className="w-12 h-12 text-text-quaternary mb-4"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <rect x="3" y="3" width="18" height="18" rx="2" />
                    <path d="M3 9h18" />
                    <path d="M9 21V9" />
                  </svg>
                  <p className="text-sm text-text-tertiary">{emptyText}</p>
                </div>
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
