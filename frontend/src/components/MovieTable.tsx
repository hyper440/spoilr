import { useState, useEffect } from "react";
import { ColumnDef, SortingState, flexRender, getCoreRowModel, getSortedRowModel, useReactTable } from "@tanstack/react-table";
import { ArrowUpDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Trash2, Copy, FileVideo2Icon, AlertCircle, AlertTriangle } from "lucide-react";
import { SpoilerService, Movie } from "@bindings/spoilr/backend";
import { useTranslation } from "@/contexts/LanguageContext";

interface MovieTableProps {
  movies: Movie[];
  processing: boolean;
  pendingCount: number;
  onStartProcessing: () => void;
  onCancelProcessing: () => void;
  onClearMovies: () => void;
  onRemoveMovie: (id: string) => void;
  onCopyMovieResult: (id: string) => void;
  onCopyAllResults: () => void;
  onReorderMovies: (newMovies: Movie[]) => void;
  onSortingChange?: (sorting: SortingState) => void;
}

export default function MovieTable({
  movies,
  processing,
  pendingCount,
  onStartProcessing,
  onCancelProcessing,
  onClearMovies,
  onRemoveMovie,
  onCopyMovieResult,
  onCopyAllResults,
  onReorderMovies,
  onSortingChange,
}: MovieTableProps) {
  const { t } = useTranslation();
  const [localSpoilers, setLocalSpoilers] = useState<Record<string, string>>({});
  const [sorting, setSorting] = useState<SortingState>([{ id: "fileName", desc: false }]);

  const handleRowHover = async (movieId: string) => {
    try {
      const spoiler = await SpoilerService.GenerateResultForMovie(movieId);
      setLocalSpoilers((prev) => ({ ...prev, [movieId]: spoiler }));
    } catch (error) {
      console.error(t("errors.generateSpoiler"), error);
    }
  };

  const getProcessingBadge = (state: string, processingError?: string, errors?: string[]) => {
    const hasWarnings = errors && errors.length > 0;

    const renderErrorIcon = () => {
      if (!processingError && !hasWarnings) return null;

      const iconClass = "w-3 h-3 cursor-pointer ml-1";
      const Icon = state === "error" ? AlertCircle : AlertTriangle;
      const iconColor = state === "error" ? "text-red-400" : "text-yellow-400";

      const tooltipContent = (
        <div className="max-w-xs">
          {processingError && (
            <div className="mb-2">
              <div className="font-semibold text-red-400">Processing Error:</div>
              <div>{processingError}</div>
            </div>
          )}
          {hasWarnings && (
            <div>
              <div className="font-semibold text-yellow-400">
                {errors.length} Warning{errors.length > 1 ? "s" : ""}:
              </div>
              <ul className="list-disc list-inside space-y-1 mt-1">
                {errors.map((error, index) => (
                  <li key={index} className="text-xs">
                    {error}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      );

      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <Icon className={`${iconClass} ${iconColor}`} />
          </TooltipTrigger>
          <TooltipContent>{tooltipContent}</TooltipContent>
        </Tooltip>
      );
    };

    switch (state) {
      case "pending":
        return (
          <div className="flex items-center">
            <Badge variant="outline" className="border-yellow-400/50 text-yellow-400">
              {t("movieTable.status.pending")}
            </Badge>
            {renderErrorIcon()}
          </div>
        );
      case "analyzing_media":
        return (
          <Badge variant="outline" className="border-blue-400/50 text-blue-400">
            {t("movieTable.status.analyzingMedia")}
          </Badge>
        );
      case "waiting_for_screenshot_slot":
        return (
          <Badge variant="outline" className="border-orange-400/50 text-orange-400">
            {t("movieTable.status.waitingForScreenshotSlot")}
          </Badge>
        );
      case "generating_screenshots":
        return (
          <Badge variant="outline" className="border-purple-400/50 text-purple-400">
            {t("movieTable.status.generatingScreenshots")}
          </Badge>
        );
      case "waiting_for_upload_slot":
        return (
          <Badge variant="outline" className="border-amber-400/50 text-amber-400">
            {t("movieTable.status.waitingForUploadSlot")}
          </Badge>
        );
      case "uploading_screenshots":
        return (
          <Badge variant="outline" className="border-cyan-400/50 text-cyan-400">
            {t("movieTable.status.uploadingScreenshots")}
          </Badge>
        );
      case "completed":
        return (
          <div className="flex items-center">
            <Badge variant="outline" className="border-green-400/50 text-green-400">
              {t("movieTable.status.completed")}
            </Badge>
            {renderErrorIcon()}
          </div>
        );
      case "error":
        return (
          <div className="flex items-center">
            <Badge variant="outline" className="border-red-400/50 text-red-400">
              {t("movieTable.status.error")}
            </Badge>
            {renderErrorIcon()}
          </div>
        );
      default:
        return null;
    }
  };

  const columns: ColumnDef<Movie>[] = [
    {
      accessorKey: "fileName",
      header: ({ column }) => {
        return (
          <Button
            variant="ghost"
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
            className="text-slate-300 hover:text-white p-0"
          >
            {t("movieTable.headers.fileName")}
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        );
      },
      sortingFn: (rowA, rowB) => {
        return rowA.original.fileName.localeCompare(rowB.original.fileName, undefined, {
          numeric: true,
          sensitivity: "base",
          ignorePunctuation: false,
        });
      },
      cell: ({ row }) => {
        const movie = row.original;
        return (
          <div className="font-medium text-white">
            <div className="max-w-[400px] truncate">
              {movie.processingState === "completed" ? (
                <HoverCard>
                  <HoverCardTrigger asChild>
                    <span className="cursor-pointer hover:underline inline-block w-full truncate">{movie.fileName}</span>
                  </HoverCardTrigger>
                  <HoverCardContent className="w-150 bg-black/90 border-white/10" side="right">
                    <div className="space-y-2 flex">
                      <Button
                        size="sm"
                        variant="ghost"
                        className="hover:bg-green-500/20 hover:text-green-400"
                        onClick={(e) => {
                          e.stopPropagation();
                          onCopyMovieResult(movie.id);
                        }}
                      >
                        <Copy className="w-4 h-4" />
                      </Button>
                      <pre className="text-xs text-slate-300 whitespace-pre-wrap font-mono bg-black/40 p-3 rounded border border-white/5 max-h-60 overflow-y-auto">
                        {localSpoilers[movie.id] || t("movieTable.loading")}
                      </pre>
                    </div>
                  </HoverCardContent>
                </HoverCard>
              ) : (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-block w-full truncate">{movie.fileName}</span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>{movie.fileName}</p>
                  </TooltipContent>
                </Tooltip>
              )}
            </div>
          </div>
        );
      },
    },
    {
      accessorKey: "fileSize",
      header: ({ column }) => {
        return (
          <Button
            variant="ghost"
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
            className="text-slate-300 hover:text-white p-0"
          >
            {t("movieTable.headers.size")}
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        );
      },
      cell: ({ row }) => <div className="text-slate-300">{row.getValue("fileSize")}</div>,
    },
    {
      accessorKey: "duration",
      header: ({ column }) => {
        return (
          <Button
            variant="ghost"
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
            className="text-slate-300 hover:text-white p-0"
          >
            {t("movieTable.headers.duration")}
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        );
      },
      cell: ({ row }) => <div className="text-slate-300">{row.getValue("duration")}</div>,
    },
    {
      id: "resolution",
      header: ({ column }) => {
        return (
          <Button
            variant="ghost"
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
            className="text-slate-300 hover:text-white p-0"
          >
            {t("movieTable.headers.resolution")}
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        );
      },
      accessorFn: (row) => parseInt(row.width) * parseInt(row.height),
      cell: ({ row }) => {
        const movie = row.original;
        return (
          <div className="text-slate-300">
            {movie.width}x{movie.height}
          </div>
        );
      },
    },
    {
      accessorKey: "processingState",
      header: ({ column }) => {
        return (
          <Button
            variant="ghost"
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
            className="text-slate-300 hover:text-white p-0"
          >
            Status
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        );
      },
      cell: ({ row }) => {
        const movie = row.original;
        return getProcessingBadge(movie.processingState, movie.processingError, movie.errors);
      },
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const movie = row.original;
        return (
          <div className="flex gap-1">
            {movie.processingState === "completed" && (
              <Button
                size="sm"
                variant="ghost"
                className="hover:bg-green-500/20 hover:text-green-400"
                onClick={(e) => {
                  e.stopPropagation();
                  onCopyMovieResult(movie.id);
                }}
              >
                <Copy className="w-4 h-4" />
              </Button>
            )}
            <Button
              size="sm"
              variant="ghost"
              className="hover:bg-red-500/20 hover:text-red-400"
              onClick={(e) => {
                e.stopPropagation();
                onRemoveMovie(movie.id);
              }}
            >
              <Trash2 className="w-4 h-4" />
            </Button>
          </div>
        );
      },
      enableSorting: false,
    },
  ];

  const table = useReactTable({
    data: movies,
    columns,
    onSortingChange: (newSorting) => {
      setSorting(newSorting);
      if (onSortingChange) {
        onSortingChange(typeof newSorting === "function" ? newSorting(sorting) : newSorting);
      }
    },
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    state: {
      sorting,
    },
  });

  // Update reordering logic to work with table data
  useEffect(() => {
    const sortedMovies = table.getRowModel().rows.map((row) => row.original);
    const orderChanged = movies.some((movie, index) => movie.id !== sortedMovies[index]?.id);

    if (orderChanged) {
      onReorderMovies(sortedMovies);
    }
  }, [table.getRowModel().rows, movies, onReorderMovies]);

  const completedMovies = movies.filter((m) => m.processingState === "completed");

  return (
    <Card className="bg-black/10 border-white/5 wails-no-drag">
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-white flex items-center gap-2 select-none">
            <FileVideo2Icon className="w-5 h-5" />
            {t("movieTable.title")} ({movies.length})
          </CardTitle>
          <div className="flex items-center gap-2">
            {pendingCount > 0 && !processing && (
              <Button onClick={onStartProcessing} className="bg-gradient-to-r from-green-600 to-emerald-600">
                {t("movieTable.startProcessing")} ({pendingCount})
              </Button>
            )}
            {processing && (
              <Button onClick={onCancelProcessing} variant="outline" className="border-red-400/50 text-red-400 hover:bg-red-500/20">
                {t("movieTable.cancel")}
              </Button>
            )}
            {!processing && (
              <Button onClick={onClearMovies} variant="outline" className="border-white/20 hover:bg-red-500/20">
                {t("movieTable.clearAll")}
              </Button>
            )}
            {movies.length !== pendingCount && !processing && (
              <Tooltip>
                <TooltipTrigger>
                  <Button
                    onClick={SpoilerService.ResetMovieStatuses}
                    variant="outline"
                    className="border-yellow-400/50 text-yellow-400 hover:bg-yellow-500/20"
                  >
                    {t("movieTable.reset")}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("movieTable.resetTooltip")}</TooltipContent>
              </Tooltip>
            )}
            {completedMovies.length > 0 && (
              <Button onClick={onCopyAllResults} className="bg-gradient-to-r from-green-600 to-emerald-600">
                {t("movieTable.copyAll")} ({completedMovies.length})
              </Button>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-[530px]">
          <div className="overflow-hidden rounded-md border border-white/5">
            <Table>
              <TableHeader>
                {table.getHeaderGroups().map((headerGroup) => (
                  <TableRow key={headerGroup.id} className="border-white/5">
                    {headerGroup.headers.map((header) => (
                      <TableHead key={header.id} className="text-slate-300">
                        {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                      </TableHead>
                    ))}
                  </TableRow>
                ))}
              </TableHeader>
              <TableBody>
                {table.getRowModel().rows?.length ? (
                  table.getRowModel().rows.map((row) => (
                    <TableRow
                      key={row.id}
                      className="border-white/5 hover:bg-white/2"
                      onMouseEnter={() => row.original.processingState === "completed" && handleRowHover(row.original.id)}
                    >
                      {row.getVisibleCells().map((cell) => (
                        <TableCell key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</TableCell>
                      ))}
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={columns.length} className="h-24 text-center text-slate-300">
                      No movies added.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
