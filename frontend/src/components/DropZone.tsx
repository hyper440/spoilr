import { Card, CardContent } from "@/components/ui/card";
import { Upload } from "lucide-react";

export default function DropZone() {
  return (
    <Card className="h-[600px] bg-black/10 border-white/5 border-dashed border-2 hover:border-purple-400/30 transition-all duration-300">
      <CardContent className="h-full flex items-center justify-center p-12">
        <div className="text-center">
          <div className="mx-auto mb-6 p-4 bg-gradient-to-br from-purple-600/20 to-blue-600/20 rounded-full w-fit">
            <Upload className="w-12 h-12 text-purple-400" />
          </div>
          <h3 className="text-2xl font-semibold mb-3 text-white">Drop Video Files Anywhere</h3>
        </div>
      </CardContent>
    </Card>
  );
}
