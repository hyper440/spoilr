import {
  type AppSettings,
  type AppState,
  type Movie,
  SpoilerService,
} from "@bindings/spoilr/backend";
import { Events, WML } from "@wailsio/runtime";
import { useEffect, useState } from "react";
import { Toaster, toast } from "sonner";
import AnimatedText from "@/components/AnimatedText";
import { ThemeProvider } from "@/components/theme-provider";
import { LanguageProvider, useTranslation } from "@/contexts/LanguageContext";
import DropZone from "./components/DropZone";
import MovieTable from "./components/MovieTable";
import SettingsPopover from "./components/SettingsPopover";
import TemplateEditor from "./components/TemplateEditorPopover";

function AppContent() {
  const { t } = useTranslation();
  const [state, setState] = useState<AppState>({
    processing: false,
    movies: [],
  });
  const [settings, setSettings] = useState<AppSettings>({} as AppSettings);

  useEffect(() => {
    const doLoad = async () => {
      try {
        const [appSettings, initialState] = await Promise.all([
          SpoilerService.GetSettings(),
          SpoilerService.GetState(),
        ]);
        setSettings(appSettings);
        setState(initialState);
      } catch (error) {
        console.error("Failed to load initial data:", error);
      }
    };

    doLoad();

    const handleStateUpdate = (ev: Events.WailsEvent) => {
      console.log("State updated:", ev.data);
      // Wails v3 no longer wraps single data argument in array
      setState(ev.data as AppState);
    };

    const handleErrorEvent = (ev: Events.WailsEvent) => {
      const data = ev.data as { message: string };
      console.log(data);
      toast.error("Error", {
        description: data.message,
        duration: 8000,
      });
    };

    const offState = Events.On("state", handleStateUpdate);
    const offError = Events.On("error", handleErrorEvent);

    // Load initial state immediately
    SpoilerService.GetState().then(setState).catch(console.error);

    WML.Reload();

    return () => {
      offState();
      offError();
    };
  }, []);

  const startProcessing = async () => {
    try {
      await SpoilerService.StartProcessing();
    } catch (error) {
      console.error(t("errors.startProcessing"), error);
    }
  };

  const cancelProcessing = async () => {
    try {
      await SpoilerService.CancelProcessing();
    } catch (error) {
      console.error(t("errors.cancelProcessing"), error);
    }
  };

  const clearMovies = async () => {
    try {
      await SpoilerService.ClearMovies();
    } catch (error) {
      console.error(t("errors.clearMovies"), error);
    }
  };

  const removeMovie = async (id: string) => {
    try {
      await SpoilerService.RemoveMovie(id);
    } catch (error) {
      console.error(t("errors.removeMovie"), error);
    }
  };

  const resetTemplateToDefault = async () => {
    try {
      const defaultTemplate = await SpoilerService.GetDefaultTemplate();
      await SpoilerService.SetTemplate(defaultTemplate);
    } catch (error) {
      console.error(t("errors.resetTemplate"), error);
    }
  };

  const updateSettings = async (newSettings: Partial<AppSettings>) => {
    const updated = { ...settings, ...newSettings };
    setSettings(updated);
    try {
      await SpoilerService.UpdateSettings(updated);
    } catch (error) {
      console.error(t("errors.updateSettings"), error);
    }
  };

  const copyMovieResult = async (movieId: string) => {
    try {
      const result = await SpoilerService.GenerateResultForMovie(movieId);
      await navigator.clipboard.writeText(result);
    } catch (error) {
      console.error(t("errors.copyResult"), error);
    }
  };

  const copyAllResults = async () => {
    try {
      const result = await SpoilerService.GenerateResult();
      await navigator.clipboard.writeText(result);
    } catch (error) {
      console.error(t("errors.copyResults"), error);
    }
  };

  const onReorderMovies = async (newMovies: Movie[]) => {
    try {
      // Extract just the IDs in the new order
      const newOrder = newMovies.map((movie) => movie.id);
      await SpoilerService.ReorderMovies(newOrder);
      // State will be updated via the event listener
    } catch (error) {
      console.error(t("errors.reorderMovies"), error);
    }
  };

  const pendingMovies =
    state.movies?.filter((m) => m.processingState === "pending") || [];
  const hasMovies = (state.movies?.length || 0) > 0;

  return (
    <div
      className="wails-drag flex h-screen w-screen flex-col overflow-hidden
      bg-[radial-gradient(circle_at_center,rgba(0,0,0,0.6),rgba(0,0,0,0.9))]"
    >
      <div className="flex flex-col flex-1 min-h-0 w-full backdrop-blur-xl bg-white/2 p-6 shadow-2xl">
        {/* Integrated Header */}
        <div className="wails-drag flex items-center justify-between mb-6">
          <a
            href="https://github.com/hyper440/spoilr"
            data-wml-openurl="https://github.com/hyper440/spoilr"
            className="cursor-pointer"
          >
            <AnimatedText>Spoilr</AnimatedText>
          </a>
          <div className="wails-no-drag flex items-center gap-10">
            <TemplateEditor onResetTemplate={resetTemplateToDefault} />
            <SettingsPopover
              settings={settings}
              onUpdateSettings={updateSettings}
            />
          </div>
        </div>

        {!hasMovies ? (
          <DropZone />
        ) : (
          <MovieTable
            movies={state.movies}
            processing={state.processing}
            pendingCount={pendingMovies.length}
            onStartProcessing={startProcessing}
            onCancelProcessing={cancelProcessing}
            onClearMovies={clearMovies}
            onRemoveMovie={removeMovie}
            onCopyMovieResult={copyMovieResult}
            onCopyAllResults={copyAllResults}
            onReorderMovies={onReorderMovies}
          />
        )}
      </div>
    </div>
  );
}
function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <Toaster />
      <LanguageProvider>
        <AppContent />
      </LanguageProvider>
    </ThemeProvider>
  );
}

export default App;
