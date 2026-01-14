import { Card, CardContent } from "@/components/ui/card";
import { Upload } from "lucide-react";
import { useTranslation } from "@/contexts/LanguageContext";

export default function DropZone() {
  const { t } = useTranslation();

  return (
    <Card
      data-wails-dropzone
      className="h-[600px] bg-black/10 border-white/5 border-dashed border-2 hover:border-purple-400/30 transition-all duration-300 [&.wails-dropzone-hover]:border-purple-400/60 [&.wails-dropzone-hover]:bg-purple-600/10"
    >
      <CardContent className="h-full flex items-center justify-center p-12">
        <div className="text-center">
          <div className="mx-auto mb-6 p-4 bg-gradient-to-br from-purple-600/20 to-blue-600/20 rounded-full w-fit">
            <Upload className="w-12 h-12 text-purple-400" />
          </div>
          <h3 className="text-2xl font-semibold mb-3 text-white">{t("dropzone.title")}</h3>
        </div>
      </CardContent>
    </Card>
  );
}
