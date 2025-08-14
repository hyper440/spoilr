import { ThemeProvider } from "@/components/theme-provider"

import { useState, useEffect, useCallback } from 'react'
import { SpoilerService, Movie, AppSettings } from "@bindings/changeme/backend"
import { Events, WML } from "@wailsio/runtime"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Progress } from "@/components/ui/progress"
import { Textarea } from "@/components/ui/textarea"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Slider } from "@/components/ui/slider"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card"
import { Trash2, Upload, Copy, Settings, FileVideo2Icon, Edit } from "lucide-react"

interface ProcessProgress {
  current: number
  total: number
  fileName: string
  message: string
  completed: boolean
  error?: string
}

function App() {
  const [movies, setMovies] = useState<Movie[]>([])
  const [settings, setSettings] = useState<AppSettings>({
    centerAlign: false,
    hideEmpty: true,
    uiFontSize: 12,
    listFontSize: 10,
    textFontSize: 12,
    screenshotCount: 6,
    fastpicSid: '',
    screenshotQuality: 2
  })
  const [template, setTemplate] = useState('')
  const [progress, setProgress] = useState<ProcessProgress | null>(null)
  const [selectedMovie, setSelectedMovie] = useState<Movie | null>(null)
  const [hoveredMovieId, setHoveredMovieId] = useState<number | null>(null)
  const [hoveredSpoiler, setHoveredSpoiler] = useState<string>('')
  const [editingTemplate, setEditingTemplate] = useState(false)

  useEffect(() => {
    loadMovies()
    loadSettings()
    loadTemplate()
    
    Events.On('progress', (ev: any) => {
      const data = ev.data as ProcessProgress
      setProgress(data[0])
      // Reload movies on every progress update to ensure UI stays in sync
      loadMovies()
    })
    
    Events.On('moviesUpdated', (ev: any) => {
      const moviesData = ev.data as Movie[]
      setMovies(moviesData)
    })
    
    WML.Reload()
    
    return () => {
      Events.Off('progress')
      Events.Off('moviesUpdated')
    }
  }, [])

  const loadMovies = async () => {
    try {
      const movieList = await SpoilerService.GetMovies()
      setMovies(movieList)
    } catch (error) {
      console.error('Failed to load movies:', error)
    }
  }

  const loadSettings = async () => {
    try {
      const appSettings = await SpoilerService.GetSettings()
      setSettings(appSettings)
    } catch (error) {
      console.error('Failed to load settings:', error)
    }
  }

  const loadTemplate = async () => {
    try {
      const tmpl = await SpoilerService.GetTemplate()
      setTemplate(tmpl)
    } catch (error) {
      console.error('Failed to load template:', error)
    }
  }

  const handleFileSelect = async () => {
    console.log('File dialog would open here')
  }

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
  }, [])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
  }, [])

  const clearMovies = async () => {
    try {
      await SpoilerService.ClearMovies()
      setMovies([])
      setProgress(null)
    } catch (error) {
      console.error('Failed to clear movies:', error)
    }
  }

  const removeMovie = async (id: number) => {
    try {
      await SpoilerService.RemoveMovie(id)
      loadMovies()
    } catch (error) {
      console.error('Failed to remove movie:', error)
    }
  }

  const saveTemplate = async () => {
    try {
      await SpoilerService.SetTemplate(template)
      setEditingTemplate(false)
    } catch (error) {
      console.error('Failed to save template:', error)
    }
  }

  const updateSettings = async (newSettings: Partial<AppSettings>) => {
    const updated = { ...settings, ...newSettings }
    setSettings(updated)
    try {
      await SpoilerService.UpdateSettings(updated)
    } catch (error) {
      console.error('Failed to update settings:', error)
    }
  }

  const copyMovieResult = async (movieId: number) => {
    try {
      const result = await SpoilerService.GenerateResultForMovie(movieId)
      await navigator.clipboard.writeText(result)
    } catch (error) {
      console.error('Failed to copy result:', error)
    }
  }

  const copyAllResults = async () => {
    try {
      const result = await SpoilerService.GenerateResult()
      await navigator.clipboard.writeText(result)
    } catch (error) {
      console.error('Failed to copy results:', error)
    }
  }

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

  const getProcessingBadge = (state: string) => {
    switch (state) {
      case 'pending':
        return <Badge variant="outline" className="border-yellow-400/50 text-yellow-400">Pending</Badge>
      case 'processing':
        return <Badge variant="outline" className="border-blue-400/50 text-blue-400">Processing</Badge>
      case 'completed':
        return <Badge variant="outline" className="border-green-400/50 text-green-400">Complete</Badge>
      case 'error':
        return <Badge variant="outline" className="border-red-400/50 text-red-400">Error</Badge>
      default:
        return null
    }
  }

  const completedMovies = movies.filter(m => m.processingState === 'completed')
  const hasFiles = movies.length > 0

  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <div className="h-screen w-full flex flex-col overflow-hidden
        backdrop-blur-xl shadow-lg 
        bg-[radial-gradient(circle_at_center,rgba(0,0,0,0.6),rgba(0,0,0,0.9))]">
        
        <div className="relative z-10 container mx-auto p-6 max-w-7xl">
          <div className="backdrop-blur-xl bg-white/2 border border-white/5 rounded-3xl p-6 shadow-2xl">
            
            {/* Header */}
            <div className="flex items-center justify-between mb-6">
              <h1 className="text-3xl font-bold text-white">Spoiler List Generator</h1>
              <div className="flex items-center gap-4">
                {/* Template Editor Toggle */}
                {!editingTemplate ? (
                  <Button 
                    onClick={() => setEditingTemplate(true)}
                    variant="outline" 
                    className="border-white/20"
                  >
                    <Edit className="w-4 h-4 mr-2" />
                    Edit Template
                  </Button>
                ) : (
                  <div className="flex gap-2">
                    <Button onClick={saveTemplate} className="bg-gradient-to-r from-blue-600 to-purple-600">
                      Save Template
                    </Button>
                    <Button onClick={() => setEditingTemplate(false)} variant="outline">
                      Cancel
                    </Button>
                  </div>
                )}

                {/* Settings Popover */}
                <Popover>
                  <PopoverTrigger asChild>
                    <Button variant="outline" className="border-white/20">
                      <Settings className="w-4 h-4 mr-2" />
                      Settings
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-80 bg-black/90 border-white/10">
                    <div className="space-y-4">
                      <h4 className="font-medium text-white">Application Settings</h4>
                      
                      {/* Fastpic Settings */}
                      <div className="space-y-2">
                        <Label htmlFor="fastpicSid" className="text-slate-300">Fastpic SID</Label>
                        <Input 
                          id="fastpicSid"
                          value={settings.fastpicSid}
                          onChange={(e) => updateSettings({ fastpicSid: e.target.value })}
                          placeholder="fp_sid cookie value"
                          className="bg-black/40 border-white/10 text-white"
                        />
                      </div>

                      <Separator className="bg-white/5" />

                      {/* Screenshot Settings */}
                      <div className="space-y-3">
                        <div>
                          <Label className="text-slate-300">Screenshots: {settings.screenshotCount}</Label>
                          <Slider
                            value={[settings.screenshotCount]}
                            onValueChange={([value]) => updateSettings({ screenshotCount: value })}
                            max={12}
                            min={1}
                            step={1}
                            className="mt-2"
                          />
                        </div>
                        <div>
                          <Label className="text-slate-300">Quality: {settings.screenshotQuality}</Label>
                          <Slider
                            value={[settings.screenshotQuality]}
                            onValueChange={([value]) => updateSettings({ screenshotQuality: value })}
                            max={5}
                            min={1}
                            step={1}
                            className="mt-2"
                          />
                        </div>
                      </div>

                      <Separator className="bg-white/5" />

                      {/* Display Options */}
                      <div className="space-y-3">
                        <div className="flex items-center space-x-2">
                          <Checkbox 
                            id="centerAlign" 
                            checked={settings.centerAlign}
                            onCheckedChange={(checked) => updateSettings({ centerAlign: checked as boolean })}
                          />
                          <Label htmlFor="centerAlign" className="text-slate-300">Center alignment</Label>
                        </div>
                        <div className="flex items-center space-x-2">
                          <Checkbox 
                            id="hideEmpty" 
                            checked={settings.hideEmpty}
                            onCheckedChange={(checked) => updateSettings({ hideEmpty: checked as boolean })}
                          />
                          <Label htmlFor="hideEmpty" className="text-slate-300">Hide empty parameters</Label>
                        </div>
                      </div>
                    </div>
                  </PopoverContent>
                </Popover>

                {/* Copy All Button */}
                {completedMovies.length > 0 && (
                  <Button 
                    onClick={copyAllResults}
                    className="bg-gradient-to-r from-green-600 to-emerald-600"
                  >
                    <Copy className="w-4 h-4 mr-2" />
                    Copy All ({completedMovies.length})
                  </Button>
                )}
              </div>
            </div>

            {/* Template Editor */}
            {editingTemplate && (
              <Card className="bg-black/10 border-white/5 mb-6">
                <CardHeader>
                  <CardTitle className="text-white">Template Editor</CardTitle>
                </CardHeader>
                <CardContent>
                  <Textarea 
                    value={template}
                    onChange={(e) => setTemplate(e.target.value)}
                    className="min-h-[200px] font-mono text-sm bg-black/40 border-white/10 text-white"
                  />
                  <div className="mt-3 flex flex-wrap gap-2">
                    {[
                      '%FILE_NAME%', '%FILE_SIZE%', '%DURATION%', '%WIDTH%', '%HEIGHT%',
                      '%BIT_RATE%', '%VIDEO_BIT_RATE%', '%AUDIO_BIT_RATE%', 
                      '%VIDEO_CODEC%', '%AUDIO_CODEC%', '%SCREENSHOTS%'
                    ].map((param) => (
                      <Badge 
                        key={param} 
                        variant="outline" 
                        className="border-purple-400/30 text-purple-400 cursor-pointer hover:bg-purple-400/5"
                        onClick={() => setTemplate(prev => prev + param)}
                      >
                        {param}
                      </Badge>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Drop Zone - only show if no files */}
            {!hasFiles && (
              <Card className="bg-black/10 border-white/5 border-dashed border-2 hover:border-purple-400/30 transition-all duration-300 mb-6">
                <CardContent className="p-12">
                  <div 
                    className="text-center"
                    onDrop={handleDrop}
                    onDragOver={handleDragOver}
                  >
                    <div className="mx-auto mb-6 p-4 bg-gradient-to-br from-purple-600/20 to-blue-600/20 rounded-full w-fit">
                      <Upload className="w-12 h-12 text-purple-400" />
                    </div>
                    <h3 className="text-2xl font-semibold mb-3 text-white">Drop Video Files Here</h3>
                    <p className="text-slate-400 mb-6">Supports MP4, AVI, MKV, MOV, and other video formats</p>
                    <div className="flex gap-4 justify-center">
                      <Button onClick={handleFileSelect} className="bg-gradient-to-r from-purple-600 to-blue-600">
                        <Upload className="w-4 h-4 mr-2" />
                        Select Files
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Files Table - only show if files exist */}
            {hasFiles && (
              <Card className="bg-black/10 border-white/5">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-white flex items-center gap-2">
                      <FileVideo2Icon className="w-5 h-5" />
                      Files ({movies.length})
                    </CardTitle>
                    <Button onClick={clearMovies} variant="outline" className="border-white/20 hover:bg-red-500/20">
                      <Trash2 className="w-4 h-4 mr-2" />
                      Clear All
                    </Button>
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
                          <HoverCard key={movie.id}>
                            <HoverCardTrigger asChild>
                              <TableRow 
                                className={`border-white/5 hover:bg-white/2 cursor-pointer ${
                                  selectedMovie?.id === movie.id ? "bg-purple-500/10" : ""
                                }`}
                                onClick={() => setSelectedMovie(movie)}
                                onMouseEnter={() => movie.processingState === 'completed' && handleRowHover(movie.id)}
                              >
                                <TableCell className="text-slate-300">{index + 1}</TableCell>
                                <TableCell className="font-medium text-white">{movie.fileName}</TableCell>
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
                                  {getProcessingBadge(movie.processingState)}
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
                                          copyMovieResult(movie.id)
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
                                        removeMovie(movie.id)
                                      }}
                                    >
                                      <Trash2 className="w-4 h-4" />
                                    </Button>
                                  </div>
                                </TableCell>
                              </TableRow>
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
                        ))}
                      </TableBody>
                    </Table>
                  </ScrollArea>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </ThemeProvider>
  )
}

export default App