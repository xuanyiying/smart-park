import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Format a monetary value given in cents (分) as a yuan string with two decimals.
 *
 * The backend stores all money as integer cents so no float rounding ever enters
 * the money path. Display layers must divide by 100 exactly once, at the edge.
 * Returns "12.50" for 1250, "0.00" for null/undefined.
 */
export function formatCents(
  cents: number | string | null | undefined,
): string {
  // protojson 会把 int64 序列化为字符串，这里统一兜底转换。
  const value = typeof cents === "string" ? Number(cents) : cents;
  if (value == null || Number.isNaN(value)) return "0.00";
  return (value / 100).toFixed(2);
}

/**
 * Same as formatCents but with thousands separators, for dashboards and reports:
 * 125000 -> "1,250.00".
 */
export function formatCentsGrouped(
  cents: number | string | null | undefined,
): string {
  const value = typeof cents === "string" ? Number(cents) : cents;
  if (value == null || Number.isNaN(value)) return "0.00";
  return (value / 100).toLocaleString("zh-CN", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}
