'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Calendar, CheckCircle2, ChevronRight, XCircle } from 'lucide-react';
import { useState } from 'react';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

export default function QuestsPage() {
  const [tab, setTab] = useState<'scheduler' | 'queue'>('queue');
  const qc = useQueryClient();

  const { data: pending = [] } = useQuery({
    queryKey: ['admin-quests', 'pending'],
    queryFn: () => api.get<any[]>('/admin/quests/sponsorships/pending'),
  });

  const { data: approved = [] } = useQuery({
    queryKey: ['admin-quests', 'approved'],
    queryFn: () => api.get<any[]>('/admin/quests/sponsorships/approved'),
  });

  const { data: schedules = [] } = useQuery({
    queryKey: ['admin-quests', 'schedules'],
    queryFn: () => api.get<any[]>('/admin/quests/schedules'),
  });

  const act = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.post(`/admin/quests/sponsorships/${id}/status`, { status }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-quests', 'pending'] });
      qc.invalidateQueries({ queryKey: ['admin-quests', 'approved'] });
    },
  });

  const schedule = useMutation({
    mutationFn: (req: { date: string; vendorId: string; goal: number }) =>
      api.post('/admin/quests/schedules', req),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-quests', 'schedules'] }),
  });

  return (
    <Shell>
      <div className="flex flex-col h-full bg-canvas">
        <header className="flex-none p-6 border-b border-white/5">
          <h1 className="text-[20px] font-display font-extrabold text-white">Quests</h1>
          <p className="text-[13.5px] text-muted font-semibold mt-1">Manage vendor sponsorships and daily quest schedule</p>
          
          <div className="flex gap-1 mt-6 p-1 bg-white/5 rounded-xl w-fit">
            <button
              onClick={() => setTab('queue')}
              className={`px-4 py-1.5 rounded-lg text-[13px] font-bold transition-all ${
                tab === 'queue' ? 'bg-white text-ink shadow-sm' : 'text-muted hover:text-white'
              }`}
            >
              Approval Queue
              {pending.length > 0 && (
                <span className="ml-2 inline-flex items-center justify-center bg-red-500 text-white text-[10px] w-4 h-4 rounded-full">
                  {pending.length}
                </span>
              )}
            </button>
            <button
              onClick={() => setTab('scheduler')}
              className={`px-4 py-1.5 rounded-lg text-[13px] font-bold transition-all ${
                tab === 'scheduler' ? 'bg-white text-ink shadow-sm' : 'text-muted hover:text-white'
              }`}
            >
              Scheduler
            </button>
          </div>
        </header>

        <div className="flex-1 overflow-auto p-6">
          {tab === 'queue' ? (
            <div className="space-y-4 max-w-3xl">
              {pending.length === 0 ? (
                <div className="text-center py-12 text-muted text-[14px]">No pending sponsorships</div>
              ) : (
                pending.map((s: any) => (
                  <div key={s.vendorId} className="card p-5 flex items-center justify-between">
                    <div>
                      <div className="font-bold text-[15px]">{s.vendorId}</div>
                      <div className="text-[13px] text-muted mt-1">Reward: {s.rewardDescription} (Limit: {s.rewardLimit})</div>
                    </div>
                    <div className="flex gap-2">
                      <button
                        onClick={() => act.mutate({ id: s.vendorId, status: 'rejected' })}
                        className="p-2 bg-red-500/10 text-red-500 rounded-lg hover:bg-red-500/20"
                      >
                        <XCircle size={18} />
                      </button>
                      <button
                        onClick={() => act.mutate({ id: s.vendorId, status: 'approved' })}
                        className="p-2 bg-green-500/10 text-green-500 rounded-lg hover:bg-green-500/20"
                      >
                        <CheckCircle2 size={18} />
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-8 max-w-5xl">
              <div>
                <h2 className="text-[14px] font-bold mb-4">Set Schedule</h2>
                <form
                  className="card p-5 space-y-4"
                  onSubmit={(e) => {
                    e.preventDefault();
                    const fd = new FormData(e.currentTarget);
                    schedule.mutate({
                      date: fd.get('date') as string,
                      vendorId: fd.get('vendorId') as string,
                      goal: 5,
                    });
                  }}
                >
                  <div>
                    <label className="block text-[12px] font-bold text-muted mb-1.5">Date</label>
                    <input name="date" type="date" required className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-[14px] outline-none focus:border-accent" />
                  </div>
                  <div>
                    <label className="block text-[12px] font-bold text-muted mb-1.5">Approved Sponsor</label>
                    <select name="vendorId" required className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-[14px] outline-none focus:border-accent">
                      {approved.map((s: any) => (
                        <option key={s.vendorId} value={s.vendorId}>{s.vendorId} - {s.rewardDescription}</option>
                      ))}
                    </select>
                  </div>
                  <button type="submit" className="w-full bg-accent text-accent-fg font-bold py-2 rounded-lg text-[13.5px]">
                    Schedule Quest
                  </button>
                </form>
              </div>

              <div>
                <h2 className="text-[14px] font-bold mb-4">Upcoming Quests</h2>
                <div className="space-y-3">
                  {schedules.map((s: any) => {
                    // Split the YYYY-MM-DD date string so it doesn't get messed up by local timezone
                    const [year, month, day] = s.date.split('-');
                    const d = new Date(parseInt(year), parseInt(month)-1, parseInt(day));
                    return (
                      <div key={s.date} className="flex items-center gap-4 bg-white/5 p-3 rounded-xl">
                        <div className="w-12 h-12 bg-white/10 rounded-lg flex flex-col items-center justify-center font-display">
                          <span className="text-[10px] text-muted font-bold uppercase">{d.toLocaleString('default', { month: 'short' })}</span>
                          <span className="text-[16px] font-extrabold">{d.getDate()}</span>
                        </div>
                        <div>
                          <div className="font-bold text-[14px]">{s.sponsorName}</div>
                          <div className="text-[12.5px] text-muted">{s.rewardDescription}</div>
                        </div>
                      </div>
                    );
                  })}
                  {schedules.length === 0 && <div className="text-[13px] text-muted">No quests scheduled.</div>}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </Shell>
  );
}
