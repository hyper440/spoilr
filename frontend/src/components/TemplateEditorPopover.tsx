import { useState } from "react";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Edit } from "lucide-react";

interface TemplateEditorProps {
  template: string;
  onTemplateChange: (template: string) => void;
}

export default function TemplateEditor({ template, onTemplateChange }: TemplateEditorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [currentTemplate, setCurrentTemplate] = useState(template);

  const templateParams = [
    "%FILE_NAME%",
    "%FILE_SIZE%",
    "%DURATION%",
    "%WIDTH%",
    "%HEIGHT%",
    "%BIT_RATE%",
    "%VIDEO_BIT_RATE%",
    "%AUDIO_BIT_RATE%",
    "%VIDEO_CODEC%",
    "%AUDIO_CODEC%",
    "%SCREENSHOTS%",
  ];

  const handleOpenChange = (open: boolean) => {
    if (!open && currentTemplate !== template) {
      // Auto-save when closing if there are changes
      onTemplateChange(currentTemplate);
    }
    setIsOpen(open);
    if (open) {
      // Reset to current template when opening
      setCurrentTemplate(template);
    }
  };

  const handleParamClick = (param: string) => {
    setCurrentTemplate(currentTemplate + param);
  };

  return (
    <Popover open={isOpen} onOpenChange={handleOpenChange}>
      <PopoverTrigger className="cursor-pointer inline-flex items-center justify-center rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-8 px-2">
        <Edit className="w-4 h-4 mr-2" />
        Edit Template
      </PopoverTrigger>
      <PopoverContent className="w-[600px]" side="bottom" align="end">
        <div className="space-y-4">
          <h4 className="font-medium text-base">Template Editor</h4>

          <Textarea
            value={currentTemplate}
            onChange={(e) => setCurrentTemplate(e.target.value)}
            className="min-h-[200px] font-mono text-sm"
            placeholder="Enter your template here..."
          />
          <div className="flex flex-wrap gap-2">
            {templateParams.map((param) => (
              <Badge key={param} variant="secondary" className="cursor-pointer hover:bg-secondary/80" onClick={() => handleParamClick(param)}>
                {param}
              </Badge>
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
