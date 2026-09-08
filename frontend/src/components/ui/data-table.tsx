import React from "react";
import {
  IconArrowsUpDown, IconChevronDown, IconChevronUp,
  IconChevronLeft, IconChevronRight, IconChevronsLeft, IconChevronsRight,
} from "@tabler/icons-react";

export interface DataTableColumn<T> {
  key: string;
  header: React.ReactNode;
  sortable?: boolean;
  sortFn?: (a: T, b: T) => number;
  cell: (row: T) => React.ReactNode;
}

interface DataTableProps<T> {
  data: T[];
  columns: DataTableColumn<T>[];
  loading?: boolean;
  error?: Error | null;
  onRetry?: () => void;
  /** Rows per table page. When the parent feeds a server-paginated slice,
   *  set this to the server limit so the footer reflects the real page.
   *  Default 10 for client-side tables. */
  pageSize?: number;
  /** Hide the built-in pager when the parent renders its own server pager. */
  hidePager?: boolean;
}

export default function DataTable<T>({ data, columns, loading, error, onRetry, pageSize = 10, hidePager = false }: DataTableProps<T>) {
  const [page, setPage] = React.useState(0);
  const [sortKey, setSortKey] = React.useState<string | null>(null);
  const [sortDir, setSortDir] = React.useState<"asc" | "desc">("desc");

  // Keep the internal page in range when the data slice shrinks (e.g. the
  // parent's server page changed) — otherwise the table can render empty.
  React.useEffect(() => {
    const maxPage = Math.max(0, Math.ceil((data?.length ?? 0) / pageSize) - 1);
    if (page > maxPage) setPage(maxPage);
  }, [data, pageSize, page]);

  const sorted = React.useMemo(() => {
    if (!sortKey || !data) return data || [];
    const col = columns.find((c) => c.key === sortKey);
    if (!col || !col.sortable) return data || [];
    return [...data].sort((a: T, b: T) => {
      if (col.sortFn) {
        return sortDir === "asc" ? col.sortFn(a, b) : col.sortFn(b, a);
      }
      const av = (a as Record<string, unknown>)[sortKey];
      const bv = (b as Record<string, unknown>)[sortKey];
      const aNum = Number(av);
      const bNum = Number(bv);
      if (!isNaN(aNum) && !isNaN(bNum)) {
        return sortDir === "asc" ? aNum - bNum : bNum - aNum;
      }
      const aStr = String(av ?? "");
      const bStr = String(bv ?? "");
      return sortDir === "asc" ? aStr.localeCompare(bStr) : bStr.localeCompare(aStr);
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
        <div className="text-sm text-pat-danger mb-2">Failed to load data</div>
        {onRetry && (
          <button onClick={onRetry} className="text-xs px-3 py-1.5 rounded border border-pat-border bg-pat-bg-surface-secondary hover:bg-pat-bg-surface transition-colors text-pat-text-secondary">
            Retry
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="overflow-x-auto border border-pat-table-border rounded-lg">
        <table className="w-full text-sm text-left">
          <thead className="bg-pat-bg-surface text-pat-text-secondary uppercase text-xs">
            <tr>
              {columns.map((col) => (
                <th
                  key={col.key}
                  className="px-4 py-3 font-medium border-b border-pat-border whitespace-nowrap"
                  aria-sort={col.sortable && sortKey === col.key ? (sortDir === "asc" ? "ascending" : "descending") : undefined}
                >
                  {col.sortable ? (
                    <button
                      onClick={() => toggleSort(col.key)}
                      className="flex items-center gap-1 hover:text-pat-text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-pat-primary rounded"
                    >
                      {col.header}
                      {sortKey === col.key ? (
                        sortDir === "asc" ? <IconChevronUp size={12} /> : <IconChevronDown size={12} />
                      ) : (
                        <IconArrowsUpDown size={12} className="opacity-40" />
                      )}
                    </button>
                  ) : (
                    <div className="flex items-center gap-1">
                      {col.header}
                    </div>
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-pat-border">
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

      {!hidePager && (
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
      )}
    </div>
  );
}