'use client';

// Vendor approval queue — master list + detail pane, approve/reject/
// request-edits with a 6s delayed-mutation undo toast and auto-advance,
// hotkeys a/r/e via keydown. Port of VendorQueue.jsx.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Camera, CheckCircle2, Edit3, FileText, Flag, MapPin, Undo2, Wifi, XCircle } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'next/navigation';

import type { Vendor } from '@spotto/shared';
import Shell from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

type Action = 'approve' | 'reject' | 'edits';
const ACTION_ENDPOINT: Record<Action, string> = {
  approve: 'approve',
  reject: 'reject',
  edits: 'request-edits',
};
const ACTION_LABEL: Record<Action, string> = {
  approve: 'Approved',
  reject: 'Rejected',
  edits: 'Edits requested',
};

interface PendingUndo {
  id: string;
  action: Action;
  timer: ReturnType<typeof setTimeout>;
}

export default function QueuePage() {
  const qc = useQueryClient();
  const { data: vendors = [], isError } = useQuery<Vendor[]>({
    queryKey: ['admin-vendors', 'pending'],
    queryFn: () => api.get<Vendor[]>('/admin/vendors?status=pending'),
  });

  const searchParams = useSearchParams();
  const queryId = searchParams.get('id');

  const [removedIds, setRemovedIds] = useState<Set<string>>(new Set());
  const [selectedId, setSelectedId] = useState<string | null>(queryId);
  const [toast, setToast] = useState<PendingUndo | null>(null);
  const pendingRef = useRef<PendingUndo | null>(null);

  useEffect(() => {
    if (queryId) setSelectedId(queryId);
  }, [queryId]);

  const visible = vendors.filter((v) => !removedIds.has(v.id));
  useEffect(() => {
    if (!selectedId && visible.length > 0) setSelectedId(visible[0].id);
  }, [visible, selectedId]);
  const selected = visible.find((v) => v.id === selectedId) ?? null;

  const runAction = useMutation({
    mutationFn: ({ id, action }: { id: string; action: Action }) =>
      api.post(`/admin/vendors/${id}/${ACTION_ENDPOINT[action]}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-vendors'] }),
  });

  const act = useCallback(
    (id: string, action: Action) => {
      // Optimistically hide the row; auto-advance selection.
      setRemovedIds((prev) => new Set(prev).add(id));
      const idx = visible.findIndex((v) => v.id === id);
      const next = visible[idx + 1] ?? visible[idx - 1];
      setSelectedId(next?.id ?? null);

      if (pendingRef.current) clearTimeout(pendingRef.current.timer);
      const timer = setTimeout(() => {
        runAction.mutate({ id, action });
        pendingRef.current = null;
        setToast(null);
      }, 6000);
      pendingRef.current = { id, action, timer };
      setToast({ id, action, timer });
    },
    [visible, runAction],
  );

  const undo = () => {
    if (!pendingRef.current) return;
    clearTimeout(pendingRef.current.timer);
    setRemovedIds((prev) => {
      const next = new Set(prev);
      next.delete(pendingRef.current!.id);
      return next;
    });
    setSelectedId(pendingRef.current.id);
    pendingRef.current = null;
    setToast(null);
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!selected) return;
      if (['INPUT', 'TEXTAREA'].includes((e.target as HTMLElement)?.tagName)) return;
      if (e.key === 'a') act(selected.id, 'approve');
      if (e.key === 'r') act(selected.id, 'reject');
      if (e.key === 'e') act(selected.id, 'edits');
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [selected, act]);

  return (
    <Shell>
      {isError && <InlineErrorBanner message="Couldn't load the approval queue — try refreshing." />}
      <div className="grid grid-cols-[340px_1fr] gap-5 items-start">
        <div className="panel overflow-hidden">
          <div className="px-4 py-3 border-b border-line flex items-center justify-between">
            <p className="font-bold text-[13.5px]">{visible.length} pending</p>
            <p className="text-[11px] text-muted font-semibold">a·approve r·reject e·edits</p>
          </div>
          <div className="max-h-[calc(100vh-160px)] overflow-y-auto">
            {visible.map((v) => (
              <button
                key={v.id}
                onClick={() => setSelectedId(v.id)}
                className="w-full text-left px-4 py-3 border-b border-line last:border-0 transition-colors"
                style={{ background: selected?.id === v.id ? 'color-mix(in srgb, var(--accent) 10%, transparent)' : 'transparent' }}
              >
                <div className="flex items-center justify-between">
                  <p className="font-bold text-[13.5px] truncate">{v.name}</p>
                  {v.priority === 'high' && <Flag size={13} className="text-[#e11d48] shrink-0" />}
                </div>
                <p className="text-[12px] text-muted font-medium">{v.category} · {v.town}</p>
                {v.flags && v.flags.length > 0 && (
                  <p className="text-[11px] font-bold mt-1" style={{ color: '#d97706' }}>{v.flags[0]}</p>
                )}
              </button>
            ))}
            {visible.length === 0 && (
              <p className="px-4 py-8 text-center text-muted font-semibold text-[13px]">Queue is empty 🎉</p>
            )}
          </div>
        </div>

        {selected ? (
          <div className="panel p-6">
            <div className="flex items-start justify-between mb-5">
              <div>
                <h2 className="text-xl font-bold">{selected.name}</h2>
                <p className="text-muted font-medium text-[13.5px] mt-0.5">
                  {selected.category} · {selected.town} · owner {selected.owner}
                </p>
              </div>
              <span className="text-[11px] font-bold uppercase rounded-full px-2.5 py-1"
                style={{ background: selected.priority === 'high' ? '#fde4d84d' : 'var(--bg)', color: selected.priority === 'high' ? '#e11d48' : 'var(--muted)' }}>
                {selected.priority} priority
              </span>
            </div>

            <div className="grid grid-cols-3 gap-3 mb-5">
              <InfoTile icon={FileText} label="Documents" value={selected.docs ? 'Submitted' : 'Missing'} ok={selected.docs} />
              <InfoTile icon={Camera} label="Photos" value={`${selected.photos}`} ok={selected.photos >= 4} />
              <InfoTile icon={Wifi} label="Wi-Fi score" value={selected.wifi != null ? selected.wifi.toFixed(1) : '—'} ok={(selected.wifi ?? 0) >= 6} />
            </div>

            {selected.flags && selected.flags.length > 0 && (
              <div className="rounded-xl p-4 mb-5" style={{ background: '#fdf0cd4d', border: '1px solid #f6b81e55' }}>
                <p className="font-bold text-[13px] flex items-center gap-2" style={{ color: '#a97b00' }}>
                  <Flag size={14} /> Flags
                </p>
                <ul className="mt-1.5 space-y-1">
                  {selected.flags.map((f) => (
                    <li key={f} className="text-[13px] font-medium">{f}</li>
                  ))}
                </ul>
              </div>
            )}

            <p className="text-[14px] font-medium leading-relaxed mb-6">{selected.note}</p>

            {selected.regionId && (
              <p className="flex items-center gap-1.5 text-[12.5px] text-muted font-semibold mb-6">
                <MapPin size={13} /> Region: {selected.regionId}
              </p>
            )}

            <div className="flex gap-3">
              <button className="btn btn-primary flex-1 py-2.5 flex items-center justify-center gap-2" onClick={() => act(selected.id, 'approve')}>
                <CheckCircle2 size={16} /> Approve <kbd className="opacity-60">A</kbd>
              </button>
              <button className="btn flex-1 py-2.5 flex items-center justify-center gap-2" onClick={() => act(selected.id, 'edits')}>
                <Edit3 size={16} /> Request edits <kbd className="opacity-60">E</kbd>
              </button>
              <button className="btn btn-danger flex-1 py-2.5 flex items-center justify-center gap-2" onClick={() => act(selected.id, 'reject')}>
                <XCircle size={16} /> Reject <kbd className="opacity-60">R</kbd>
              </button>
            </div>
          </div>
        ) : (
          <div className="panel p-10 text-center text-muted font-semibold">Nothing selected.</div>
        )}
      </div>

      {toast && (
        <div className="toast-in fixed bottom-6 left-1/2 -translate-x-1/2 panel px-5 py-3 flex items-center gap-4 shadow-xl">
          <span className="font-bold text-[13.5px]">{ACTION_LABEL[toast.action]}</span>
          <button onClick={undo} className="flex items-center gap-1.5 font-bold text-[13px]" style={{ color: 'var(--accent)' }}>
            <Undo2 size={14} /> Undo
          </button>
        </div>
      )}
    </Shell>
  );
}

function InfoTile({ icon: Icon, label, value, ok }: { icon: typeof FileText; label: string; value: string; ok: boolean }) {
  return (
    <div className="rounded-xl p-3.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <p className="flex items-center gap-1.5 text-[11.5px] font-bold text-muted"><Icon size={13} /> {label}</p>
      <p className="text-[15px] font-bold mt-1" style={{ color: ok ? 'var(--ink)' : '#e11d48' }}>{value}</p>
    </div>
  );
}
