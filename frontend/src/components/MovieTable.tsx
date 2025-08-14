import { useState } from 'react'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card"
import { Trash2, Copy, FileVideo2Icon, Play, Square, AlertCircle } from "lucide-react"
import { SpoilerService, Movie } from "@bindings/changeme/backend"

interface MovieTableProps {
  movies: Movie[]
  processing: boolean
  pendingCount: number
  onStartProcessing: () => void
  onCancelProcessing: () => void
  onClearMovies: () => void
  onRemoveMovie: (id: number) => void
  onCopyMovieResult: (id: number) => void
}

export default function MovieTable({
  movies,
  processing,
  pendingCount,
  onStartProcessing,
  onCancelProcessing,
  onClearMovies,
  onRemoveMovie,
  onCopyMovieResult
}: MovieTableProps) {
  const [selectedMovie, setSelectedMovie] = useState<Movie | null>(null)
  const [hoveredMovieId, setHoveredMovieId] = useState<number | null>(null)
  const [hoveredSpoiler, setHoveredSpoiler] = useState<string>('')

  const handleRowHover = async (movieId: number) => {
    if (hoveredMovieId === movieId) return
    
    setHoveredMovieId(movieId)
    try {
      const spoiler = await SpoilerService.GenerateResultForMovie(movieId)
      setHoveredSpoiler(spoiler)
    } catch (error) {
      console.error('Failed to generate spoiler preview:', error)
    }
  }

  const getProcessingBadge = (state: string, processingError?: string) => {
    switch (state) {
      case 'pending':
        return <Badge variant="outline" className="border-yellow-400/50 text-yellow-400">Pending</Badge>
      case 'analyzing_media':
        return <Badge variant="outline" className="border-blue-400/50 text-blue-400">Analyzing Media</Badge>
      case 'generating_screenshots':
        return <Badge variant="outline" className="border-purple-400/50 text-purple-400">Generating Screenshots</Badge>
      case 'uploading_screenshots':
        return <Badge variant="outline" className="border-cyan-400/50 text-cyan-400">Uploading Screenshots</Badge>
      case 'completed':
        return <Badge variant="outline" className="border-green-400/50 text-green-400">Complete</Badge>
      case 'error':
        return (
          <div className="flex items-center gap-1">
            <Badge variant="outline" className="border-red-400/50 text-red-400">Error</Badge>
            {processingError && (
              <div className="text-xs text-red-400" title={processingError}>
                <AlertCircle className="w-3 h-3" />
              </div>
            )}
          </div>
        )
      default:
        return null
    }
  }

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
              <Button 
                onClick={onStartProcessing}
                className="bg-gradient-to-r from-green-600 to-emerald-600"
              >
                <Play className="w-4 h-4 mr-2" />
                Start Processing ({pendingCount})
              </Button>
            )}
            {processing && (
              <Button 
                onClick={onCancelProcessing}
                variant="outline" 
                className="border-red-400/50 text-red-400 hover:bg-red-500/20"
              >
                <Square className="w-4 h-4 mr-2" />
                Cancel
              </Button>
            )}
            <Button onClick={onClearMovies} variant="outline" className="border-white/20 hover:bg-red-500/20">
              <Trash2 className="w-4 h-4 mr-2" />
              Clear All
            </Button>
            
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-[400px]">
          <Table>
            <TableHeader>
              <TableRow className="border-white/5">
                <TableHead className="text-slate-300">#</TableHead>
                <TableHead className="text-slate-300">File Name</TableHead>
                <TableHead className="text-slate-300">Size</TableHead>
                <TableHead className="text-slate-300">Duration</TableHead>
                <TableHead className="text-slate-300">Resolution</TableHead>
                <TableHead className="text-slate-300">Screenshots</TableHead>
                <TableHead className="text-slate-300">Status</TableHead>
                <TableHead className="text-slate-300 w-24"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {movies.map((movie, index) => (
                <TableRow 
                  key={movie.id}
                  className={`border-white/5 hover:bg-white/2 cursor-pointer relative ${
                    selectedMovie?.id === movie.id ? "bg-purple-500/10" : ""
                  }`}
                  onClick={() => setSelectedMovie(movie)}
                  onMouseEnter={() => movie.processingState === 'completed' && handleRowHover(movie.id)}
                >
                  <TableCell className="text-slate-300">{index + 1}</TableCell>
                  <TableCell className="font-medium text-white">
                    {movie.processingState === 'completed' ? (
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <span className="cursor-pointer hover:underline">{movie.fileName}</span>
                        </HoverCardTrigger>
                        <HoverCardContent className="w-96 bg-black/90 border-white/10" side="right">
                          <div className="space-y-2">
                            <h4 className="text-sm font-semibold text-white">Spoiler Preview</h4>
                            <pre className="text-xs text-slate-300 whitespace-pre-wrap font-mono bg-black/40 p-3 rounded border border-white/5 max-h-60 overflow-y-auto">
                              {hoveredMovieId === movie.id ? hoveredSpoiler : 'Loading...'}
                            </pre>
                          </div>
                        </HoverCardContent>
                      </HoverCard>
                    ) : (
                      movie.fileName
                    )}
                  </TableCell>
                  <TableCell className="text-slate-300">{movie.fileSize}</TableCell>
                  <TableCell className="text-slate-300">{movie.duration}</TableCell>
                  <TableCell className="text-slate-300">{movie.width}x{movie.height}</TableCell>
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
                  <TableCell>
                    {getProcessingBadge(movie.processingState, movie.processingError)}
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      {movie.processingState === 'completed' && (
                        <Button 
                          size="sm" 
                          variant="ghost"
                          className="hover:bg-green-500/20 hover:text-green-400"
                          onClick={(e) => {
                            e.stopPropagation()
                            onCopyMovieResult(movie.id)
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
                          e.stopPropagation()
                          onRemoveMovie(movie.id)
                        }}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </ScrollArea>
      </CardContent>
    </Card>
  )
}