import { ThemeProvider } from "@/components/theme-provider";
import { useState, useEffect } from "react";
import { SpoilerService, AppSettings, AppState, Movie } from "@bindings/changeme/backend";
import { Events, WML } from "@wailsio/runtime";

import DropZone from "./components/DropZone";
import MovieTable from "./components/MovieTable";
import Header from "./components/Header";

function App() {
  const [state, setState] = useState<AppState>({
    processing: false,
    movies: [],
  });
  const [settings, setSettings] = useState<AppSettings>({
    hideEmpty: true,
    uiFontSize: 12,
    listFontSize: 10,
    textFontSize: 12,
    screenshotCount: 6,
    fastpicSid: "",
    screenshotQuality: 2,
    maxConcurrentScreenshots: 3,
    maxConcurrentUploads: 2,
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

    Events.On("state", handleStateUpdate);

    // Load initial state immediately
    SpoilerService.GetState().then(setState).catch(console.error);

    WML.Reload();

    return () => {
      Events.Off("state");
    };
  }, []);

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
      console.error("Failed to load initial data:", error);
    }
  };

  const startProcessing = async () => {
    try {
      await SpoilerService.StartProcessing();
    } catch (error) {
      console.error("Failed to start processing:", error);
    }
  };

  const cancelProcessing = async () => {
    try {
      await SpoilerService.CancelProcessing();
    } catch (error) {
      console.error("Failed to cancel processing:", error);
    }
  };

  const clearMovies = async () => {
    try {
      await SpoilerService.ClearMovies();
    } catch (error) {
      console.error("Failed to clear movies:", error);
    }
  };

  const removeMovie = async (id: string) => {
    try {
      await SpoilerService.RemoveMovie(id);
    } catch (error) {
      console.error("Failed to remove movie:", error);
    }
  };

  const saveTemplate = async (newTemplate: string) => {
    try {
      await SpoilerService.SetTemplate(newTemplate);
      setTemplate(newTemplate);
    } catch (error) {
      console.error("Failed to save template:", error);
    }
  };

  const resetTemplateToDefault = async () => {
    try {
      const defaultTemplate = await SpoilerService.GetDefaultTemplate();
      await SpoilerService.SetTemplate(defaultTemplate);
      setTemplate(defaultTemplate);
    } catch (error) {
      console.error("Failed to reset template to default:", error);
    }
  };

  const updateSettings = async (newSettings: Partial<AppSettings>) => {
    const updated = { ...settings, ...newSettings };
    setSettings(updated);
    try {
      await SpoilerService.UpdateSettings(updated);
    } catch (error) {
      console.error("Failed to update settings:", error);
    }
  };

  const copyMovieResult = async (movieId: string) => {
    try {
      const result = await SpoilerService.GenerateResultForMovie(movieId);
      await navigator.clipboard.writeText(result);
    } catch (error) {
      console.error("Failed to copy result:", error);
    }
  };

  const copyAllResults = async () => {
    try {
      const result = await SpoilerService.GenerateResult();
      await navigator.clipboard.writeText(result);
    } catch (error) {
      console.error("Failed to copy results:", error);
    }
  };

  const onReorderMovies = async (newMovies: Movie[]) => {
    try {
      // Extract just the IDs in the new order
      const newOrder = newMovies.map((movie) => movie.id);
      await SpoilerService.ReorderMovies(newOrder);
      // State will be updated via the event listener
    } catch (error) {
      console.error("Failed to reorder movies:", error);
    }
  };

  const pendingMovies = state.movies?.filter((m) => m.processingState === "pending") || [];
  const hasMovies = (state.movies?.length || 0) > 0;

  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <div
        className="wails-drag h-screen w-full flex flex-col overflow-hidden
        backdrop-blur-xl shadow-lg 
        bg-[radial-gradient(circle_at_center,rgba(0,0,0,0.6),rgba(0,0,0,0.9))]"
      >
        <div className="relative z-10 container mx-auto p-6 max-w-7xl">
          <div className="backdrop-blur-xl bg-white/2 border border-white/5 rounded-3xl p-6 shadow-2xl">
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
      </div>
    </ThemeProvider>
  );
}

export default App;
