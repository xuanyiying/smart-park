import React from 'react';
import type { CardProps } from '@/types';

/**
 * Card component with hover effect support
 */
export function Card({
  hoverable = false,
  className = '',
  children,
  ...props
}: CardProps) {
  const baseStyles = 'bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-2xl transition-all';

  const hoverStyles = hoverable
    ? 'cursor-pointer hover:border-indigo-500 hover:shadow-xl hover:shadow-indigo-500/15'
    : '';

  return (
    <div className={`${baseStyles} ${hoverStyles} ${className}`} {...props}>
      {children}
    </div>
  );
}