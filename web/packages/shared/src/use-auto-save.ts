import { useEffect, useRef, type DependencyList } from 'react';

/**
 * Debounced auto-save: while `dirty` is true, calls `onSave` ~`debounceMs`
 * after the last change to `deps` — never synchronously inside a
 * state-setting handler, only from a `useEffect`'s timer callback, which
 * only ever runs after React has committed the latest state. That
 * ordering matters: an earlier version of the vendor menu page called its
 * save mutation synchronously right after `setPresets(...)` in the same
 * event handler, which raced React's state batching and silently
 * persisted the *previous* state (see app/menu/page.tsx's own comment on
 * that incident). Scheduling from an effect instead avoids that class of
 * bug by construction, and TanStack Query's `mutate()` always executes
 * against whatever `mutationFn` closure was current as of the most recent
 * render regardless of when it's actually invoked, so a call fired later
 * (once the debounce timer elapses) still saves the latest value.
 *
 * `deps` works exactly like a `useEffect` dependency list — pass whatever
 * draft state should trigger a save when it changes (a single
 * useState-managed object/array, which only changes identity when you
 * actually call its setter, or several primitive fields listed
 * individually). Don't pass a fresh object/array literal built from
 * several separate `useState`s (e.g. `[{ name, email, phone }]`) — that
 * has a new identity every render regardless of whether anything actually
 * changed, which would keep re-triggering the debounce forever; list the
 * primitives directly instead (e.g. `[name, email, phone]`).
 *
 * Returns `flush()` — cancels any pending debounce and saves immediately.
 * Wire it to `onBlur` on text fields (so leaving a field never waits out
 * the debounce) and keep it as what the existing manual "Save" button
 * calls, so manual save keeps working exactly as a fallback.
 */
export function useAutoSave({
  deps,
  dirty,
  onSave,
  debounceMs = 1200,
}: {
  deps: DependencyList;
  dirty: boolean;
  onSave: () => void;
  debounceMs?: number;
}): { flush: () => void } {
  const onSaveRef = useRef(onSave);
  onSaveRef.current = onSave;
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!dirty) return;
    timeoutRef.current = setTimeout(() => {
      timeoutRef.current = null;
      onSaveRef.current();
    }, debounceMs);
    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, dirty, debounceMs]);

  return {
    // Explicitly clears any pending timer first so a manual flush (blur,
    // the fallback Save button) never races a duplicate debounced save
    // shortly after.
    flush: () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      }
      onSaveRef.current();
    },
  };
}
