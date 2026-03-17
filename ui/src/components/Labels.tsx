interface LabelsProps {
  labels?: Record<string, string>;
  maxDisplay?: number;
}

function Labels({ labels, maxDisplay }: LabelsProps) {
  if (!labels || Object.keys(labels).length === 0) return null;

  const entries = Object.entries(labels);
  const displayEntries = maxDisplay ? entries.slice(0, maxDisplay) : entries;
  const remaining = maxDisplay ? Math.max(0, entries.length - maxDisplay) : 0;

  return (
    <div className="flex flex-wrap gap-1">
      {displayEntries.map(([key, value]) => (
        <span
          key={key}
          className="inline-flex items-center px-2 py-0.5 rounded-md text-[11px] bg-stone-50 text-stone-600 border border-stone-200"
          title={`${key}=${value}`}
        >
          <span className="text-stone-400">{key}=</span>
          <span>{value}</span>
        </span>
      ))}
      {remaining > 0 && (
        <span className="text-[11px] text-stone-400 self-center">+{remaining}</span>
      )}
    </div>
  );
}

export default Labels;
