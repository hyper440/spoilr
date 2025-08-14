import { Button } from "@/components/ui/button"
import { Copy, Edit } from "lucide-react"
import { Movie, AppSettings } from "@bindings/changeme/backend"
import SettingsPopover from './SettingsPopover'

interface HeaderProps {
  editingTemplate: boolean
  onEditTemplate: () => void
  onSaveTemplate: () => void
  onCancelTemplate: () => void
  settings: AppSettings
  onUpdateSettings: (settings: Partial<AppSettings>) => void
  completedMovies: Movie[]
  onCopyAllResults: () => void
}

export default function Header({
  editingTemplate,
  onEditTemplate,
  onSaveTemplate,
  onCancelTemplate,
  settings,
  onUpdateSettings,
  completedMovies,
  onCopyAllResults
}: HeaderProps) {
  return (
    <div className="flex items-center justify-between mb-6">
      <h1 className="text-3xl font-bold text-white">Spoiler List Generator</h1>
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

        {/* Copy All Button */}
        {completedMovies.length > 0 && (
          <Button 
            onClick={onCopyAllResults}
            className="bg-gradient-to-r from-green-600 to-emerald-600"
          >
            <Copy className="w-4 h-4 mr-2" />
            Copy All ({completedMovies.length})
          </Button>
        )}
      </div>
    </div>
  )
}