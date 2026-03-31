import React from 'react';

interface TableProps {
  children: React.ReactNode;
  className?: string;
}

interface TableHeadProps {
  children: React.ReactNode;
}

interface TableBodyProps {
  children: React.ReactNode;
}

interface TableRowProps {
  children: React.ReactNode;
  className?: string;
}

interface TableHeaderProps {
  children: React.ReactNode;
  className?: string;
}

interface TableCellProps {
  children: React.ReactNode;
  className?: string;
}

/**
 * Table component with header, body, row, header, and cell subcomponents
 */
export function Table({ children, className = '' }: TableProps) {
  return (
    <div className={`overflow-hidden ${className}`}>
      <table className="w-full text-sm">{children}</table>
    </div>
  );
}

export function TableHead({ children, className = '' }: TableHeadProps) {
  return (
    <thead className={`bg-slate-50 dark:bg-slate-800 ${className}`}>
      {children}
    </thead>
  );
}

export function TableBody({ children, className = '' }: TableBodyProps) {
  return <tbody className={className}>{children}</tbody>;
}

export function TableRow({ children, className = '' }: TableRowProps) {
  return (
    <tr
      className={`border-t border-slate-100 dark:border-slate-700 ${className}`}
    >
      {children}
    </tr>
  );
}

export function TableHeader({ children, className = '' }: TableHeaderProps) {
  return (
    <th
      className={`text-left p-3 text-slate-600 dark:text-slate-400 font-medium ${className}`}
    >
      {children}
    </th>
  );
}

export function TableCell({ children, className = '' }: TableCellProps) {
  return <td className={`p-3 text-slate-600 dark:text-slate-400 ${className}`}>{children}</td>;
}