"use client";

import { useCallback, useEffect, useState } from "react";
import {
  type Column,
  type ColumnDef,
  type PaginationState,
  type SortingState,
  type Table as TableInstance,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import {
  ArrowUpDown,
  ChevronLeft,
  ChevronRight,
  Inbox,
  RefreshCw,
  Search,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Page, PageRequest } from "@/lib/api";

const DEFAULT_PAGE_SIZE = 25;

/** Sortable column header button, for use in column definitions. */
export function SortButton<T>({
  column,
  label,
}: {
  column: Column<T, unknown>;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
      className="hover:text-foreground inline-flex items-center gap-1.5 transition-colors"
    >
      {label}
      <ArrowUpDown className="size-3.5 opacity-60" />
    </button>
  );
}

export interface RowDetail {
  title: React.ReactNode;
  description?: React.ReactNode;
  body: React.ReactNode;
}

function useDebounced<T>(value: T, ms: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return debounced;
}

interface ShellProps<T> {
  table: TableInstance<T>;
  columnCount: number;
  loading: boolean;
  error: string | null;
  emptyMessage: string;
  searchPlaceholder: string;
  globalFilter: string;
  setGlobalFilter: (v: string) => void;
  refresh: () => void;
  total: number;
  onRowClick?: (row: T) => void;
  renderDetail?: (row: T) => RowDetail;
}

/** Presentational shell shared by the client- and server-paginated tables. */
function TableShell<T>({
  table,
  columnCount,
  loading,
  error,
  emptyMessage,
  searchPlaceholder,
  globalFilter,
  setGlobalFilter,
  refresh,
  total,
  onRowClick,
  renderDetail,
}: ShellProps<T>) {
  const [detail, setDetail] = useState<RowDetail | null>(null);

  const pageRows = table.getRowModel().rows;
  const { pageIndex, pageSize } = table.getState().pagination;
  const from = total === 0 ? 0 : pageIndex * pageSize + 1;
  const to = pageIndex * pageSize + pageRows.length;
  const pageCount = table.getPageCount();

  const clickable = Boolean(onRowClick || renderDetail);
  const handleClick = (row: T) => {
    if (onRowClick) onRowClick(row);
    else if (renderDetail) setDetail(renderDetail(row));
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative w-full max-w-sm">
          <Search className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
          <Input
            value={globalFilter}
            onChange={(e) => setGlobalFilter(e.target.value)}
            placeholder={searchPlaceholder}
            className="pl-8"
            aria-label="Search"
          />
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={refresh}
          disabled={loading}
          className="ml-auto"
        >
          <RefreshCw className={loading ? "animate-spin" : ""} />
          Refresh
        </Button>
      </div>

      <Card className="overflow-hidden py-0">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((hg) => (
              <TableRow key={hg.id} className="hover:bg-transparent">
                {hg.headers.map((header) => (
                  <TableHead key={header.id} className="h-11">
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                          header.column.columnDef.header,
                          header.getContext(),
                        )}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 6 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: columnCount }).map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-5 w-full max-w-[180px]" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : error ? (
              <TableRow>
                <TableCell
                  colSpan={columnCount}
                  className="text-destructive py-10 text-center"
                >
                  Failed to load: {error}
                </TableCell>
              </TableRow>
            ) : pageRows.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={columnCount}
                  className="text-muted-foreground py-12 text-center"
                >
                  <Inbox className="mx-auto mb-2 size-6 opacity-50" />
                  {globalFilter ? "No rows match your search." : emptyMessage}
                </TableCell>
              </TableRow>
            ) : (
              pageRows.map((row) => (
                <TableRow
                  key={row.id}
                  onClick={clickable ? () => handleClick(row.original) : undefined}
                  className={clickable ? "cursor-pointer" : undefined}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className="py-3 align-top">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Card>

      <div className="flex items-center justify-between gap-2">
        <p className="text-muted-foreground text-sm">
          {loading ? "Loading…" : `${from}–${to} of ${total}`}
        </p>
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground text-sm">
            Page {pageCount === 0 ? 0 : pageIndex + 1} of {pageCount}
          </span>
          <Button
            variant="outline"
            size="icon"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
            aria-label="Previous page"
          >
            <ChevronLeft />
          </Button>
          <Button
            variant="outline"
            size="icon"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
            aria-label="Next page"
          >
            <ChevronRight />
          </Button>
        </div>
      </div>

      {renderDetail && (
        <Dialog
          open={detail != null}
          onOpenChange={(open) => {
            if (!open) setDetail(null);
          }}
        >
          <DialogContent className="flex max-h-[85vh] flex-col gap-0 p-0 sm:max-w-2xl">
            <DialogHeader className="border-b p-4">
              <DialogTitle className="pr-8">{detail?.title}</DialogTitle>
              {detail?.description != null && (
                <DialogDescription className="font-mono text-xs break-all">
                  {detail.description}
                </DialogDescription>
              )}
            </DialogHeader>
            <div className="flex flex-col gap-4 overflow-y-auto p-4">
              {detail?.body}
            </div>
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Client-side table: loads the full array once, paginates/sorts/filters in JS.
// ---------------------------------------------------------------------------

export interface DataTableProps<T> {
  load: () => Promise<T[]>;
  columns: ColumnDef<T>[];
  /** Returns the searchable text for a row (used by the global filter). */
  searchText: (row: T) => string;
  searchPlaceholder?: string;
  emptyMessage?: string;
  pageSize?: number;
  initialSorting?: SortingState;
  onRowClick?: (row: T) => void;
  renderDetail?: (row: T) => RowDetail;
}

export function DataTable<T>({
  load,
  columns,
  searchText,
  searchPlaceholder = "Search…",
  emptyMessage = "Nothing here yet.",
  pageSize = DEFAULT_PAGE_SIZE,
  initialSorting = [],
  onRowClick,
  renderDetail,
}: DataTableProps<T>) {
  const [data, setData] = useState<T[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [globalFilter, setGlobalFilter] = useState("");
  const [sorting, setSorting] = useState<SortingState>(initialSorting);

  const refresh = useCallback(() => {
    setLoading(true);
    setError(null);
    load()
      .then(setData)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [load]);

  useEffect(refresh, [refresh]);

  const table = useReactTable({
    data,
    columns,
    state: { globalFilter, sorting },
    onGlobalFilterChange: setGlobalFilter,
    onSortingChange: setSorting,
    globalFilterFn: (row, _columnId, value) => {
      const needle = String(value).toLowerCase().trim();
      if (!needle) return true;
      return searchText(row.original).toLowerCase().includes(needle);
    },
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    initialState: { pagination: { pageSize } },
  });

  return (
    <TableShell
      table={table}
      columnCount={columns.length}
      loading={loading}
      error={error}
      emptyMessage={emptyMessage}
      searchPlaceholder={searchPlaceholder}
      globalFilter={globalFilter}
      setGlobalFilter={setGlobalFilter}
      refresh={refresh}
      total={table.getFilteredRowModel().rows.length}
      onRowClick={onRowClick}
      renderDetail={renderDetail}
    />
  );
}

// ---------------------------------------------------------------------------
// Server-side table: pagination, search, and sort all happen on the backend.
// One page of rows is held at a time; `total` comes from the server.
// ---------------------------------------------------------------------------

export interface ServerDataTableProps<T> {
  loadPage: (req: PageRequest) => Promise<Page<T>>;
  columns: ColumnDef<T>[];
  searchPlaceholder?: string;
  emptyMessage?: string;
  pageSize?: number;
  initialSorting?: SortingState;
  onRowClick?: (row: T) => void;
  renderDetail?: (row: T) => RowDetail;
}

export function ServerDataTable<T>({
  loadPage,
  columns,
  searchPlaceholder = "Search…",
  emptyMessage = "Nothing here yet.",
  pageSize = DEFAULT_PAGE_SIZE,
  initialSorting = [],
  onRowClick,
  renderDetail,
}: ServerDataTableProps<T>) {
  const [data, setData] = useState<T[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [globalFilter, setGlobalFilter] = useState("");
  const [sorting, setSorting] = useState<SortingState>(initialSorting);
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize,
  });
  const [reloadKey, setReloadKey] = useState(0);

  const debouncedFilter = useDebounced(globalFilter, 300);

  // A new search or sort resets to the first page so the user sees the start of
  // the new result set rather than an out-of-range offset.
  useEffect(() => {
    setPagination((p) => (p.pageIndex === 0 ? p : { ...p, pageIndex: 0 }));
  }, [debouncedFilter, sorting]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    const active = sorting[0];
    loadPage({
      limit: pagination.pageSize,
      offset: pagination.pageIndex * pagination.pageSize,
      q: debouncedFilter || undefined,
      sort: active?.id,
      order: active ? (active.desc ? "desc" : "asc") : undefined,
    })
      .then((res) => {
        if (cancelled) return;
        setData(res.items);
        setTotal(res.total);
      })
      .catch((err: Error) => {
        if (!cancelled) setError(err.message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [
    loadPage,
    pagination.pageIndex,
    pagination.pageSize,
    debouncedFilter,
    sorting,
    reloadKey,
  ]);

  const table = useReactTable({
    data,
    columns,
    state: { sorting, pagination, globalFilter },
    manualPagination: true,
    manualSorting: true,
    manualFiltering: true,
    rowCount: total,
    onSortingChange: setSorting,
    onPaginationChange: setPagination,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
  });

  const refresh = useCallback(() => setReloadKey((k) => k + 1), []);

  return (
    <TableShell
      table={table}
      columnCount={columns.length}
      loading={loading}
      error={error}
      emptyMessage={emptyMessage}
      searchPlaceholder={searchPlaceholder}
      globalFilter={globalFilter}
      setGlobalFilter={setGlobalFilter}
      refresh={refresh}
      total={total}
      onRowClick={onRowClick}
      renderDetail={renderDetail}
    />
  );
}
