import { useEffect, useState, ReactNode } from "react";
import { Moon, Sun, Monitor } from "lucide-react";

import { Button } from "./ui/button";
import {
  getNextTheme,
  getTheme,
  Theme,
  ThemeProviderContext,
  useTheme,
} from "@/components/ThemeProviderState.ts";
import { setMemory } from "@/utils/memory.ts";
import { themeEvent } from "@/events/theme.ts";

export type { Theme } from "@/components/ThemeProviderState.ts";

type ResolvedTheme = "dark" | "light";

type ThemeProviderProps = {
  children?: ReactNode;
  defaultTheme?: Theme;
};

export function ThemeProvider({
  defaultTheme = "system",
  ...props
}: ThemeProviderProps) {
  const [theme, setTheme] = useState<Theme>(
    () => getTheme() || defaultTheme,
  );

  useEffect(() => {
    const root = window.document.documentElement;
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    let disposed = false;
    let appliedTheme: ResolvedTheme | null = null;

    const browserSystemTheme = (): ResolvedTheme =>
      media.matches ? "dark" : "light";

    const resolveSystemTheme = async (): Promise<ResolvedTheme> => {
      try {
        const tauriTheme = await window.__TAURI__?.window?.appWindow?.theme?.();
        if (tauriTheme === "dark" || tauriTheme === "light") {
          return tauriTheme;
        }
      } catch {
        // Browser mode and older WebViews fall back to matchMedia.
      }

      return browserSystemTheme();
    };

    const applyTheme = async () => {
      const resolvedTheme =
        theme === "system" ? await resolveSystemTheme() : theme;

      if (disposed) return;
      if (appliedTheme === resolvedTheme) return;

      root.classList.remove("light", "dark");
      root.classList.add(resolvedTheme);
      appliedTheme = resolvedTheme;
      themeEvent.emit(resolvedTheme);
    };

    void applyTheme();

    if (theme !== "system") {
      return () => {
        disposed = true;
      };
    }

    const handleSystemThemeChange = () => {
      void applyTheme();
    };

    media.addEventListener?.("change", handleSystemThemeChange);
    media.addListener?.(handleSystemThemeChange);
    window.addEventListener("focus", handleSystemThemeChange);
    document.addEventListener("visibilitychange", handleSystemThemeChange);

    const timer = window.setInterval(handleSystemThemeChange, 1500);

    return () => {
      disposed = true;
      media.removeEventListener?.("change", handleSystemThemeChange);
      media.removeListener?.(handleSystemThemeChange);
      window.removeEventListener("focus", handleSystemThemeChange);
      document.removeEventListener("visibilitychange", handleSystemThemeChange);
      window.clearInterval(timer);
    };
  }, [theme]);

  const value = {
    theme,
    setTheme: (newTheme: Theme) => {
      setMemory("theme", newTheme);
      setTheme(newTheme);
    },
    toggleTheme: () => {
      const nextTheme: Theme = getNextTheme(theme);
      setMemory("theme", nextTheme);
      setTheme(nextTheme);
    },
  };

  return <ThemeProviderContext.Provider {...props} value={value} />;
}

export function ThemeToggle({ className, size = "icon" }: { className?: string; size?: "icon" | "icon-md" }) {
  const { theme, toggleTheme } = useTheme();

  return (
    <Button
      variant="outline"
      size={size}
      onClick={() => toggleTheme?.()}
      className={`!m-0 ${className || ''}`}
    >
      <Sun
        className={`h-4 w-4 transition-all ${theme === "light" ? "relative rotate-0 scale-100" : "absolute -rotate-90 scale-0"}`}
      />
      <Moon
        className={`h-4 w-4 transition-all ${theme === "dark" ? "relative rotate-0 scale-100" : "absolute rotate-90 scale-0"}`}
      />
      <Monitor
        className={`h-4 w-4 transition-all ${theme === "system" ? "relative rotate-0 scale-100" : "absolute rotate-90 scale-0"}`}
      />
    </Button>
  );
}

export default ThemeToggle;
