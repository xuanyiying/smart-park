"use client";

import { Toaster as Sonner, toast as sonnerToast } from "sonner";

export function Toaster() {
  return (
    <Sonner
      position="top-center"
      richColors
      closeButton
      duration={4000}
      toastOptions={{
        style: {
          fontFamily: "inherit",
        },
      }}
    />
  );
}

export const toast = sonnerToast;
