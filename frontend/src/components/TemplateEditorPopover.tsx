import { useState, useEffect } from "react";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import { RotateCcw, Save, X, Plus } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useTranslation } from "@/contexts/LanguageContext";
import { SpoilerService } from "@bindings/spoilr/backend";
import AnimatedText from "@/components/AnimatedText";

interface TemplateEditorProps {
  onResetTemplate: () => void;
}

interface TemplateParam {
  name: string;
  description: string;
  category?: string;
}

interface TemplatePreset {
  id: string;
  name: string;
  template: string;
}

export default function TemplateEditor({ onResetTemplate }: TemplateEditorProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [currentTemplate, setCurrentTemplate] = useState("");
  const [cursorPosition, setCursorPosition] = useState(0);
  const [presets, setPresets] = useState<TemplatePreset[]>([]);
  const [currentPresetId, setCurrentPresetId] = useState<string>("");
  const [newPresetName, setNewPresetName] = useState("");
  const [isSavingPreset, setIsSavingPreset] = useState(false);
  const [showNewPreset, setShowNewPreset] = useState(false);

  useEffect(() => {
    if (isOpen) {
      loadPresetsAndCurrentTemplate();
    }
  }, [isOpen]);

  const loadPresetsAndCurrentTemplate = async () => {
    try {
      const [loadedPresets, currentPreset, template] = await Promise.all([
        SpoilerService.GetTemplatePresets(),
        SpoilerService.GetCurrentPresetID(),
        SpoilerService.GetTemplate(),
      ]);
      setPresets(loadedPresets);
      setCurrentPresetId(currentPreset);
      setCurrentTemplate(template);
    } catch (error) {
      console.error("Failed to load template data:", error);
    }
  };

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

    // Contact Sheets (MTN-generated grids)
    { name: "%CONTACT_SHEET_FP%", description: t("templateEditor.parameters.contactSheetFp"), category: "Contact Sheets" },
    { name: "%CONTACT_SHEET_FP_BIG%", description: t("templateEditor.parameters.contactSheetFpBig"), category: "Contact Sheets" },
    { name: "%CONTACT_SHEET_IB%", description: t("templateEditor.parameters.contactSheetIb"), category: "Contact Sheets" },
    { name: "%CONTACT_SHEET_IB_BIG%", description: t("templateEditor.parameters.contactSheetIbBig"), category: "Contact Sheets" },
    { name: "%CONTACT_SHEET_HAM%", description: t("templateEditor.parameters.contactSheetHam"), category: "Contact Sheets" },
    { name: "%CONTACT_SHEET_HAM_BIG%", description: t("templateEditor.parameters.contactSheetHamBig"), category: "Contact Sheets" },

    // Fastpic Screenshots
    { name: "%SCREENSHOTS_FP%", description: t("templateEditor.parameters.screenshotsFp"), category: "Fastpic Screenshots" },
    { name: "%SCREENSHOTS_FP_SPACED%", description: t("templateEditor.parameters.screenshotsFpSpaced"), category: "Fastpic Screenshots" },
    { name: "%SCREENSHOTS_FP_BIG%", description: t("templateEditor.parameters.screenshotsFpBig"), category: "Fastpic Screenshots" },
    { name: "%SCREENSHOTS_FP_BIG_SPACED%", description: t("templateEditor.parameters.screenshotsFpBigSpaced"), category: "Fastpic Screenshots" },

    // Imgbox Screenshots
    { name: "%SCREENSHOTS_IB%", description: t("templateEditor.parameters.screenshotsIb"), category: "Imgbox Screenshots" },
    { name: "%SCREENSHOTS_IB_SPACED%", description: t("templateEditor.parameters.screenshotsIbSpaced"), category: "Imgbox Screenshots" },
    { name: "%SCREENSHOTS_IB_BIG%", description: t("templateEditor.parameters.screenshotsIbBig"), category: "Imgbox Screenshots" },
    { name: "%SCREENSHOTS_IB_BIG_SPACED%", description: t("templateEditor.parameters.screenshotsIbBigSpaced"), category: "Imgbox Screenshots" },

    // Hamster Screenshots
    { name: "%SCREENSHOTS_HAM%", description: t("templateEditor.parameters.screenshotsHam"), category: "Hamster Screenshots" },
    { name: "%SCREENSHOTS_HAM_SPACED%", description: t("templateEditor.parameters.screenshotsHamSpaced"), category: "Hamster Screenshots" },
    { name: "%SCREENSHOTS_HAM_BIG%", description: t("templateEditor.parameters.screenshotsHamBig"), category: "Hamster Screenshots" },
    { name: "%SCREENSHOTS_HAM_BIG_SPACED%", description: t("templateEditor.parameters.screenshotsHamBigSpaced"), category: "Hamster Screenshots" },
  ];

  const groupedParams = templateParams.reduce((acc, param) => {
    const category = param.category || "Other";
    if (!acc[category]) {
      acc[category] = [];
    }
    acc[category].push(param);
    return acc;
  }, {} as Record<string, TemplateParam[]>);

  const categoryOrder = ["File Info", "Video", "Audio", "Contact Sheets", "Fastpic Screenshots", "Imgbox Screenshots", "Hamster Screenshots"];

  const handleOpenChange = (open: boolean) => {
    setIsOpen(open);
    if (open) {
      setNewPresetName("");
      setShowNewPreset(false);
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

  const handleSaveTemplate = async () => {
    try {
      await SpoilerService.SetTemplate(currentTemplate);
      setIsOpen(false);
    } catch (error) {
      console.error("Failed to save template:", error);
    }
  };

  const handlePresetClick = async (preset: TemplatePreset) => {
    try {
      await SpoilerService.SetCurrentPreset(preset.id);
      setCurrentTemplate(preset.template);
      setCurrentPresetId(preset.id);
    } catch (error) {
      console.error("Failed to set current preset:", error);
    }
  };

  const handleSavePreset = async () => {
    if (!newPresetName.trim()) return;

    setIsSavingPreset(true);
    try {
      await SpoilerService.SaveTemplatePreset(newPresetName.trim(), currentTemplate);
      await loadPresetsAndCurrentTemplate(); // Reload data from backend
      setNewPresetName("");
      setShowNewPreset(false);
    } catch (error) {
      console.error("Failed to save template preset:", error);
    } finally {
      setIsSavingPreset(false);
    }
  };

  const handleDeletePreset = async (presetId: string) => {
    try {
      await SpoilerService.DeleteTemplatePreset(presetId);
      await loadPresetsAndCurrentTemplate(); // Reload data from backend
    } catch (error) {
      console.error("Failed to delete template preset:", error);
    }
  };

  return (
    <Popover open={isOpen} onOpenChange={handleOpenChange}>
      <PopoverTrigger className="cursor-pointer inline-flex items-center justify-center rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50">
        <AnimatedText>{t("templateEditor.editTemplate")}</AnimatedText>
      </PopoverTrigger>
      <PopoverContent className="w-[800px] p-4" side="bottom" align="end">
        <div className="space-y-4">
          {/* Header */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <h4 className="font-medium text-sm">{t("templateEditor.title")}</h4>
            </div>

            <div className="flex items-center gap-2">
              {/* Add Preset Toggle */}
              {!showNewPreset ? (
                <Button variant="outline" size="sm" onClick={() => setShowNewPreset(true)} className="h-8 px-2">
                  <Plus className="w-3 h-3 mr-1" />
                  {t("templateEditor.saveAsPreset")}
                </Button>
              ) : (
                <div className="flex gap-1">
                  <Input
                    placeholder={t("templateEditor.presetName")}
                    value={newPresetName}
                    onChange={(e) => setNewPresetName(e.target.value)}
                    className="h-6 text-xs w-45"
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        handleSavePreset();
                      } else if (e.key === "Escape") {
                        setShowNewPreset(false);
                        setNewPresetName("");
                      }
                    }}
                    autoFocus
                  />
                  <Button onClick={handleSavePreset} disabled={!newPresetName.trim() || isSavingPreset} size="sm" className="h-6 px-2 text-xs">
                    <Save className="w-3 h-3" />
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => {
                      setShowNewPreset(false);
                      setNewPresetName("");
                    }}
                    size="sm"
                    className="h-6 px-1 text-xs"
                  >
                    <X className="w-3 h-3" />
                  </Button>
                </div>
              )}
              <Button variant="outline" size="sm" onClick={onResetTemplate} className="h-8 px-2">
                <RotateCcw className="w-3 h-3 mr-1" />
                {t("templateEditor.resetToDefault")}
              </Button>
              <Button onClick={handleSaveTemplate} size="sm" className="h-8">
                <Save className="w-4 h-4" />
                {t("templateEditor.saveTemplate")}
              </Button>
            </div>
          </div>

          {/* Presets Row */}
          {presets.length > 0 && (
            <div className="flex items-center gap-2">
              <div className="flex flex-wrap gap-1 flex-1">
                {presets.map((preset) => (
                  <div key={preset.id} className="relative group">
                    <Badge
                      variant={preset.id === currentPresetId ? "default" : "outline"}
                      className="cursor-pointer text-xs px-2 py-1 h-6"
                      onClick={() => handlePresetClick(preset)}
                    >
                      {preset.name}
                    </Badge>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="absolute -top-2 -right-2 size-4 rounded-full p-0 opacity-0 group-hover:opacity-100 transition-opacity bg-secondary hover:bg-destructive text-secondary-foreground hover:text-destructive-foreground"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDeletePreset(preset.id);
                      }}
                    >
                      <X className="h-2 w-2" />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          )}

          <Separator />

          {/* Template Textarea */}
          <Textarea
            value={currentTemplate}
            onChange={handleTextareaChange}
            onSelect={handleTextareaSelect}
            onKeyUp={handleTextareaSelect}
            onClick={handleTextareaSelect}
            className="min-h-[100px] font-mono text-sm resize-none"
            placeholder={t("templateEditor.placeholder")}
          />

          {/* Parameters Tabs */}
          <div className="space-y-2">
            <div className="text-xs text-muted-foreground font-medium">{t("templateEditor.parametersLabel")}</div>
            <Tabs defaultValue={categoryOrder[0]} className="w-full">
              <TabsList className="flex w-full h-auto flex-wrap justify-start">
                {categoryOrder.map((category) => {
                  const params = groupedParams[category];
                  if (!params || params.length === 0) return null;

                  return (
                    <TabsTrigger key={category} value={category} className="text-xs py-2 px-3 h-auto whitespace-nowrap flex-shrink-0">
                      {category}
                    </TabsTrigger>
                  );
                })}
              </TabsList>
              {categoryOrder.map((category) => {
                const params = groupedParams[category];
                if (!params || params.length === 0) return null;

                return (
                  <TabsContent key={category} value={category} className="mt-3">
                    <div className="grid grid-cols-2 gap-2">
                      {params.map((param) => (
                        <Tooltip key={param.name}>
                          <TooltipTrigger asChild>
                            <button
                              onClick={() => handleParamClick(param.name)}
                              className="text-left p-2 rounded text-xs font-mono transition-colors border hover:bg-accent hover:text-accent-foreground"
                            >
                              {param.name}
                            </button>
                          </TooltipTrigger>
                          <TooltipContent side="top">
                            <p className="text-xs">{param.description}</p>
                          </TooltipContent>
                        </Tooltip>
                      ))}
                    </div>
                  </TabsContent>
                );
              })}
            </Tabs>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
