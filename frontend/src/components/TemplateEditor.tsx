import { Textarea } from "@/components/ui/textarea"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

interface TemplateEditorProps {
  template: string
  onTemplateChange: (template: string) => void
}

export default function TemplateEditor({ template, onTemplateChange }: TemplateEditorProps) {
  const templateParams = [
    '%FILE_NAME%', '%FILE_SIZE%', '%DURATION%', '%WIDTH%', '%HEIGHT%',
    '%BIT_RATE%', '%VIDEO_BIT_RATE%', '%AUDIO_BIT_RATE%', 
    '%VIDEO_CODEC%', '%AUDIO_CODEC%', '%SCREENSHOTS%'
  ]

  return (
    <Card className="bg-black/10 border-white/5 mb-6">
      <CardHeader>
        <CardTitle className="text-white">Template Editor</CardTitle>
      </CardHeader>
      <CardContent>
        <Textarea 
          value={template}
          onChange={(e) => onTemplateChange(e.target.value)}
          className="min-h-[200px] font-mono text-sm bg-black/40 border-white/10 text-white"
        />
        <div className="mt-3 flex flex-wrap gap-2">
          {templateParams.map((param) => (
            <Badge 
              key={param} 
              variant="outline" 
              className="border-purple-400/30 text-purple-400 cursor-pointer hover:bg-purple-400/5"
              onClick={() => onTemplateChange(template + param)}
            >
              {param}
            </Badge>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}