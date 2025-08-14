import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Slider } from "@/components/ui/slider";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Settings } from "lucide-react";
import { AppSettings } from "@bindings/changeme/backend";

interface SettingsPopoverProps {
  settings: AppSettings;
  onUpdateSettings: (settings: Partial<AppSettings>) => void;
}

export default function SettingsPopover({ settings, onUpdateSettings }: SettingsPopoverProps) {
  return (
    <Popover>
      <PopoverTrigger className=" cursor-pointer inline-flex items-center justify-center rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-8 px-2">
        <Settings className="w-4 h-4 mr-2" />
        Settings
      </PopoverTrigger>
      <PopoverContent className="w-80" side="bottom" align="end">
        <div className="space-y-4">
          <h4 className="font-medium text-base">Application Settings</h4>

          {/* Fastpic Settings */}
          <div className="space-y-2">
            <Label htmlFor="fastpicSid" className="text-sm font-medium">
              Fastpic SID
            </Label>
            <Input
              id="fastpicSid"
              value={settings.fastpicSid}
              onChange={(e) => onUpdateSettings({ fastpicSid: e.target.value })}
              placeholder="fp_sid cookie value"
            />
          </div>

          <Separator />

          {/* Screenshot Settings */}
          <div className="space-y-3">
            <div className="space-y-2">
              <Label className="text-sm font-medium">Screenshots: {settings.screenshotCount}</Label>
              <Slider
                value={[settings.screenshotCount]}
                onValueChange={([value]) => onUpdateSettings({ screenshotCount: value })}
                max={20}
                min={1}
                step={1}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">Quality: {settings.screenshotQuality}</Label>
              <Slider
                value={[settings.screenshotQuality]}
                onValueChange={([value]) => onUpdateSettings({ screenshotQuality: value })}
                max={5}
                min={1}
                step={1}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">Parallel Generation: {settings.maxConcurrentScreenshots}</Label>
              <Slider
                value={[settings.maxConcurrentScreenshots]}
                onValueChange={([value]) => onUpdateSettings({ maxConcurrentScreenshots: value })}
                max={30}
                min={1}
                step={1}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">Parallel Uploads: {settings.maxConcurrentUploads}</Label>
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
