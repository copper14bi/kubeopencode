import { useState, useCallback, useEffect } from 'react';
import type { FilterState } from '../hooks/useFilterState';

interface ResourceFilterProps {
  onFilterChange: (filters: FilterState) => void;
  filters: FilterState;
  placeholder?: string;
}

function ResourceFilter({ onFilterChange, filters, placeholder }: ResourceFilterProps) {
  const [name, setName] = useState(filters.name);
  const [labelSelector, setLabelSelector] = useState(filters.labelSelector);

  useEffect(() => {
    setName(filters.name);
    setLabelSelector(filters.labelSelector);
  }, [filters.name, filters.labelSelector]);

  const handleApply = useCallback(() => {
    onFilterChange({ name, labelSelector });
  }, [name, labelSelector, onFilterChange]);

  const handleClear = useCallback(() => {
    setName('');
    setLabelSelector('');
    onFilterChange({ name: '', labelSelector: '' });
  }, [onFilterChange]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        handleApply();
      }
    },
    [handleApply]
  );

  const hasFilters = name || labelSelector;

  return (
    <div className="flex items-center space-x-2 flex-wrap gap-y-2">
      <div className="relative">
        <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-stone-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder || 'Filter by name...'}
          className="block w-48 sm:w-64 pl-9 rounded-lg border-stone-200 bg-white shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm text-stone-700 placeholder:text-stone-300"
        />
      </div>

      <input
        type="text"
        value={labelSelector}
        onChange={(e) => setLabelSelector(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Label filter (e.g. app=myapp)"
        className="block w-48 sm:w-56 rounded-lg border-stone-200 bg-white shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm text-stone-700 placeholder:text-stone-300"
      />

      <button
        onClick={handleApply}
        className="px-3.5 py-2 text-xs font-medium text-white bg-stone-900 rounded-lg hover:bg-stone-800 transition-colors"
      >
        Filter
      </button>

      {hasFilters && (
        <button
          onClick={handleClear}
          className="text-xs text-stone-400 hover:text-stone-600 transition-colors font-medium"
        >
          Clear
        </button>
      )}
    </div>
  );
}

export default ResourceFilter;
