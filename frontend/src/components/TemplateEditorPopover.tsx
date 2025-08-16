import { useState, useEffect } from "react";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { RotateCcw } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useTranslation } from "@/contexts/LanguageContext";
import AnimatedText from "@/components/AnimatedText";

interface TemplateEditorProps {
  template: string;
  onTemplateChange: (template: string) => void;
  onResetToDefault: () => void;
}

interface TemplateParam {
  name: string;
  description: string;
  category?: string;
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
    // File Information
    { name: "%FILE_NAME%", description: t("templateEditor.parameters.fileName"), category: "File Info" },
    { name: "%FILE_SIZE%", description: t("templateEditor.parameters.fileSize"), category: "File Info" },
    { name: "%DURATION%", description: t("templateEditor.parameters.duration"), category: "File Info" },

    // Video Information
    { name: "%WIDTH%", description: t("templateEditor.parameters.width"), category: "Video" },
    { name: "%HEIGHT%", description: t("templateEditor.parameters.height"), category: "Video" },
    { name: "%BIT_RATE%", description: t("templateEditor.parameters.bitRate"), category: "Video" },
    { name: "%VIDEO_BIT_RATE%", description: t("templateEditor.parameters.videoBitRate"), category: "Video" },
    { name: "%VIDEO_CODEC%", description: t("templateEditor.parameters.videoCodec"), category: "Video" },
    { name: "%VIDEO_FPS%", description: t("templateEditor.parameters.videoFps"), category: "Video" },
    { name: "%VIDEO_FPS_FRACTIONAL%", description: t("templateEditor.parameters.videoFpsFractional"), category: "Video" },

    // Audio Information
    { name: "%AUDIO_BIT_RATE%", description: t("templateEditor.parameters.audioBitRate"), category: "Audio" },
    { name: "%AUDIO_CODEC%", description: t("templateEditor.parameters.audioCodec"), category: "Audio" },
    { name: "%AUDIO_SAMPLE_RATE%", description: t("templateEditor.parameters.audioSampleRate"), category: "Audio" },
    { name: "%AUDIO_CHANNELS%", description: t("templateEditor.parameters.audioChannels"), category: "Audio" },

    // Fastpic Images
    { name: "%THUMBNAIL_FP%", description: "Fastpic thumbnail (BBCode)", category: "Fastpic" },
    { name: "%THUMBNAIL_FP_BIG%", description: "Fastpic thumbnail big (BBCode)", category: "Fastpic" },
    { name: "%SCREENSHOTS_FP%", description: "Fastpic screenshots (newline separated)", category: "Fastpic" },
    { name: "%SCREENSHOTS_FP_SPACED%", description: "Fastpic screenshots (space separated)", category: "Fastpic" },
    { name: "%SCREENSHOTS_FP_BIG%", description: "Fastpic screenshots big (newline separated)", category: "Fastpic" },
    { name: "%SCREENSHOTS_FP_BIG_SPACED%", description: "Fastpic screenshots big (space separated)", category: "Fastpic" },

    // Imgbox Images
    { name: "%THUMBNAIL_IB%", description: "Imgbox thumbnail (BBCode)", category: "Imgbox" },
    { name: "%THUMBNAIL_IB_BIG%", description: "Imgbox thumbnail big (BBCode)", category: "Imgbox" },
    { name: "%SCREENSHOTS_IB%", description: "Imgbox screenshots (newline separated)", category: "Imgbox" },
    { name: "%SCREENSHOTS_IB_SPACED%", description: "Imgbox screenshots (space separated)", category: "Imgbox" },
    { name: "%SCREENSHOTS_IB_BIG%", description: "Imgbox screenshots big (newline separated)", category: "Imgbox" },
    { name: "%SCREENSHOTS_IB_BIG_SPACED%", description: "Imgbox screenshots big (space separated)", category: "Imgbox" },
  ];

  // Group parameters by category
  const groupedParams = templateParams.reduce((acc, param) => {
    const category = param.category || "Other";
    if (!acc[category]) {
      acc[category] = [];
    }
    acc[category].push(param);
    return acc;
  }, {} as Record<string, TemplateParam[]>);

  // Define category order
  const categoryOrder = ["File Info", "Video", "Audio", "Fastpic", "Imgbox", "Legacy", "Other"];

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

  const getCategoryColor = (category: string) => {
    switch (category) {
      case "File Info":
        return "bg-blue-100 text-blue-800 hover:bg-blue-200";
      case "Video":
        return "bg-green-100 text-green-800 hover:bg-green-200";
      case "Audio":
        return "bg-purple-100 text-purple-800 hover:bg-purple-200";
      case "Fastpic":
        return "bg-orange-100 text-orange-800 hover:bg-orange-200";
      case "Imgbox":
        return "bg-cyan-100 text-cyan-800 hover:bg-cyan-200";
      case "Legacy":
        return "bg-gray-100 text-gray-600 hover:bg-gray-200";
      default:
        return "bg-gray-100 text-gray-800 hover:bg-gray-200";
    }
  };

  return (
    <Popover open={isOpen} onOpenChange={handleOpenChange}>
      <PopoverTrigger className="cursor-pointer inline-flex items-center justify-center rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50">
        <AnimatedText>{t("header.editTemplate")}</AnimatedText>
      </PopoverTrigger>
      <PopoverContent className="w-[700px]" side="bottom" align="end">
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

          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">{t("templateEditor.availableParameters")}</p>

            <div className="space-y-3 max-h-[300px] overflow-y-auto">
              {categoryOrder.map((category) => {
                const params = groupedParams[category];
                if (!params || params.length === 0) return null;

                return (
                  <div key={category} className="space-y-2">
                    <h5 className="text-sm font-medium text-muted-foreground border-b pb-1">{category}</h5>
                    <div className="flex flex-wrap gap-2">
                      {params.map((param) => (
                        <Tooltip key={param.name}>
                          <TooltipTrigger asChild>
                            <span>
                              <Badge
                                variant="secondary"
                                className={`cursor-pointer ${getCategoryColor(category)} transition-colors`}
                                onClick={() => handleParamClick(param.name)}
                              >
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
                );
              })}
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
