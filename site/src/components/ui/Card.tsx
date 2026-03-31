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
    ? 'cursor-pointer hover:border-sky-500 hover:shadow-lg hover:shadow-sky-500/10'
    : '';

  return (
    <div className={`${baseStyles} ${hoverStyles} ${className}`} {...props}>
      {children}
    </div>
  );
}