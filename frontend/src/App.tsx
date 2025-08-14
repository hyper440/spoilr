import { ThemeProvider } from "@/components/theme-provider"

import { useState, useEffect, useCallback } from 'react'
import { SpoilerService, Movie, AppSettings } from "@bindings/changeme/backend"
import { Events, WML } from "@wailsio/runtime"

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Progress } from "@/components/ui/progress"
import { Textarea } from "@/components/ui/textarea"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Trash2, Upload, Copy, Settings, Filter, Link, FileText } from "lucide-react"

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
    language: 'English',
    convertToGb: true,
    centerAlign: false,
    acceptOnlyLinks: false,
    hideEmpty: true,
    uiFontSize: 12,
    listFontSize: 10,
    textFontSize: 12
  })
  const [template, setTemplate] = useState('')
  const [result, setResult] = useState('')
  const [progress, setProgress] = useState<ProcessProgress | null>(null)
  
  // Unbender state
  const [originalText, setOriginalText] = useState('')
  const [bendLinks, setBendLinks] = useState('')
  const [unbendLinks, setUnbendLinks] = useState('')
  const [replaceLinks, setReplaceLinks] = useState(false)
  
  // Link editor state
  const [originalText2, setOriginalText2] = useState('')
  const [allLinks, setAllLinks] = useState('')
  const [filterLinks, setFilterLinks] = useState('')
  const [allowFilter, setAllowFilter] = useState('Allow')
  const [blockFilter, setBlockFilter] = useState('Block')
  
  const [selectedMovie, setSelectedMovie] = useState<Movie | null>(null)
  const [activeTab, setActiveTab] = useState('general')

  // Load initial data and set up event listeners
  useEffect(() => {
    loadMovies()
    loadSettings()
    loadTemplate()
    
    // Listen for progress events from backend
    Events.On('progress', (ev: any) => {
      const data = ev.data as ProcessProgress
      setProgress(data[0]) // wails wraps event data in array
      if (data.completed) {
        loadMovies()
      }
    })
    
    // Reload WML so it picks up any wml tags
    WML.Reload()
    
    // Cleanup event listeners on unmount
    return () => {
      Events.Off('progress')
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
    try {
      // In a real Wails app, you'd use the file dialog API
      // For now, this is a placeholder that demonstrates the structure
      // const files = await WailsAPI.OpenFileDialog({
      //   filters: [
      //     { displayName: 'Video Files', pattern: '*.mp4;*.avi;*.mkv;*.mov;*.wmv;*.flv;*.webm;*.m4v' }
      //   ],
      //   allowMultiple: true
      // })
      
      console.log('File dialog would open here')
      // const expandedPaths = await SpoilerService.GetExpandedFilePaths(files)
      // await SpoilerService.ProcessFilesAsync(expandedPaths)
    } catch (error) {
      console.error('File selection failed:', error)
    }
  }

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    // File drop is now handled by the Wails backend through WindowFilesDropped event
    // The backend will automatically process dropped files and emit progress events
  }, [])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
  }, [])

  const clearMovies = async () => {
    try {
      await SpoilerService.ClearMovies()
      setMovies([])
      setResult('')
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

  const pasteURLs = async () => {
    try {
      const text = await navigator.clipboard.readText()
      const newMovies = await SpoilerService.ProcessURLs(text, settings.acceptOnlyLinks)
      setMovies(prev => [...prev, ...newMovies])
    } catch (error) {
      console.error('Failed to paste URLs:', error)
    }
  }

  const generateResult = async () => {
    try {
      const generated = await SpoilerService.GenerateResult()
      setResult(generated)
    } catch (error) {
      console.error('Failed to generate result:', error)
    }
  }

  const saveTemplate = async () => {
    try {
      await SpoilerService.SetTemplate(template)
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

  const processUnbendLinks = async () => {
    try {
      const result = await SpoilerService.ProcessLinks(originalText, replaceLinks)
      setBendLinks(result.originalLinks.join('\n'))
      setUnbendLinks(result.processedLinks.join('\n'))
      if (replaceLinks) {
        // Update original text with processed links
        let updatedText = originalText
        result.originalLinks.forEach((original, index) => {
          if (result.processedLinks[index] && !result.processedLinks[index].startsWith('Error:')) {
            updatedText = updatedText.replace(original, result.processedLinks[index])
          }
        })
        setOriginalText(updatedText)
      }
    } catch (error) {
      console.error('Failed to process links:', error)
    }
  }

  const filterLinksFunction = async () => {
    try {
      const allowFilters = allowFilter.split(' ').filter(f => f.trim() !== '')
      const blockFilters = blockFilter.split(' ').filter(f => f.trim() !== '')
      const [all, filtered] = await SpoilerService.FilterLinks(originalText2, allowFilters, blockFilters)
      setAllLinks(all.join('\n'))
      setFilterLinks(filtered.join('\n'))
    } catch (error) {
      console.error('Failed to filter links:', error)
    }
  }

  const clearUnbender = () => {
    setOriginalText('')
    setBendLinks('')
    setUnbendLinks('')
  }

  const clearLinkEditor = () => {
    setOriginalText2('')
    setAllLinks('')
    setFilterLinks('')
  }

  // Auto-generate result when switching to result tab
  useEffect(() => {
    if (activeTab === 'result') {
      generateResult()
    }
  }, [activeTab, movies, template, settings])

  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
    <div className="container mx-auto p-4 max-w-7xl">
      <div className="mb-6">
        <h1 className="text-3xl font-bold text-center mb-2">Spoiler List Generator</h1>
        <p className="text-center text-muted-foreground">Media file analyzer for generating torrent spoiler lists</p>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="general" className="flex items-center gap-2">
            <Upload className="w-4 h-4" />
            General
          </TabsTrigger>
          <TabsTrigger value="result" className="flex items-center gap-2">
            <FileText className="w-4 h-4" />
            Result
          </TabsTrigger>
          <TabsTrigger value="template" className="flex items-center gap-2">
            <Settings className="w-4 h-4" />
            Template
          </TabsTrigger>
          <TabsTrigger value="unbender" className="flex items-center gap-2">
            <Link className="w-4 h-4" />
            Unbender
          </TabsTrigger>
          <TabsTrigger value="link-editor" className="flex items-center gap-2">
            <Filter className="w-4 h-4" />
            Link Editor
          </TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <span>Files</span>
                <div className="flex gap-2">
                  <Button onClick={handleFileSelect} variant="outline" size="sm">
                    <Upload className="w-4 h-4 mr-2" />
                    Select Files
                  </Button>
                  <Button onClick={pasteURLs} variant="outline" size="sm">
                    <Copy className="w-4 h-4 mr-2" />
                    Paste URLs
                  </Button>
                  <Button onClick={clearMovies} variant="outline" size="sm">
                    <Trash2 className="w-4 h-4 mr-2" />
                    Clear
                  </Button>
                </div>
              </CardTitle>
            </CardHeader>
            <CardContent>
              {/* Settings Panel */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                <div className="space-y-2">
                  <Label>Language</Label>
                  <Select value={settings.language} onValueChange={(value) => updateSettings({ language: value })}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="English">English</SelectItem>
                      <SelectItem value="Русский">Русский</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                
                <div className="space-y-3">
                  <div className="flex items-center space-x-2">
                    <Checkbox 
                      id="convertToGb" 
                      checked={settings.convertToGb}
                      onCheckedChange={(checked) => updateSettings({ convertToGb: checked as boolean })}
                    />
                    <Label htmlFor="convertToGb">Convert to GB</Label>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Checkbox 
                      id="centerAlign" 
                      checked={settings.centerAlign}
                      onCheckedChange={(checked) => updateSettings({ centerAlign: checked as boolean })}
                    />
                    <Label htmlFor="centerAlign">Center alignment</Label>
                  </div>
                </div>
                
                <div className="space-y-3">
                  <div className="flex items-center space-x-2">
                    <Checkbox 
                      id="acceptOnlyLinks" 
                      checked={settings.acceptOnlyLinks}
                      onCheckedChange={(checked) => updateSettings({ acceptOnlyLinks: checked as boolean })}
                    />
                    <Label htmlFor="acceptOnlyLinks">Accept only links</Label>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Checkbox 
                      id="hideEmpty" 
                      checked={settings.hideEmpty}
                      onCheckedChange={(checked) => updateSettings({ hideEmpty: checked as boolean })}
                    />
                    <Label htmlFor="hideEmpty">Hide empty params</Label>
                  </div>
                </div>
              </div>
              
              <Separator className="my-4" />

              {/* Drop Zone */}
              <div 
                className="border-2 border-dashed border-muted-foreground/25 rounded-lg p-8 text-center transition-colors hover:border-muted-foreground/50"
                onDrop={handleDrop}
                onDragOver={handleDragOver}
              >
                <Upload className="w-12 h-12 mx-auto mb-4 text-muted-foreground" />
                <p className="text-lg font-medium mb-2">Drop video files or folders here</p>
                <p className="text-sm text-muted-foreground">Supports MP4, AVI, MKV, MOV, and other video formats</p>
                <p className="text-xs text-green-600 mt-2">
                  ✓ Drag & drop is fully supported through Wails backend
                </p>
              </div>

              {/* Progress */}
              {progress && (
                <div className="space-y-2">
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium">{progress.message}</span>
                    <span className="text-sm text-muted-foreground">
                      {progress.current}/{progress.total}
                    </span>
                  </div>
                  <Progress value={(progress.current / progress.total) * 100} />
                  {progress.error && (
                    <p className="text-sm text-destructive">{progress.error}</p>
                  )}
                </div>
              )}

              {/* Movies Table */}
              <ScrollArea className="h-[400px] mt-4">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-12">#</TableHead>
                      <TableHead>File Name</TableHead>
                      <TableHead>Size</TableHead>
                      <TableHead>Duration</TableHead>
                      <TableHead>Width</TableHead>
                      <TableHead>Height</TableHead>
                      <TableHead>Screen URL</TableHead>
                      <TableHead className="w-12"></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {movies.map((movie, index) => (
                      <TableRow 
                        key={movie.id}
                        className={selectedMovie?.id === movie.id ? "bg-muted" : ""}
                        onClick={() => setSelectedMovie(movie)}
                      >
                        <TableCell>{index + 1}</TableCell>
                        <TableCell className="font-medium">{movie.fileName}</TableCell>
                        <TableCell>{movie.fileSize}</TableCell>
                        <TableCell>{movie.duration}</TableCell>
                        <TableCell>{movie.width}</TableCell>
                        <TableCell>{movie.height}</TableCell>
                        <TableCell className="max-w-xs truncate">{movie.screenListURL}</TableCell>
                        <TableCell>
                          <Button 
                            size="sm" 
                            variant="ghost"
                            onClick={(e) => {
                              e.stopPropagation()
                              removeMovie(movie.id)
                            }}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </ScrollArea>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="result" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Generated Spoiler List</CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea 
                value={result}
                onChange={(e) => setResult(e.target.value)}
                className="min-h-[500px] font-mono text-sm"
                placeholder="Generated spoiler list will appear here..."
              />
              <div className="flex justify-end mt-2">
                <Button 
                  onClick={() => navigator.clipboard.writeText(result)}
                  disabled={!result}
                  size="sm"
                >
                  <Copy className="w-4 h-4 mr-2" />
                  Copy to Clipboard
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="template" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <Card>
              <CardHeader>
                <CardTitle>Template Editor</CardTitle>
              </CardHeader>
              <CardContent>
                <Textarea 
                  value={template}
                  onChange={(e) => setTemplate(e.target.value)}
                  className="min-h-[300px] font-mono text-sm"
                  placeholder="Enter your spoiler template here..."
                />
                <div className="flex justify-end mt-2">
                  <Button onClick={saveTemplate} size="sm">
                    Save Template
                  </Button>
                </div>
              </CardContent>
            </Card>
            
            <Card>
              <CardHeader>
                <CardTitle>Available Parameters</CardTitle>
                {selectedMovie && (
                  <p className="text-sm text-muted-foreground">
                    Parameters for: {selectedMovie.fileName}
                  </p>
                )}
              </CardHeader>
              <CardContent>
                <ScrollArea className="h-[350px]">
                  {selectedMovie ? (
                    <div className="space-y-2">
                      <div className="space-y-1">
                        <Badge variant="outline">%FILE_NAME%</Badge>
                        <Badge variant="outline">%FILE_SIZE%</Badge>
                        <Badge variant="outline">%DURATION%</Badge>
                        <Badge variant="outline">%WIDTH%</Badge>
                        <Badge variant="outline">%HEIGHT%</Badge>
                        <Badge variant="outline">%IMG%</Badge>
                      </div>
                      <Separator />
                      {Object.entries(selectedMovie.params)
                        .filter(([, value]) => !settings.hideEmpty || value)
                        .map(([param, value]) => (
                          <div key={param} className="p-2 border rounded">
                            <div className="font-mono text-xs text-blue-600 mb-1">{param}</div>
                            <div className="text-sm">{value || '(empty)'}</div>
                          </div>
                        ))}
                    </div>
                  ) : (
                    <p className="text-muted-foreground">Select a movie to view available parameters</p>
                  )}
                </ScrollArea>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="unbender" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <span>Link Unbender</span>
                <div className="flex gap-2">
                  <Button onClick={processUnbendLinks} size="sm">
                    Process Links
                  </Button>
                  <Button onClick={clearUnbender} variant="outline" size="sm">
                    Clear
                  </Button>
                </div>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center space-x-2 mb-4">
                <Checkbox 
                  id="replaceLinks" 
                  checked={replaceLinks}
                  onCheckedChange={(checked) => setReplaceLinks(checked as boolean)}
                />
                <Label htmlFor="replaceLinks">Replace bended links in original</Label>
              </div>
              
              <Tabs defaultValue="original" className="w-full">
                <TabsList className="grid w-full grid-cols-3">
                  <TabsTrigger value="original">Original</TabsTrigger>
                  <TabsTrigger value="bend">Bend Links</TabsTrigger>
                  <TabsTrigger value="unbend">Unbend Links</TabsTrigger>
                </TabsList>
                <TabsContent value="original">
                  <Textarea 
                    value={originalText}
                    onChange={(e) => setOriginalText(e.target.value)}
                    className="min-h-[400px] font-mono text-sm"
                    placeholder="Paste text with links to process..."
                  />
                </TabsContent>
                <TabsContent value="bend">
                  <Textarea 
                    value={bendLinks}
                    readOnly
                    className="min-h-[400px] font-mono text-sm"
                    placeholder="Original links will appear here..."
                  />
                </TabsContent>
                <TabsContent value="unbend">
                  <Textarea 
                    value={unbendLinks}
                    readOnly
                    className="min-h-[400px] font-mono text-sm"
                    placeholder="Processed links will appear here..."
                  />
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="link-editor" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Link Filter</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                <div className="space-y-2">
                  <Label htmlFor="allowFilter">Allow Filter</Label>
                  <Input 
                    id="allowFilter"
                    value={allowFilter}
                    onChange={(e) => setAllowFilter(e.target.value)}
                    placeholder="Words that must be present"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="blockFilter">Block Filter</Label>
                  <Input 
                    id="blockFilter"
                    value={blockFilter}
                    onChange={(e) => setBlockFilter(e.target.value)}
                    placeholder="Words that block the link"
                  />
                </div>
                <div className="flex items-end space-x-2">
                  <Button onClick={filterLinksFunction}>
                    <Filter className="w-4 h-4 mr-2" />
                    Filter
                  </Button>
                  <Button onClick={clearLinkEditor} variant="outline">
                    Clear
                  </Button>
                </div>
              </div>
              
              <Tabs defaultValue="original2" className="w-full">
                <TabsList className="grid w-full grid-cols-3">
                  <TabsTrigger value="original2">Original</TabsTrigger>
                  <TabsTrigger value="all">All Links</TabsTrigger>
                  <TabsTrigger value="filtered">Filtered Links</TabsTrigger>
                </TabsList>
                <TabsContent value="original2">
                  <Textarea 
                    value={originalText2}
                    onChange={(e) => setOriginalText2(e.target.value)}
                    className="min-h-[400px] font-mono text-sm"
                    placeholder="Paste text with links to filter..."
                  />
                </TabsContent>
                <TabsContent value="all">
                  <Textarea 
                    value={allLinks}
                    readOnly
                    className="min-h-[400px] font-mono text-sm"
                    placeholder="All extracted links will appear here..."
                  />
                </TabsContent>
                <TabsContent value="filtered">
                  <Textarea 
                    value={filterLinks}
                    readOnly
                    className="min-h-[400px] font-mono text-sm"
                    placeholder="Filtered links will appear here..."
                  />
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
    </ThemeProvider>
  )
}

export default App