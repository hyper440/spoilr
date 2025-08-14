import { AppSettings } from "@bindings/changeme/backend";
import SettingsPopover from "./SettingsPopover";
import TemplateEditor from "./TemplateEditorPopover";
import AnimatedText from "@/components/AnimatedText";

interface HeaderProps {
  template: string;
  onTemplateChange: (template: string) => void;
  settings: AppSettings;
  onUpdateSettings: (settings: Partial<AppSettings>) => void;
}

export default function Header({ template, onTemplateChange, settings, onUpdateSettings }: HeaderProps) {
  return (
    <div className="flex items-center justify-between mb-6">
      <AnimatedText>Spoiler List Generator</AnimatedText>
      <div className="flex items-center gap-4">
        {/* Template Editor Popover */}
        <TemplateEditor template={template} onTemplateChange={onTemplateChange} />

        {/* Settings Popover */}
        <SettingsPopover settings={settings} onUpdateSettings={onUpdateSettings} />
      </div>
    </div>
  );
}
