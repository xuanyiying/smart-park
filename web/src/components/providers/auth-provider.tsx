"use client";

import { useAuthStore } from "@/stores/auth-store";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

interface AuthProviderProps {
  children: React.ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const router = useRouter();
  const [mounted, setMounted] = useState(false);

  // Get auth state after mount to avoid hydration mismatch
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const token = useAuthStore((state) => state.token);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (mounted && !isAuthenticated && !token) {
      router.push("/login");
    }
  }, [mounted, isAuthenticated, token, router]);

  // Don't render anything during SSR or before mount
  if (!mounted) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-gray-500">加载中...</div>
      </div>
    );
  }

  // After mount, check auth state
  if (!isAuthenticated && !token) {
    return null;
  }

  return <>{children}</>;
}
