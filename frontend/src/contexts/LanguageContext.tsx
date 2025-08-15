import React, { createContext, useContext, useState, useEffect, ReactNode } from "react";

// Import translation files
import enTranslations from "../locales/en.json";
import ruTranslations from "../locales/ru.json";

export type SupportedLanguage = "en" | "ru";

interface LanguageContextType {
  language: SupportedLanguage;
  t: (key: string) => string;
  setLanguage: (lang: SupportedLanguage) => void;
}

const translations = {
  en: enTranslations,
  ru: ruTranslations,
};

const LanguageContext = createContext<LanguageContextType | undefined>(undefined);

// Helper function to get nested object value by dot notation path
const getNestedValue = (obj: any, path: string): string => {
  return path.split(".").reduce((current, key) => current?.[key], obj) || path;
};

// Function to detect system language
const getSystemLanguage = (): SupportedLanguage => {
  const systemLang = navigator.language.toLowerCase();
  return systemLang.startsWith("ru") ? "ru" : "en";
};

interface LanguageProviderProps {
  children: ReactNode;
}

export const LanguageProvider: React.FC<LanguageProviderProps> = ({ children }) => {
  const [language, setLanguage] = useState<SupportedLanguage>("en");

  useEffect(() => {
    // Set language based on system locale on initial load
    const systemLang = getSystemLanguage();
    setLanguage(systemLang);

    // Add keyboard listener for Ctrl+L to toggle language (debugging)
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.ctrlKey && event.code === "KeyL") {
        event.preventDefault();
        setLanguage((prev) => (prev === "en" ? "ru" : "en"));
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  const t = (key: string): string => {
    return getNestedValue(translations[language], key);
  };

  const value: LanguageContextType = {
    language,
    t,
    setLanguage,
  };

  return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>;
};

// Custom hook to use language context
export const useLanguage = (): LanguageContextType => {
  const context = useContext(LanguageContext);
  if (!context) {
    throw new Error("useLanguage must be used within a LanguageProvider");
  }
  return context;
};

// Convenience hook for just the translation function
export const useTranslation = () => {
  const { t } = useLanguage();
  return { t };
};
