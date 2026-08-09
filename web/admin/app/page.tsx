'use client';

import { useQuery } from '@tanstack/react-query';
import { Users, Store, Ticket, Star, Loader2, ArrowRight } from 'lucide-react';
import Shell from '../components/shell';
import { api } from '../lib/api';
import Link from 'next/link';

interface PlatformAnalytics {
  totalUsers: number;
  totalVendors: number;
  totalClaims: number;
  totalReviews: number;
}

export default function DashboardPage() {
  const { data, isLoading, isError } = useQuery<PlatformAnalytics>({
    queryKey: ['adminAnalytics'],
    queryFn: () => api.get('/admin/analytics').then(r => r.data),
    refetchInterval: 30000,
  });

  if (isLoading) {
    return (
      <Shell>
        <div className="flex h-full items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-[var(--muted)]" />
        </div>
      </Shell>
    );
  }

  if (isError || !data) {
    return (
      <Shell>
        <div className="p-8 text-[var(--accent)] font-medium">Failed to load analytics</div>
      </Shell>
    );
  }

  const kpis = [
    { label: 'Total Travellers', value: data.totalUsers, icon: Users, color: 'text-blue-400', bg: 'bg-blue-400/10' },
    { label: 'Registered Vendors', value: data.totalVendors, icon: Store, color: 'text-emerald-400', bg: 'bg-emerald-400/10' },
    { label: 'Offers Claimed', value: data.totalClaims, icon: Ticket, color: 'text-purple-400', bg: 'bg-purple-400/10' },
    { label: 'Reviews Written', value: data.totalReviews, icon: Star, color: 'text-amber-400', bg: 'bg-amber-400/10' },
  ];

  return (
    <Shell>
      <div className="p-6 md:p-8 max-w-7xl mx-auto w-full">
        <header className="mb-10">
          <h1 className="text-3xl font-bold tracking-tight text-white mb-2">Spotto HQ</h1>
          <p className="text-[var(--muted)] text-sm md:text-base max-w-2xl">
            Real-time platform metrics and top-level activity overview.
          </p>
        </header>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 md:gap-6">
          {kpis.map((kpi, i) => {
            const Icon = kpi.icon;
            return (
              <div key={i} className="relative group overflow-hidden rounded-2xl bg-[#141517] border border-white/5 p-6 hover:border-white/10 transition-colors">
                <div className="flex items-center justify-between mb-4">
                  <div className={`p-3 rounded-xl ${kpi.bg}`}>
                    <Icon className={`h-6 w-6 ${kpi.color}`} />
                  </div>
                </div>
                <div className="flex flex-col gap-1">
                  <span className="text-4xl font-bold text-white tracking-tight">
                    {kpi.value.toLocaleString()}
                  </span>
                  <span className="text-sm font-medium text-[var(--muted)]">
                    {kpi.label}
                  </span>
                </div>
                
                {/* Subtle gradient glow effect */}
                <div className={`absolute -inset-x-4 -bottom-4 h-12 bg-gradient-to-t ${kpi.bg.replace('/10', '/0')} to-transparent opacity-0 group-hover:opacity-100 transition-opacity blur-xl`} />
              </div>
            );
          })}
        </div>

        <div className="mt-12 grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Quick Actions / Shortcuts */}
            <div className="rounded-2xl bg-[#141517] border border-white/5 p-6 flex flex-col">
              <h2 className="text-lg font-semibold text-white mb-4">Quick Actions</h2>
              <div className="flex flex-col gap-2 flex-grow">
                <Link href="/queue" className="flex items-center justify-between p-4 rounded-xl bg-white/5 hover:bg-white/10 transition-colors group">
                  <span className="text-sm font-medium text-white">Review Pending Vendors</span>
                  <ArrowRight className="h-4 w-4 text-[var(--muted)] group-hover:text-white transition-colors" />
                </Link>
                <Link href="/access-grants" className="flex items-center justify-between p-4 rounded-xl bg-white/5 hover:bg-white/10 transition-colors group">
                  <span className="text-sm font-medium text-white">Manage Profile Grants</span>
                  <ArrowRight className="h-4 w-4 text-[var(--muted)] group-hover:text-white transition-colors" />
                </Link>
                <Link href="/local-guide" className="flex items-center justify-between p-4 rounded-xl bg-white/5 hover:bg-white/10 transition-colors group">
                  <span className="text-sm font-medium text-white">Train Local Guide AI</span>
                  <ArrowRight className="h-4 w-4 text-[var(--muted)] group-hover:text-white transition-colors" />
                </Link>
              </div>
            </div>

            {/* Platform Security */}
            <SecuritySettingsPanel />
        </div>

      </div>
    </Shell>
  );
}

function SecuritySettingsPanel() {
  const { data, refetch } = useQuery<{ rateLimitRPM: number }>({
    queryKey: ['adminSettings'],
    queryFn: () => api.get('/admin/settings').then(r => r.data),
  });

  const updateSettings = useMutation({
    mutationFn: (rpm: number) => api.put('/admin/settings', { rateLimitRPM: rpm }),
    onSuccess: () => refetch(),
  });

  if (!data) return (
    <div className="rounded-2xl bg-[#141517] border border-white/5 p-6 flex items-center justify-center">
      <Loader2 className="h-6 w-6 animate-spin text-[var(--muted)]" />
    </div>
  );

  return (
    <div className="rounded-2xl bg-[#141517] border border-white/5 p-6 flex flex-col">
      <div className="flex items-center justify-between mb-2">
        <h2 className="text-lg font-semibold text-white">Platform Security</h2>
        {updateSettings.isPending && <Loader2 className="h-4 w-4 animate-spin text-[var(--muted)]" />}
      </div>
      <p className="text-sm text-[var(--muted)] mb-6">
        Adjust the global API rate limit (requests per minute per IP address) in real-time to defend against traffic spikes or abuse.
      </p>

      <div className="flex-grow flex flex-col justify-center">
        <div className="flex justify-between items-end mb-3">
          <label className="text-sm font-medium text-[var(--accent)]">Global API Rate Limit</label>
          <span className="text-xl font-bold text-white tracking-tight">{data.rateLimitRPM} <span className="text-sm font-normal text-[var(--muted)]">RPM</span></span>
        </div>
        <div className="flex items-center gap-4">
          <input 
            type="range" 
            min="10" 
            max="1000" 
            step="10"
            value={data.rateLimitRPM}
            onChange={(e) => updateSettings.mutate(parseInt(e.target.value, 10))}
            className="w-full accent-emerald-500 bg-white/10 h-2 rounded-lg appearance-none cursor-pointer"
          />
        </div>
        <div className="flex justify-between text-xs text-[var(--muted)] mt-2 font-medium uppercase tracking-wider">
          <span>Strict (10)</span>
          <span>Permissive (1000)</span>
        </div>
      </div>
    </div>
  );
}
