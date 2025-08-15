import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Slider } from "@/components/ui/slider";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Textarea } from "@/components/ui/textarea";
import { AppSettings } from "@bindings/spoilr/backend";
import { useTranslation } from "@/contexts/LanguageContext";
import AnimatedText from "@/components/AnimatedText";

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
      <PopoverContent className="w-80" side="bottom" align="end">
        <div className="space-y-4">
          <h4 className="font-medium text-base">{t("settings.title")}</h4>

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
            />
          </div>

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
              rows={3}
              className="text-xs font-mono"
            />
            <p className="text-xs text-muted-foreground">{t("settings.mtnArgsDescription")}</p>
          </div>

          <Separator />

          {/* Screenshot Settings */}
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
      </PopoverContent>
    </Popover>
  );
}
