import { AppSettings } from "@bindings/slg/backend";
import { useTranslation } from "@/contexts/LanguageContext";
import SettingsPopover from "./SettingsPopover";
import TemplateEditor from "./TemplateEditorPopover";
import AnimatedText from "@/components/AnimatedText";

interface HeaderProps {
  template: string;
  onTemplateChange: (template: string) => void;
  onResetTemplate: () => void;
  settings: AppSettings;
  onUpdateSettings: (settings: Partial<AppSettings>) => void;
}

export default function Header({ template, onTemplateChange, onResetTemplate, settings, onUpdateSettings }: HeaderProps) {
  const { t } = useTranslation();

  return (
    <div className="wails-drag flex items-center justify-between mb-6">
      <AnimatedText>{t("app.title")}</AnimatedText>
      <div className="wails-no-drag flex items-center gap-4">
        <TemplateEditor template={template} onTemplateChange={onTemplateChange} onResetToDefault={onResetTemplate} />
        <SettingsPopover settings={settings} onUpdateSettings={onUpdateSettings} />
      </div>
    </div>
  );
}
