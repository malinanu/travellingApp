'use client';

import { useEffect, useState } from 'react';

/** value, once it has stopped changing for ms — for lookups while someone types. */
export function useDebounced<T>(value: T, ms = 500): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return debounced;
}
