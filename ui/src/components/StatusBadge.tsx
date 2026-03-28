import React from 'react';

interface StatusBadgeProps {
  phase: string;
}

function StatusBadge({ phase }: StatusBadgeProps) {
  const lowerPhase = phase.toLowerCase();

  const styles: Record<string, { bg: string; text: string; dot: string; border: string }> = {
    pending: { bg: 'bg-slate-50', text: 'text-slate-600', dot: 'bg-slate-400', border: 'border-slate-200' },
    queued: { bg: 'bg-amber-50', text: 'text-amber-700', dot: 'bg-amber-400', border: 'border-amber-200' },
    running: { bg: 'bg-primary-50', text: 'text-primary-700', dot: 'bg-primary-400', border: 'border-primary-200' },
    completed: { bg: 'bg-emerald-50', text: 'text-emerald-700', dot: 'bg-emerald-400', border: 'border-emerald-200' },
    failed: { bg: 'bg-red-50', text: 'text-red-700', dot: 'bg-red-400', border: 'border-red-200' },
  };

  const style = styles[lowerPhase] || styles.pending;
  const isActive = lowerPhase === 'running' || lowerPhase === 'queued';

  return (
    <span
      className={`inline-flex items-center text-xs font-medium ${style.text}`}
    >
      {isActive ? (
        <span className="relative mr-1.5 flex h-1.5 w-1.5">
          <span className={`animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 ${style.dot}`} />
          <span className={`relative inline-flex rounded-full h-1.5 w-1.5 ${style.dot}`} />
        </span>
      ) : (
        <span className={`mr-1.5 inline-flex rounded-full h-1.5 w-1.5 ${style.dot}`} />
      )}
      {phase}
    </span>
  );
}

export default StatusBadge;
