"use client";

import { IconChevronLeft, IconChevronRight, IconChevronsLeft, IconChevronsRight, IconChevronUp, IconChevronDown } from "@tabler/icons-react";
import React from "react";

export interface DataTableColumn<T> {
  key: string;
  header: string;
  cell: (row: T) => React.ReactNode;
  sortable?: boolean;
}

interface DataTableProps<T> {
  data: T[];
  columns: DataTableColumn<T>[];
  loading?: boolean;
  error?: Error | null;
  onRetry?: () => void;
}

export default function DataTable<T>({ data, columns, loading, error, onRetry }: DataTableProps<T>) {
  const [page, setPage] = React.useState(0);
  const [sortKey, setSortKey] = React.useState<string | null>(null);
  const [sortDir, setSortDir] = React.useState<"asc" | "desc">("desc");
  const pageSize = 10;

  const sorted = React.useMemo(() => {
    if (!sortKey || !data) return data || [];
    const col = columns.find((c) => c.key === sortKey);
    if (!col || !col.sortable) return data || [];
    return [...data].sort((a: T, b: T) => {
      const av = String((a as Record<string, unknown>)[sortKey] ?? "");
      const bv = String((b as Record<string, unknown>)[sortKey] ?? "");
      return sortDir === "asc" ? av.localeCompare(bv) : bv.localeCompare(av);
    });
  }, [data, sortKey, sortDir, columns]);

  const totalPages = Math.max(1, Math.ceil(sorted.length / pageSize));
  const paginated = sorted.slice(page * pageSize, (page + 1) * pageSize);

  const toggleSort = (key: string) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("desc");
    }
    setPage(0);
  };

  if (loading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="h-10 bg-pat-bg-surface-secondary/50 rounded animate-pulse" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-12 border border-pat-table-border rounded-lg bg-pat-bg-surface/50">
        <div className="text-pat-danger text-sm mb-2">Failed to load data</div>
        <div className="text-pat-text-muted text-xs mb-4">{error.message}</div>
        {onRetry && (
          <button onClick={onRetry} className="text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary px-3 py-1.5 rounded transition-colors">
            Retry
          </button>
        )}
      </div>
    );
  }

  if (!data?.length) {
    return (
      <div className="text-center py-12 border border-pat-table-border rounded-lg bg-pat-bg-surface/50">
        <div className="text-pat-text-muted text-sm">No data found</div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="overflow-x-auto border border-pat-table-border rounded-lg">
        <table className="w-full text-sm text-left">
          <thead className="bg-pat-bg-surface text-pat-text-secondary uppercase text-xs">
            <tr>
              {columns.map((col) => (
                <th
                  key={col.key}
                  className={`px-4 py-3 font-medium border-b border-pat-border whitespace-nowrap ${col.sortable ? "cursor-pointer hover:text-pat-text-primary select-none" : ""}`}
                  onClick={() => col.sortable && toggleSort(col.key)}
                >
                  <div className="flex items-center gap-1">
                    {col.header}
                    {col.sortable && sortKey === col.key && (
                      sortDir === "asc" ? <IconChevronUp size={12} /> : <IconChevronDown size={12} />
                    )}
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-800">
            {paginated.map((row, idx) => (
              <tr key={idx} className="hover:bg-pat-table-hover transition-colors">
                {columns.map((col) => (
                  <td key={col.key} className="px-4 py-3 whitespace-nowrap text-pat-text-primary">
                    {col.cell(row)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-between text-xs text-pat-text-secondary">
        <div>
          Showing {Math.min(sorted.length, page * pageSize + 1)} to{" "}
          {Math.min((page + 1) * pageSize, sorted.length)} of{" "}
          {sorted.length} entries
        </div>
        <div className="flex items-center gap-1">
          <button onClick={() => setPage(0)} disabled={page <= 0} className="p-1 rounded hover:bg-pat-bg-surface-secondary disabled:opacity-30 disabled:cursor-not-allowed">
            <IconChevronsLeft size={16} />
          </button>
          <button onClick={() => setPage(p => p - 1)} disabled={page <= 0} className="p-1 rounded hover:bg-pat-bg-surface-secondary disabled:opacity-30 disabled:cursor-not-allowed">
            <IconChevronLeft size={16} />
          </button>
          <span className="px-2">Page {page + 1} of {totalPages}</span>
          <button onClick={() => setPage(p => p + 1)} disabled={page >= totalPages - 1} className="p-1 rounded hover:bg-pat-bg-surface-secondary disabled:opacity-30 disabled:cursor-not-allowed">
            <IconChevronRight size={16} />
          </button>
          <button onClick={() => setPage(totalPages - 1)} disabled={page >= totalPages - 1} className="p-1 rounded hover:bg-pat-bg-surface-secondary disabled:opacity-30 disabled:cursor-not-allowed">
            <IconChevronsRight size={16} />
          </button>
        </div>
      </div>
    </div>
  );
}
