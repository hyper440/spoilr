import { useCallback } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Upload } from "lucide-react";

export default function DropZone() {
  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
  }, []);

  return (
    <Card className="bg-black/10 border-white/5 border-dashed border-2 hover:border-purple-400/30 transition-all duration-300 mb-6">
      <CardContent className="p-12">
        <div className="text-center" onDrop={handleDrop} onDragOver={handleDragOver}>
          <div className="mx-auto mb-6 p-4 bg-gradient-to-br from-purple-600/20 to-blue-600/20 rounded-full w-fit">
            <Upload className="w-12 h-12 text-purple-400" />
          </div>
          <h3 className="text-2xl font-semibold mb-3 text-white">Drop Video Files Anywhere</h3>
        </div>
      </CardContent>
    </Card>
  );
}
