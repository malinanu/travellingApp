'use client';

// Vendor subscription plans & payment checkout page.
// Allows vendors to upgrade from Starter ($0/mo) to Pro ($29/mo) or Enterprise ($99/mo)
// with direct server checkout integration via POST /vendor/subscription/checkout.

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Check, ShieldCheck, Zap, Sparkles, Crown } from 'lucide-react';

import Shell, { RoleLockedCard, useMe } from '../../components/shell';
import { api } from '../../lib/api';

const PLANS = [
  {
    id: 'starter',
    name: 'Starter',
    price: '$0',
    period: '/month',
    icon: Zap,
    popular: false,
    features: [
      'Basic Vendor Listing in Sri Lanka Spots',
      '1 Active Offer or Experience',
      'Standard Search Visibility',
      'Community Vibe Reviews',
    ],
  },
  {
    id: 'pro',
    name: 'Pro Partner',
    price: '$29',
    period: '/month',
    icon: Sparkles,
    popular: true,
    features: [
      'Unlimited Active Offers & Experiences',
      'Priority Geo-Search Placement',
      'Featured Region Boost & Flash Sales',
      'Real-Time Booking & Capacity System',
      'Dedicated Vendor Support',
    ],
  },
  {
    id: 'enterprise',
    name: 'Enterprise / Chain',
    price: '$99',
    period: '/month',
    icon: Crown,
    popular: false,
    features: [
      'Multi-Location Vendor Management',
      'Featured Homepage & Map Pin Placement',
      'Custom API & Audio Guide Integrations',
      'Advanced Analytics & PDF Reporting',
      '1-on-1 Account Manager & Setup',
    ],
  },
];

export default function SubscriptionPage() {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const isStaff = me?.vendorRole === 'staff';
  const [selectedTier, setSelectedTier] = useState<string>('pro');
  const [successTier, setSuccessTier] = useState<string | null>(null);

  const checkoutMutation = useMutation({
    mutationFn: (tier: string) => api.post('/vendor/subscription/checkout', { tier }),
    onSuccess: (_, tier) => {
      setSuccessTier(tier);
      qc.invalidateQueries({ queryKey: ['me'] });
    },
  });

  if (isStaff) {
    return (
      <Shell>
        <div className="mb-5">
          <h1 className="text-2xl font-extrabold font-display">Vendor Subscriptions</h1>
        </div>
        <RoleLockedCard what="Subscription Billing & Upgrades" />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="mb-6">
        <h1 className="text-2xl font-extrabold font-display text-emerald-950">Subscription Plans & Billing</h1>
        <p className="text-sm text-emerald-800/70">Upgrade your vendor presence to reach more travellers across Sri Lanka.</p>
      </div>

      {successTier && (
        <div className="mb-6 p-4 rounded-xl bg-emerald-50 border border-emerald-200 flex items-center gap-3">
          <ShieldCheck className="w-6 h-6 text-emerald-600 shrink-0" />
          <div>
            <h3 className="font-bold text-emerald-900">Subscription Updated!</h3>
            <p className="text-xs text-emerald-700">You are now subscribed to the <strong>{successTier.toUpperCase()}</strong> tier.</p>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        {PLANS.map((plan) => {
          const Icon = plan.icon;
          const isSelected = selectedTier === plan.id;
          return (
            <div
              key={plan.id}
              onClick={() => setSelectedTier(plan.id)}
              className={`relative rounded-2xl p-6 border transition-all cursor-pointer bg-white flex flex-col justify-between ${
                plan.popular
                  ? 'border-emerald-500 shadow-lg shadow-emerald-900/10 ring-2 ring-emerald-500/20'
                  : isSelected
                  ? 'border-emerald-600 ring-1 ring-emerald-600/30'
                  : 'border-emerald-900/10 hover:border-emerald-300'
              }`}
            >
              {plan.popular && (
                <div className="absolute -top-3 right-6 px-3 py-1 bg-emerald-600 text-white text-[10px] font-bold rounded-full uppercase tracking-wider">
                  MOST POPULAR
                </div>
              )}

              <div>
                <div className="w-10 h-10 rounded-xl bg-emerald-50 flex items-center justify-center mb-4 text-emerald-700">
                  <Icon className="w-5 h-5" />
                </div>

                <h3 className="font-bold text-lg text-emerald-950">{plan.name}</h3>
                <div className="flex items-baseline gap-1 my-3">
                  <span className="text-3xl font-extrabold text-emerald-950">{plan.price}</span>
                  <span className="text-xs text-emerald-700/60 font-medium">{plan.period}</span>
                </div>

                <ul className="space-y-2.5 my-6 text-xs text-emerald-900/80">
                  {plan.features.map((feat, i) => (
                    <li key={i} className="flex items-start gap-2">
                      <Check className="w-4 h-4 text-emerald-600 shrink-0 mt-0.5" />
                      <span>{feat}</span>
                    </li>
                  ))}
                </ul>
              </div>

              <button
                disabled={checkoutMutation.isPending}
                onClick={(e) => {
                  e.stopPropagation();
                  checkoutMutation.mutate(plan.id);
                }}
                className={`w-full py-2.5 px-4 rounded-xl text-xs font-bold transition-all ${
                  plan.popular
                    ? 'bg-emerald-700 hover:bg-emerald-800 text-white shadow-md shadow-emerald-900/15'
                    : 'bg-emerald-50 hover:bg-emerald-100 text-emerald-800'
                }`}
              >
                {checkoutMutation.isPending && selectedTier === plan.id
                  ? 'Processing...'
                  : `Select ${plan.name}`}
              </button>
            </div>
          );
        })}
      </div>
    </Shell>
  );
}
