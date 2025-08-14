import React, { useState, forwardRef } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Trash2, Copy, FileVideo2Icon, AlertCircle, Move } from "lucide-react";
import { SpoilerService, Movie } from "@bindings/changeme/backend";

import { DndContext, closestCenter, PointerSensor, useSensor, useSensors } from "@dnd-kit/core";
import { arrayMove, SortableContext, verticalListSortingStrategy, useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";

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
}

// Forward ref to real <tr> element so DnD works
const TableRowWrapper = forwardRef<HTMLTableRowElement, React.ComponentProps<"tr">>((props, ref) => {
  return <tr ref={ref} {...props} />;
});
TableRowWrapper.displayName = "TableRowWrapper";

// Sortable row
function SortableRow({
  movie,
  index,
  selectedMovie,
  setSelectedMovie,
  localSpoilers,
  handleRowHover,
  onRemoveMovie,
  onCopyMovieResult,
  getProcessingBadge,
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: movie.id });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <TableRowWrapper
      ref={setNodeRef}
      style={style}
      {...attributes}
      className={`border-white/5 hover:bg-white/2 relative`}
      onClick={() => setSelectedMovie(movie)}
      onMouseEnter={() => movie.processingState === "completed" && handleRowHover(movie.id)}
    >
      <TableCell className="text-slate-300 flex items-center gap-2">
        <div {...listeners} className="cursor-grab hover:cursor-grabbing">
          <Move className="w-4 h-4" />
        </div>
        {index + 1}
      </TableCell>

      <TableCell className="font-medium text-white">
        <div className="max-w-[500px] truncate">
          {movie.processingState === "completed" ? (
            <HoverCard>
              <HoverCardTrigger asChild>
                <span className="cursor-pointer hover:underline inline-block w-full truncate">{movie.fileName}</span>
              </HoverCardTrigger>
              <HoverCardContent className="w-150 bg-black/90 border-white/10" side="right">
                <div className="space-y-2">
                  <h4 className="text-sm font-semibold text-white">Spoiler Preview</h4>
                  <pre className="text-xs text-slate-300 whitespace-pre-wrap font-mono bg-black/40 p-3 rounded border border-white/5 max-h-60 overflow-y-auto">
                    {localSpoilers[movie.id] || "Loading..."}
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
      </TableCell>

      <TableCell className="text-slate-300">{movie.fileSize}</TableCell>
      <TableCell className="text-slate-300">{movie.duration}</TableCell>
      <TableCell className="text-slate-300">
        {movie.width}x{movie.height}
      </TableCell>
      <TableCell>
        {movie.screenshotUrls && movie.screenshotUrls.length > 0 ? (
          <Badge variant="outline" className="border-green-400/50 text-green-400">
            {movie.screenshotUrls.length} shots
          </Badge>
        ) : (
          <Badge variant="outline" className="border-slate-400/50 text-slate-400">
            No shots
          </Badge>
        )}
      </TableCell>
      <TableCell>{getProcessingBadge(movie.processingState, movie.processingError)}</TableCell>
      <TableCell>
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
      </TableCell>
    </TableRowWrapper>
  );
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
}: MovieTableProps) {
  const [selectedMovie, setSelectedMovie] = useState<Movie | null>(null);
  const [localSpoilers, setLocalSpoilers] = useState<Record<string, string>>({});

  const sensors = useSensors(useSensor(PointerSensor));

  const handleDragEnd = (event: any) => {
    const { active, over } = event;
    if (over && active.id !== over.id) {
      const oldIndex = movies.findIndex((m) => m.id === active.id);
      const newIndex = movies.findIndex((m) => m.id === over.id);
      const newMovies = arrayMove(movies, oldIndex, newIndex);
      onReorderMovies(newMovies);
    }
  };

  const handleRowHover = async (movieId: string) => {
    try {
      const spoiler = await SpoilerService.GenerateResultForMovie(movieId);
      setLocalSpoilers((prev) => ({ ...prev, [movieId]: spoiler }));
    } catch (error) {
      console.error("Failed to generate spoiler preview:", error);
    }
  };

  const getProcessingBadge = (state: string, processingError?: string) => {
    switch (state) {
      case "pending":
        return (
          <Badge variant="outline" className="border-yellow-400/50 text-yellow-400">
            Pending
          </Badge>
        );
      case "analyzing_media":
        return (
          <Badge variant="outline" className="border-blue-400/50 text-blue-400">
            Analyzing Media
          </Badge>
        );
      case "waiting_for_screenshot_slot":
        return (
          <Badge variant="outline" className="border-orange-400/50 text-orange-400">
            Waiting for Screenshot Slot
          </Badge>
        );
      case "generating_screenshots":
        return (
          <Badge variant="outline" className="border-purple-400/50 text-purple-400">
            Generating Screenshots
          </Badge>
        );
      case "waiting_for_upload_slot":
        return (
          <Badge variant="outline" className="border-amber-400/50 text-amber-400">
            Waiting for Upload Slot
          </Badge>
        );
      case "uploading_screenshots":
        return (
          <Badge variant="outline" className="border-cyan-400/50 text-cyan-400">
            Uploading Screenshots
          </Badge>
        );
      case "completed":
        return (
          <Badge variant="outline" className="border-green-400/50 text-green-400">
            Complete
          </Badge>
        );
      case "error":
        return (
          <div className="flex items-center gap-1">
            <Badge variant="outline" className="border-red-400/50 text-red-400">
              Error
            </Badge>
            {processingError && (
              <div className="text-xs text-red-400">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <AlertCircle className="w-3 h-3 cursor-pointer" />
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>{processingError}</p>
                  </TooltipContent>
                </Tooltip>
              </div>
            )}
          </div>
        );
      default:
        return null;
    }
  };

  const completedMovies = movies.filter((m) => m.processingState === "completed");

  return (
    <Card className="bg-black/10 border-white/5">
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-white flex items-center gap-2">
            <FileVideo2Icon className="w-5 h-5" />
            Files ({movies.length})
          </CardTitle>
          <div className="flex items-center gap-2">
            {pendingCount > 0 && !processing && (
              <Button onClick={onStartProcessing} className="bg-gradient-to-r from-green-600 to-emerald-600">
                Start Processing ({pendingCount})
              </Button>
            )}
            {processing && (
              <Button onClick={onCancelProcessing} variant="outline" className="border-red-400/50 text-red-400 hover:bg-red-500/20">
                Cancel
              </Button>
            )}
            <Button onClick={onClearMovies} variant="outline" className="border-white/20 hover:bg-red-500/20">
              Clear All
            </Button>
            {completedMovies.length > 0 && (
              <Button onClick={onCopyAllResults} className="bg-gradient-to-r from-green-600 to-emerald-600">
                Copy All ({completedMovies.length})
              </Button>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="wails-no-drag">
        <ScrollArea className="h-[400px]">
          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
            <SortableContext items={movies.map((m) => m.id)} strategy={verticalListSortingStrategy}>
              <Table>
                <TableHeader>
                  <TableRowWrapper className="border-white/5">
                    <TableHead className="text-slate-300">#</TableHead>
                    <TableHead className="text-slate-300">File Name</TableHead>
                    <TableHead className="text-slate-300">Size</TableHead>
                    <TableHead className="text-slate-300">Duration</TableHead>
                    <TableHead className="text-slate-300">Resolution</TableHead>
                    <TableHead className="text-slate-300">Screenshots</TableHead>
                    <TableHead className="text-slate-300">Status</TableHead>
                    <TableHead className="text-slate-300 w-24"></TableHead>
                  </TableRowWrapper>
                </TableHeader>
                <TableBody>
                  {movies.map((movie, index) => (
                    <SortableRow
                      key={movie.id}
                      movie={movie}
                      index={index}
                      selectedMovie={selectedMovie}
                      setSelectedMovie={setSelectedMovie}
                      localSpoilers={localSpoilers}
                      handleRowHover={handleRowHover}
                      onRemoveMovie={onRemoveMovie}
                      onCopyMovieResult={onCopyMovieResult}
                      getProcessingBadge={getProcessingBadge}
                    />
                  ))}
                </TableBody>
              </Table>
            </SortableContext>
          </DndContext>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
