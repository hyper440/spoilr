import { useState, useEffect } from "react";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Edit, RotateCcw } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useTranslation } from "@/contexts/LanguageContext";

interface TemplateEditorProps {
  template: string;
  onTemplateChange: (template: string) => void;
  onResetToDefault: () => void;
}

interface TemplateParam {
  name: string;
  description: string;
}

export default function TemplateEditor({ template, onTemplateChange, onResetToDefault }: TemplateEditorProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [currentTemplate, setCurrentTemplate] = useState(template);
  const [cursorPosition, setCursorPosition] = useState(0);

  useEffect(() => {
    setCurrentTemplate(template);
  }, [template]);

  const templateParams: TemplateParam[] = [
    { name: "%FILE_NAME%", description: t("templateEditor.parameters.fileName") },
    { name: "%FILE_SIZE%", description: t("templateEditor.parameters.fileSize") },
    { name: "%DURATION%", description: t("templateEditor.parameters.duration") },
    { name: "%WIDTH%", description: t("templateEditor.parameters.width") },
    { name: "%HEIGHT%", description: t("templateEditor.parameters.height") },
    { name: "%BIT_RATE%", description: t("templateEditor.parameters.bitRate") },
    { name: "%VIDEO_BIT_RATE%", description: t("templateEditor.parameters.videoBitRate") },
    { name: "%AUDIO_BIT_RATE%", description: t("templateEditor.parameters.audioBitRate") },
    { name: "%VIDEO_CODEC%", description: t("templateEditor.parameters.videoCodec") },
    { name: "%AUDIO_CODEC%", description: t("templateEditor.parameters.audioCodec") },
    { name: "%VIDEO_FPS%", description: t("templateEditor.parameters.videoFps") },
    { name: "%VIDEO_FPS_FRACTIONAL%", description: t("templateEditor.parameters.videoFpsFractional") },
    { name: "%AUDIO_SAMPLE_RATE%", description: t("templateEditor.parameters.audioSampleRate") },
    { name: "%AUDIO_CHANNELS%", description: t("templateEditor.parameters.audioChannels") },
    { name: "%SCREENSHOTS%", description: t("templateEditor.parameters.screenshots") },
    { name: "%SCREENSHOTS_SPACED%", description: t("templateEditor.parameters.screenshotsSpaced") },
    { name: "%SCREENSHOTS_BIG%", description: t("templateEditor.parameters.screenshotsBig") },
    { name: "%SCREENSHOTS_BIG_SPACED%", description: t("templateEditor.parameters.screenshotsBigSpaced") },
    { name: "%THUMBNAIL%", description: t("templateEditor.parameters.thumbnail") },
    { name: "%THUMBNAIL_BIG%", description: t("templateEditor.parameters.thumbnailBig") },
  ];

  const handleOpenChange = (open: boolean) => {
    setIsOpen(open);
    if (open) {
      setCurrentTemplate(template);
    }
  };

  const handleParamClick = (param: string) => {
    const start = cursorPosition;
    const end = cursorPosition;
    const newValue = currentTemplate.substring(0, start) + param + currentTemplate.substring(end);
    setCurrentTemplate(newValue);
    setCursorPosition(start + param.length);
  };

  const handleTextareaChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setCurrentTemplate(e.target.value);
    setCursorPosition(e.target.selectionStart);
  };

  const handleTextareaSelect = (e: React.SyntheticEvent<HTMLTextAreaElement>) => {
    const target = e.target as HTMLTextAreaElement;
    setCursorPosition(target.selectionStart);
  };

  const handleResetToDefault = () => {
    onResetToDefault();
  };

  const handleSaveTemplate = () => {
    onTemplateChange(currentTemplate);
    setIsOpen(false);
  };

  return (
    <Popover open={isOpen} onOpenChange={handleOpenChange}>
      <PopoverTrigger className="cursor-pointer inline-flex items-center justify-center rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-8 px-2">
        <Edit className="w-4 h-4 mr-2" />
        {t("header.editTemplate")}
      </PopoverTrigger>
      <PopoverContent className="w-[600px]" side="bottom" align="end">
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h4 className="font-medium text-base">{t("templateEditor.title")}</h4>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={handleResetToDefault} className="text-xs">
                <RotateCcw className="w-3 h-3 mr-1" />
                {t("templateEditor.resetToDefault")}
              </Button>
              <Button onClick={handleSaveTemplate} size="sm">
                {t("templateEditor.saveTemplate")}
              </Button>
            </div>
          </div>

          <Textarea
            value={currentTemplate}
            onChange={handleTextareaChange}
            onSelect={handleTextareaSelect}
            onKeyUp={handleTextareaSelect}
            onClick={handleTextareaSelect}
            className="min-h-[200px] font-mono text-sm"
            placeholder={t("templateEditor.placeholder")}
          />

          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">{t("templateEditor.availableParameters")}</p>
            <div className="flex flex-wrap gap-2">
              {templateParams.map((param) => (
                <Tooltip key={param.name}>
                  <TooltipTrigger asChild>
                    <span>
                      <Badge variant="secondary" className="cursor-pointer hover:bg-secondary/80" onClick={() => handleParamClick(param.name)}>
                        {param.name}
                      </Badge>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>{param.description}</p>
                  </TooltipContent>
                </Tooltip>
              ))}
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
