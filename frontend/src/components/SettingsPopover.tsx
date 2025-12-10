import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Slider } from "@/components/ui/slider";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { AppSettings } from "@bindings/spoilr/backend";
import * as SpoilerService from "@bindings/spoilr/backend/spoilerservice";
import { useTranslation } from "@/contexts/LanguageContext";
import AnimatedText from "@/components/AnimatedText";
import { FolderOpen, X } from "lucide-react";

interface SettingsPopoverProps {
  settings: AppSettings;
  onUpdateSettings: (settings: Partial<AppSettings>) => void;
}

export default function SettingsPopover({ settings, onUpdateSettings }: SettingsPopoverProps) {
  const { t } = useTranslation();

  return (
    <Popover>
      <PopoverTrigger className="cursor-pointer inline-flex items-center justify-center">
        <AnimatedText>{t("header.settings")}</AnimatedText>
      </PopoverTrigger>
      <PopoverContent className="w-[600px]" side="bottom" align="end">
        <div className="grid grid-cols-2 gap-6">
          {/* Left Column - Screenshot Settings */}
          <div className="space-y-4">
            <div className="space-y-3">
              <div className="space-y-2">
                <Label className="text-sm font-medium">
                  {t("settings.screenshots")}: {settings.screenshotCount}
                </Label>
                <Slider
                  value={[settings.screenshotCount]}
                  onValueChange={([value]) => onUpdateSettings({ screenshotCount: value })}
                  max={20}
                  min={1}
                  step={1}
                />
              </div>
              <div className="space-y-2">
                <Label className="text-sm font-medium">
                  {t("settings.imageMiniatureSize")}: {settings.imageMiniatureSize}px
                </Label>
                <Slider
                  value={[settings.imageMiniatureSize]}
                  onValueChange={([value]) => onUpdateSettings({ imageMiniatureSize: value })}
                  max={800}
                  min={100}
                  step={50}
                />
              </div>
              <div className="space-y-2">
                <Label className="text-sm font-medium">
                  {t("settings.quality")}: {settings.screenshotQuality}
                </Label>
                <Slider
                  value={[settings.screenshotQuality]}
                  onValueChange={([value]) => onUpdateSettings({ screenshotQuality: value })}
                  max={5}
                  min={1}
                  step={1}
                />
              </div>
              <div className="space-y-2">
                <Label className="text-sm font-medium">
                  {t("settings.parallelGeneration")}: {settings.maxConcurrentScreenshots}
                </Label>
                <Slider
                  value={[settings.maxConcurrentScreenshots]}
                  onValueChange={([value]) => onUpdateSettings({ maxConcurrentScreenshots: value })}
                  max={30}
                  min={1}
                  step={1}
                />
              </div>
              <div className="space-y-2">
                <Label className="text-sm font-medium">
                  {t("settings.parallelUploads")}: {settings.maxConcurrentUploads}
                </Label>
                <Slider
                  value={[settings.maxConcurrentUploads]}
                  onValueChange={([value]) => onUpdateSettings({ maxConcurrentUploads: value })}
                  max={20}
                  min={1}
                  step={1}
                />
              </div>
            </div>
          </div>

          {/* Right Column - Service Settings */}
          <div className="space-y-4">
            <div className="space-y-4">
              {/* Fastpic Settings */}
              <div className="space-y-2">
                <Label htmlFor="fastpicSid" className="text-sm font-medium">
                  {t("settings.fastpicSid")}
                </Label>
                <Input
                  id="fastpicSid"
                  value={settings.fastpicSid}
                  onChange={(e) => onUpdateSettings({ fastpicSid: e.target.value })}
                  placeholder={t("settings.fastpicSidPlaceholder")}
                  className="w-full"
                />
              </div>

              <Separator />

              {/* Hamster Settings */}
              <div className="space-y-2">
                <Label htmlFor="hamsterEmail" className="text-sm font-medium">
                  {t("settings.hamsterEmail")}
                </Label>
                <Input
                  id="hamsterEmail"
                  type="email"
                  value={settings.hamsterEmail || ""}
                  onChange={(e) => onUpdateSettings({ hamsterEmail: e.target.value })}
                  placeholder={t("settings.hamsterEmailPlaceholder")}
                  className="w-full"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="hamsterPassword" className="text-sm font-medium">
                  {t("settings.hamsterPassword")}
                </Label>
                <Input
                  id="hamsterPassword"
                  type="password"
                  value={settings.hamsterPassword || ""}
                  onChange={(e) => onUpdateSettings({ hamsterPassword: e.target.value })}
                  placeholder={t("settings.hamsterPasswordPlaceholder")}
                  className="w-full"
                />
              </div>
              <p className="text-xs text-muted-foreground">{t("settings.hamsterDescription")}</p>

              <Separator />

              {/* MTN Settings */}
              <div className="space-y-2">
                <Label htmlFor="mtnArgs" className="text-sm font-medium">
                  {t("settings.mtnArgs")}
                </Label>
                <Textarea
                  id="mtnArgs"
                  value={settings.mtnArgs}
                  onChange={(e) => onUpdateSettings({ mtnArgs: e.target.value })}
                  placeholder={t("settings.mtnArgsPlaceholder")}
                  rows={4}
                  className="w-full text-xs font-mono"
                />
                <p className="text-xs text-muted-foreground">{t("settings.mtnArgsDescription")}</p>
              </div>

              <Separator />

              {/* Save Media Settings */}
              <div className="space-y-2">
                <Label className="text-sm font-medium">{t("settings.saveMedia")}</Label>
                <p className="text-xs text-muted-foreground">{t("settings.saveMediaDescription")}</p>
                <div className="flex gap-2">
                  <Input
                    value={settings.saveMediaDirectory || ""}
                    readOnly
                    placeholder={t("settings.saveMediaDirectoryPlaceholder")}
                    className="flex-1 text-xs"
                  />
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={async () => {
                      try {
                        const dir = await SpoilerService.SelectSaveMediaDirectory();
                        if (dir) {
                          onUpdateSettings({ saveMediaDirectory: dir });
                        }
                      } catch (err) {
                        console.error("Failed to select directory:", err);
                      }
                    }}
                  >
                    <FolderOpen className="h-4 w-4" />
                  </Button>
                  {settings.saveMediaDirectory && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => onUpdateSettings({ saveMediaDirectory: "" })}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
