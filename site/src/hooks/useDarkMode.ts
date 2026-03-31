"use client";

import { useState, useEffect } from 'react';

interface UseDarkModeReturn {
  darkMode: boolean;
  toggleDarkMode: () => void;
  setDarkMode: (value: boolean) => void;
  mounted: boolean;
}

/**
 * Hook for managing dark mode state
 */
export function useDarkMode(): UseDarkModeReturn {
  const [darkMode, setDarkMode] = useState(false);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (mounted) {
      if (darkMode) {
        document.documentElement.classList.add('dark');
      } else {
        document.documentElement.classList.remove('dark');
      }
    }
  }, [darkMode, mounted]);

  const toggleDarkMode = () => {
    setDarkMode((prev) => !prev);
  };

  return {
    darkMode,
    toggleDarkMode,
    setDarkMode,
    mounted,
  };
}