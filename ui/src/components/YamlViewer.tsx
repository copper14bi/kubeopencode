import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';

interface YamlViewerProps {
  queryKey: string[];
  fetchYaml: () => Promise<string>;
}

function YamlViewer({ queryKey, fetchYaml }: YamlViewerProps) {
  const [isOpen, setIsOpen] = useState(false);

  const { data: yaml, isLoading, error } = useQuery({
    queryKey: [...queryKey, 'yaml'],
    queryFn: fetchYaml,
    enabled: isOpen,
  });

  return (
    <div className="mt-6">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center space-x-2 text-xs font-display font-medium text-stone-400 hover:text-stone-600 uppercase tracking-wider transition-colors"
      >
        <svg
          className={`w-3.5 h-3.5 transform transition-transform ${isOpen ? 'rotate-90' : ''}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
        </svg>
        <span>YAML</span>
      </button>
      {isOpen && (
        <div className="mt-2 bg-stone-900 rounded-xl overflow-hidden border border-stone-800 animate-fade-in">
          <div className="px-4 py-2.5 bg-stone-800/50 flex items-center justify-between border-b border-stone-700/50">
            <span className="text-xs text-stone-400 font-display">Resource Definition</span>
            {yaml && (
              <button
                onClick={() => navigator.clipboard.writeText(yaml)}
                className="text-[11px] text-stone-500 hover:text-stone-300 transition-colors font-medium"
              >
                Copy
              </button>
            )}
          </div>
          <div className="p-4 max-h-96 overflow-y-auto sidebar-scroll">
            {isLoading ? (
              <span className="text-stone-500 text-sm">Loading...</span>
            ) : error ? (
              <span className="text-red-400 text-sm">Error: {(error as Error).message}</span>
            ) : (
              <pre className="text-xs text-stone-300 font-mono whitespace-pre leading-relaxed">{yaml}</pre>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export default YamlViewer;
