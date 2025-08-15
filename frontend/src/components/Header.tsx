import { AppSettings } from "@bindings/spoilr/backend";
import { useTranslation } from "@/contexts/LanguageContext";
import SettingsPopover from "./SettingsPopover";
import TemplateEditor from "./TemplateEditorPopover";
import AnimatedText from "@/components/AnimatedText";
import { WML } from "@wailsio/runtime";

interface HeaderProps {
  template: string;
  onTemplateChange: (template: string) => void;
  onResetTemplate: () => void;
  settings: AppSettings;
  onUpdateSettings: (settings: Partial<AppSettings>) => void;
}

WML.Reload();

export default function Header({ template, onTemplateChange, onResetTemplate, settings, onUpdateSettings }: HeaderProps) {
  const { t } = useTranslation();

  return (
    <div className="wails-drag flex items-center justify-between mb-6">
      <a data-wml-openURL="https://github.com/hyper440/spoilr" className="cursor-pointer">
        <AnimatedText>{t("app.title")}</AnimatedText>
      </a>
      <div className="wails-no-drag flex items-center gap-10">
        <TemplateEditor template={template} onTemplateChange={onTemplateChange} onResetToDefault={onResetTemplate} />
        <SettingsPopover settings={settings} onUpdateSettings={onUpdateSettings} />
      </div>
    </div>
  );
}
