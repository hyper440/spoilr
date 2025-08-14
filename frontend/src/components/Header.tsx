import { Button } from "@/components/ui/button"
import { Edit } from "lucide-react"
import {  AppSettings } from "@bindings/changeme/backend"
import SettingsPopover from './SettingsPopover'
import AnimatedText from "@/components/AnimatedText"

interface HeaderProps {
  editingTemplate: boolean
  onEditTemplate: () => void
  onSaveTemplate: () => void
  onCancelTemplate: () => void
  settings: AppSettings
  onUpdateSettings: (settings: Partial<AppSettings>) => void
}

export default function Header({
  editingTemplate,
  onEditTemplate,
  onSaveTemplate,
  onCancelTemplate,
  settings,
  onUpdateSettings,
}: HeaderProps) {
  return (
    <div className="flex items-center justify-between mb-6">
      <AnimatedText>Spoiler List Generator</AnimatedText>
      <div className="flex items-center gap-4">
        {/* Template Editor Toggle */}
        {!editingTemplate ? (
          <Button 
            onClick={onEditTemplate}
            variant="outline" 
            className="border-white/20"
          >
            <Edit className="w-4 h-4 mr-2" />
            Edit Template
          </Button>
        ) : (
          <div className="flex gap-2">
            <Button onClick={onSaveTemplate} className="bg-gradient-to-r from-blue-600 to-purple-600">
              Save Template
            </Button>
            <Button onClick={onCancelTemplate} variant="outline">
              Cancel
            </Button>
          </div>
        )}

        {/* Settings Popover */}
        <SettingsPopover settings={settings} onUpdateSettings={onUpdateSettings} />
      </div>
    </div>
  )
}