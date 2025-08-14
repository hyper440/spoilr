import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Separator } from "@/components/ui/separator"
import { Slider } from "@/components/ui/slider"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Settings } from "lucide-react"
import { AppSettings } from "@bindings/changeme/backend"

interface SettingsPopoverProps {
  settings: AppSettings
  onUpdateSettings: (settings: Partial<AppSettings>) => void
}

export default function SettingsPopover({ settings, onUpdateSettings }: SettingsPopoverProps) {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <div 
          className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background border border-white/20 bg-background hover:bg-accent hover:text-accent-foreground h-10 py-2 px-4 cursor-pointer"
          role="button"
          tabIndex={0}
        >
          <Settings className="w-4 h-4 mr-2" />
          Settings
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-80 bg-black/90 border-white/10">
        <div className="space-y-4">
          <h4 className="font-medium text-white">Application Settings</h4>
          
          {/* Fastpic Settings */}
          <div className="space-y-2">
            <Label htmlFor="fastpicSid" className="text-slate-300">Fastpic SID</Label>
            <Input 
              id="fastpicSid"
              value={settings.fastpicSid}
              onChange={(e) => onUpdateSettings({ fastpicSid: e.target.value })}
              placeholder="fp_sid cookie value"
              className="bg-black/40 border-white/10 text-white"
            />
          </div>

          <Separator className="bg-white/5" />

          {/* Screenshot Settings */}
          <div className="space-y-3">
            <div>
              <Label className="text-slate-300">Screenshots: {settings.screenshotCount}</Label>
              <Slider
                value={[settings.screenshotCount]}
                onValueChange={([value]) => onUpdateSettings({ screenshotCount: value })}
                max={12}
                min={1}
                step={1}
                className="mt-2"
              />
            </div>
            <div>
              <Label className="text-slate-300">Quality: {settings.screenshotQuality}</Label>
              <Slider
                value={[settings.screenshotQuality]}
                onValueChange={([value]) => onUpdateSettings({ screenshotQuality: value })}
                max={5}
                min={1}
                step={1}
                className="mt-2"
              />
            </div>
          </div>

          <Separator className="bg-white/5" />

          {/* Display Options */}
          <div className="space-y-3">
            <div className="flex items-center space-x-2">
              <Checkbox 
                id="hideEmpty" 
                checked={settings.hideEmpty}
                onCheckedChange={(checked) => onUpdateSettings({ hideEmpty: checked as boolean })}
              />
              <Label htmlFor="hideEmpty" className="text-slate-300">Hide empty parameters</Label>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}