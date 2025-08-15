import { ThemeProvider } from "@/components/theme-provider";
import { LanguageProvider, useTranslation } from "@/contexts/LanguageContext";
import { useState, useEffect } from "react";
import { SpoilerService, AppSettings, AppState, Movie } from "@bindings/spoilr/backend";
import { Events, WML } from "@wailsio/runtime";
import { toast } from "sonner";

import DropZone from "./components/DropZone";
import MovieTable from "./components/MovieTable";
import Header from "./components/Header";

function AppContent() {
  const { t } = useTranslation();
  const [state, setState] = useState<AppState>({
    processing: false,
    movies: [],
  });
  const [settings, setSettings] = useState<AppSettings>({
    screenshotCount: 6,
    fastpicSid: "",
    screenshotQuality: 2,
    maxConcurrentScreenshots: 3,
    maxConcurrentUploads: 2,
    mtnArgs: "",
  });
  const [template, setTemplate] = useState("");

  useEffect(() => {
    loadInitialData();

    const handleStateUpdate = (ev: any) => {
      console.log("State updated:", ev.data);

      // Wails sends data wrapped in array
      const newState = Array.isArray(ev.data) ? ev.data[0] : ev.data;
      console.log("Setting state:", newState);
      setState(newState as AppState);
    };

    const handleMtnMissing = (ev: any) => {
      const data = Array.isArray(ev.data) ? ev.data[0] : ev.data;
      toast.error(t("toast.mtnMissing"), {
        description: data.message || t("toast.mtnMissingDescription"),
        duration: 8000,
      });
    };

    Events.On("state", handleStateUpdate);
    Events.On("mtn-missing", handleMtnMissing);

    // Load initial state immediately
    SpoilerService.GetState().then(setState).catch(console.error);

    WML.Reload();

    return () => {
      Events.Off("state");
      Events.Off("mtn-missing");
    };
  }, [t]);

  const loadInitialData = async () => {
    try {
      const [appSettings, tmpl, initialState] = await Promise.all([
        SpoilerService.GetSettings(),
        SpoilerService.GetTemplate(),
        SpoilerService.GetState(),
      ]);

      setSettings(appSettings);
      setTemplate(tmpl);
      setState(initialState);
    } catch (error) {
      console.error(t("errors.loadInitialData"), error);
    }
  };

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

  const saveTemplate = async (newTemplate: string) => {
    try {
      await SpoilerService.SetTemplate(newTemplate);
      setTemplate(newTemplate);
    } catch (error) {
      console.error(t("errors.saveTemplate"), error);
    }
  };

  const resetTemplateToDefault = async () => {
    try {
      const defaultTemplate = await SpoilerService.GetDefaultTemplate();
      await SpoilerService.SetTemplate(defaultTemplate);
      setTemplate(defaultTemplate);
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

  const pendingMovies = state.movies?.filter((m) => m.processingState === "pending") || [];
  const hasMovies = (state.movies?.length || 0) > 0;

  return (
    <div
      className="wails-drag h-screen w-screen flex-col overflow-hidden
      bg-[radial-gradient(circle_at_center,rgba(0,0,0,0.6),rgba(0,0,0,0.9))]"
    >
      <div className="h-full self-center p-2backdrop-blur-xl bg-white/2 p-6 shadow-2xl">
        <Header
          template={template}
          onTemplateChange={saveTemplate}
          onResetTemplate={resetTemplateToDefault}
          settings={settings}
          onUpdateSettings={updateSettings}
        />

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
      <LanguageProvider>
        <AppContent />
      </LanguageProvider>
    </ThemeProvider>
  );
}

export default App;
