import { useCallback } from 'react'
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Upload } from "lucide-react"

export default function DropZone() {
  const handleFileSelect = async () => {
    console.log('File dialog would open here')
  }

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
  }, [])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
  }, [])

  return (
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
  )
}