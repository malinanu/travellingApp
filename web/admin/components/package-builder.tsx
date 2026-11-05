'use client';

// Package Builder — Kanban / To-Do board style drag-and-drop feature allocator
// and pricing manager for vendor tiers (dynamic multi-tier: Starter, Growth, Pro, Enterprise, etc.).
// Adheres strictly to stitch-design-taste and design-taste-frontend-v1:
// - Strict typography and calibrated contrast
// - Real spring-physics hover transitions
// - No emojis, clean Lucide iconography
// - Zero third-party drag-and-drop runtime bloat (native HTML5 DnD + accessible click fallback)
// - Supports Super Admin creation of brand new package tiers, reordering, shuffling, and deletion

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertCircle,
  Award,
  BarChart3,
  Calendar,
  Check,
  ChevronLeft,
  ChevronRight,
  Compass,
  Copy,
  Crown,
  DollarSign,
  Film,
  GripHorizontal,
  GripVertical,
  Info,
  Infinity as InfinityIcon,
  Layers,
  LucideIcon,
  MapPin,
  Plus,
  QrCode,
  Radio,
  RotateCcw,
  Save,
  Search,
  ShieldAlert,
  ShieldCheck,
  Shuffle,
  Sparkles,
  Tag,
  Trash2,
  TrendingUp,
  User,
  Users,
  X,
  Zap,
} from 'lucide-react';
import React, { useMemo, useState } from 'react';

import type { PackageFeature, PackagesConfig, VendorPackage } from '@spotto/shared';
import { api } from '../lib/api';

const FEATURE_ICONS: Record<string, LucideIcon> = {
  MapPin,
  ShieldCheck,
  Tag,
  Layers,
  Infinity: Sparkles,
  QrCode,
  User,
  Award,
  Zap,
  Radio,
  Calendar,
  Film,
  Users,
  TrendingUp,
  Compass,
  Crown,
  BarChart3,
  ShieldAlert,
  Sparkles,
};

const CATEGORY_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  discovery: { bg: 'rgba(13,148,136,0.12)', text: '#0d9488', border: 'rgba(13,148,136,0.25)' },
  marketing: { bg: 'rgba(246,184,30,0.14)', text: '#d97706', border: 'rgba(246,184,30,0.3)' },
  loyalty: { bg: 'rgba(219,39,119,0.12)', text: '#db2777', border: 'rgba(219,39,119,0.25)' },
  operations: { bg: 'rgba(3,105,161,0.12)', text: '#0284c7', border: 'rgba(3,105,161,0.25)' },
  insights: { bg: 'rgba(124,58,237,0.12)', text: '#7c3aed', border: 'rgba(124,58,237,0.25)' },
  trust: { bg: 'rgba(22,163,74,0.12)', text: '#16a34a', border: 'rgba(22,163,74,0.25)' },
};

export default function PackageBuilder() {
  const qc = useQueryClient();
  const [filterCategory, setFilterCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [draggedFeatureId, setDraggedFeatureId] = useState<string | null>(null);
  const [dragSource, setDragSource] = useState<string | null>(null); // 'pool' or package slug
  const [activeDropZone, setActiveDropZone] = useState<string | null>(null);
  const [statusMessage, setStatusMessage] = useState<{ text: string; type: 'success' | 'error' } | null>(null);

  // Column reordering & drag state
  const [draggedPkgSlug, setDraggedPkgSlug] = useState<string | null>(null);
  const [dropTargetPkgSlug, setDropTargetPkgSlug] = useState<string | null>(null);

  // Create New Package Modal State
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newPkgName, setNewPkgName] = useState('');
  const [newPkgSlug, setNewPkgSlug] = useState('');
  const [newPkgMonthly, setNewPkgMonthly] = useState<number>(19.99);
  const [newPkgYearly, setNewPkgYearly] = useState<number>(199.9);
  const [newPkgTagline, setNewPkgTagline] = useState('');
  const [newPkgBadge, setNewPkgBadge] = useState('');
  const [newPkgHighlighted, setNewPkgHighlighted] = useState(false);
  const [newPkgOffers, setNewPkgOffers] = useState<number>(10);
  const [newPkgFlash, setNewPkgFlash] = useState<number>(5);
  const [newPkgSeats, setNewPkgSeats] = useState<number>(5);
  const [slugManuallyEdited, setSlugManuallyEdited] = useState(false);

  // Fetch packages config from Go API
  const { data: remoteConfig, isLoading } = useQuery<PackagesConfig>({
    queryKey: ['admin-packages'],
    queryFn: () => api.get<PackagesConfig>('/admin/packages'),
  });

  // Local editable draft state
  const [draftConfig, setDraftConfig] = useState<PackagesConfig | null>(null);

  // Sync draft when remote data arrives or on initial load
  React.useEffect(() => {
    if (remoteConfig && !draftConfig) {
      setDraftConfig(remoteConfig);
    }
  }, [remoteConfig, draftConfig]);

  const isDirty = useMemo(() => {
    if (!remoteConfig || !draftConfig) return false;
    return JSON.stringify(remoteConfig.packages) !== JSON.stringify(draftConfig.packages);
  }, [remoteConfig, draftConfig]);

  const saveMutation = useMutation({
    mutationFn: (cfg: PackagesConfig) => api.put<PackagesConfig>('/admin/packages', cfg),
    onSuccess: (saved) => {
      setDraftConfig(saved);
      qc.invalidateQueries({ queryKey: ['admin-packages'] });
      setStatusMessage({ text: 'Packages, limits, and pricing published live to the marketing site!', type: 'success' });
      setTimeout(() => setStatusMessage(null), 5000);
    },
    onError: (err: any) => {
      setStatusMessage({ text: err?.message || 'Failed to publish packages.', type: 'error' });
      setTimeout(() => setStatusMessage(null), 6000);
    },
  });

  if (isLoading && !draftConfig) {
    return (
      <div className="panel p-8 text-center text-muted">
        <p className="text-[14px] font-medium">Loading package builder...</p>
      </div>
    );
  }

  const current = draftConfig || remoteConfig;
  if (!current) return null;

  const allMasterFeatures = current.featurePool || [];

  // Drag handlers for features
  const handleDragStart = (e: React.DragEvent, featureId: string, source: string) => {
    setDraggedFeatureId(featureId);
    setDragSource(source);
    e.dataTransfer.setData('text/plain', JSON.stringify({ featureId, source }));
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleDragOver = (e: React.DragEvent, zone: string) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    if (activeDropZone !== zone) {
      setActiveDropZone(zone);
    }
  };

  const handleDragLeave = (e: React.DragEvent, zone: string) => {
    if (e.currentTarget.contains(e.relatedTarget as Node)) return;
    if (activeDropZone === zone) {
      setActiveDropZone(null);
    }
  };

  const handleDrop = (e: React.DragEvent, targetZone: string) => {
    e.preventDefault();
    setActiveDropZone(null);

    const dataRaw = e.dataTransfer.getData('text/plain');
    if (!dataRaw) return;

    try {
      const { featureId, source } = JSON.parse(dataRaw);
      if (!featureId) return;

      moveFeature(featureId, source, targetZone);
    } catch {
      // ignore invalid drag data
    } finally {
      setDraggedFeatureId(null);
      setDragSource(null);
    }
  };

  const moveFeature = (featureId: string, fromZone: string, toZone: string, targetFeatureId?: string) => {
    const base = draftConfig || remoteConfig;
    if (!base) return;

    let featureToMove = allMasterFeatures.find((f) => f.id === featureId);
    if (!featureToMove && fromZone !== 'pool') {
      const sourcePkg = base.packages.find((p) => p.slug === fromZone);
      featureToMove = sourcePkg?.features.find((f) => f.id === featureId);
    }
    if (!featureToMove) return;

    // Moving back to pool (remove from the source package)
    if (toZone === 'pool') {
      removeFeatureFromTier(fromZone, featureId);
      return;
    }

    const nextPackages = base.packages.map((pkg) => {
      let features = [...pkg.features];
      let excluded = Array.isArray(pkg.excludedFeatures) ? [...pkg.excludedFeatures] : [];

      // Reordering within the same tier
      if (fromZone === toZone && pkg.slug === toZone) {
        const fromIdx = features.findIndex((f) => f.id === featureId);
        if (fromIdx !== -1) {
          const [removed] = features.splice(fromIdx, 1);
          if (targetFeatureId) {
            const targetIdx = features.findIndex((f) => f.id === targetFeatureId);
            if (targetIdx !== -1) {
              features.splice(targetIdx, 0, removed);
            } else {
              features.push(removed);
            }
          } else {
            features.push(removed);
          }
        }
        return { ...pkg, features };
      }

      // Remove from source tier
      if (pkg.slug === fromZone) {
        features = features.filter((f) => f.id !== featureId);
      }

      // Add to destination tier
      if (pkg.slug === toZone) {
        if (!features.some((f) => f.id === featureId)) {
          if (targetFeatureId) {
            const targetIdx = features.findIndex((f) => f.id === targetFeatureId);
            if (targetIdx !== -1) {
              features.splice(targetIdx, 0, featureToMove!);
            } else {
              features.push(featureToMove!);
            }
          } else {
            features.push(featureToMove!);
          }
        }
        // Remove from excludedFeatures if present
        excluded = excluded.filter((id) => id !== featureId);
      }

      return { ...pkg, features, excludedFeatures: excluded };
    });

    setDraftConfig({
      ...base,
      packages: nextPackages,
    });
  };

  const removeFeatureFromTier = (pkgSlug: string, featureId: string) => {
    const base = draftConfig || remoteConfig;
    if (!base) return;
    setDraftConfig({
      ...base,
      packages: base.packages.map((p) =>
        p.slug === pkgSlug ? { ...p, features: p.features.filter((f) => f.id !== featureId) } : p
      ),
    });
  };

  const addFeatureToTier = (pkgSlug: string, feature: PackageFeature) => {
    const base = draftConfig || remoteConfig;
    if (!base) return;
    setDraftConfig({
      ...base,
      packages: base.packages.map((p) => {
        if (p.slug === pkgSlug && !p.features.some((f) => f.id === feature.id)) {
          const excluded = (p.excludedFeatures || []).filter((id) => id !== feature.id);
          return { ...p, features: [...p.features, feature], excludedFeatures: excluded };
        }
        return p;
      }),
    });
  };

  const updatePackageField = (pkgSlug: string, field: keyof VendorPackage, value: any) => {
    const base = draftConfig || remoteConfig;
    if (!base) return;
    setDraftConfig({
      ...base,
      packages: base.packages.map((p) => (p.slug === pkgSlug ? { ...p, [field]: value } : p)),
    });
  };

  // --- Package Tier Creation, Reordering & Deletion ---

  const handleCreatePackage = (e: React.FormEvent) => {
    e.preventDefault();
    const base = draftConfig || remoteConfig;
    if (!base) return;

    const trimmedName = newPkgName.trim();
    const derivedSlug = trimmedName.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
    const cleanInputSlug = newPkgSlug.trim().toLowerCase().replace(/[^a-z0-9-]/g, '');
    const finalSlug = (cleanInputSlug || derivedSlug) || 'custom-tier';

    if (!trimmedName) {
      alert('Package name is required.');
      return;
    }

    if (base.packages.some((p) => p.slug === finalSlug)) {
      alert(`A package with slug "${finalSlug}" already exists. Please choose a distinct slug.`);
      return;
    }

    // Include basic starter features by default (verified map pin & BR badge)
    const initialFeatures = allMasterFeatures.filter(
      (f) => f.id === 'feat_map_pin' || f.id === 'feat_br_badge'
    );

    const newPackage: VendorPackage = {
      id: `pkg_${finalSlug}`,
      slug: finalSlug,
      name: trimmedName,
      tagline: newPkgTagline.trim() || `For ${trimmedName} hospitality and tourism operations`,
      monthlyPriceUsd: Number(newPkgMonthly) >= 0 ? Number(newPkgMonthly) : 0,
      yearlyPriceUsd: Number(newPkgYearly) >= 0 ? Number(newPkgYearly) : 0,
      badge: newPkgBadge.trim() || undefined,
      highlighted: newPkgHighlighted,
      activeOffers: newPkgOffers,
      flashSalesPerMonth: newPkgFlash,
      teamSeats: newPkgSeats,
      features: initialFeatures,
      excludedFeatures: [],
      sortOrder: base.packages.length + 1,
    };

    setDraftConfig({
      ...base,
      packages: [...base.packages, newPackage],
    });

    setIsCreateModalOpen(false);
    setNewPkgName('');
    setNewPkgSlug('');
    setNewPkgMonthly(19.99);
    setNewPkgYearly(199.9);
    setNewPkgTagline('');
    setNewPkgBadge('');
    setNewPkgHighlighted(false);
    setNewPkgOffers(10);
    setNewPkgFlash(5);
    setNewPkgSeats(5);
    setSlugManuallyEdited(false);

    setStatusMessage({
      text: `Created "${newPackage.name}" package tier! Drag features to allocate capabilities, then publish live.`,
      type: 'success',
    });
    setTimeout(() => setStatusMessage(null), 5000);
  };

  const handleDeletePackage = (pkgSlug: string) => {
    const base = draftConfig || remoteConfig;
    if (!base) return;
    if (base.packages.length <= 1) {
      alert('At least one package is required. You cannot delete the only remaining package.');
      return;
    }

    const target = base.packages.find((p) => p.slug === pkgSlug);
    if (!target) return;

    if (!confirm(`Are you sure you want to delete the "${target.name}" tier? Features assigned to it will return to the pool.`)) {
      return;
    }

    const remaining = base.packages
      .filter((p) => p.slug !== pkgSlug)
      .map((p, idx) => ({ ...p, sortOrder: idx + 1 }));

    setDraftConfig({
      ...base,
      packages: remaining,
    });

    setStatusMessage({ text: `Package tier "${target.name}" deleted.`, type: 'success' });
    setTimeout(() => setStatusMessage(null), 4000);
  };

  const handleMovePackage = (pkgSlug: string, direction: 'left' | 'right') => {
    const base = draftConfig || remoteConfig;
    if (!base) return;
    const pkgs = [...base.packages];
    const idx = pkgs.findIndex((p) => p.slug === pkgSlug);
    if (idx === -1) return;

    const targetIdx = direction === 'left' ? idx - 1 : idx + 1;
    if (targetIdx < 0 || targetIdx >= pkgs.length) return;

    const temp = pkgs[idx];
    pkgs[idx] = pkgs[targetIdx];
    pkgs[targetIdx] = temp;

    const reordered = pkgs.map((p, i) => ({ ...p, sortOrder: i + 1 }));
    setDraftConfig({
      ...base,
      packages: reordered,
    });
  };

  const handleShufflePackages = () => {
    const base = draftConfig || remoteConfig;
    if (!base || base.packages.length <= 1) return;
    // Rotate package order by 1 (first tier moves to end)
    const pkgs = [...base.packages];
    const first = pkgs.shift()!;
    pkgs.push(first);
    const reordered = pkgs.map((p, i) => ({ ...p, sortOrder: i + 1 }));
    setDraftConfig({
      ...base,
      packages: reordered,
    });
    setStatusMessage({ text: 'Package columns shuffled! Click "Publish to Live Website" to push live.', type: 'success' });
    setTimeout(() => setStatusMessage(null), 4000);
  };

  const handleDuplicatePackage = (pkg: VendorPackage) => {
    const base = draftConfig || remoteConfig;
    if (!base) return;
    let baseSlug = `${pkg.slug}-copy`;
    let counter = 1;
    while (base.packages.some((p) => p.slug === baseSlug)) {
      counter++;
      baseSlug = `${pkg.slug}-copy-${counter}`;
    }

    const copy: VendorPackage = {
      ...pkg,
      id: `pkg_${baseSlug}`,
      slug: baseSlug,
      name: `${pkg.name} Copy`,
      sortOrder: base.packages.length + 1,
      features: [...pkg.features],
      excludedFeatures: [...(pkg.excludedFeatures || [])],
    };

    setDraftConfig({
      ...base,
      packages: [...base.packages, copy],
    });

    setStatusMessage({ text: `Duplicated "${pkg.name}" tier as "${copy.name}".`, type: 'success' });
    setTimeout(() => setStatusMessage(null), 4000);
  };

  // Drag and drop for package columns
  const handleColDragStart = (e: React.DragEvent, slug: string) => {
    e.stopPropagation();
    setDraggedPkgSlug(slug);
    e.dataTransfer.setData('application/spotto-col', slug);
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleColDragOver = (e: React.DragEvent, slug: string) => {
    if (!draggedPkgSlug || draggedPkgSlug === slug) return;
    e.preventDefault();
    e.stopPropagation();
    e.dataTransfer.dropEffect = 'move';
    if (dropTargetPkgSlug !== slug) {
      setDropTargetPkgSlug(slug);
    }
  };

  const handleColDrop = (e: React.DragEvent, targetSlug: string) => {
    e.preventDefault();
    e.stopPropagation();
    const sourceSlug = e.dataTransfer.getData('application/spotto-col') || draggedPkgSlug;
    setDropTargetPkgSlug(null);
    setDraggedPkgSlug(null);

    const base = draftConfig || remoteConfig;
    if (!sourceSlug || sourceSlug === targetSlug || !base) return;

    const pkgs = [...base.packages];
    const sourceIdx = pkgs.findIndex((p) => p.slug === sourceSlug);
    const targetIdx = pkgs.findIndex((p) => p.slug === targetSlug);
    if (sourceIdx === -1 || targetIdx === -1) return;

    const [moved] = pkgs.splice(sourceIdx, 1);
    pkgs.splice(targetIdx, 0, moved);

    const reordered = pkgs.map((p, i) => ({ ...p, sortOrder: i + 1 }));
    setDraftConfig({
      ...base,
      packages: reordered,
    });
  };

  const handleReset = () => {
    if (confirm('Reset packages, pricing, and feature allocations to last saved version?')) {
      if (remoteConfig) {
        setDraftConfig(remoteConfig);
      }
    }
  };

  const categories = ['all', 'discovery', 'marketing', 'loyalty', 'operations', 'insights', 'trust'];

  const filteredPool = allMasterFeatures.filter((f) => {
    const matchesCategory = filterCategory === 'all' || f.category === filterCategory;
    const matchesSearch =
      searchQuery === '' ||
      f.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      f.description.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesCategory && matchesSearch;
  });

  return (
    <div className="space-y-6">
      {/* Top Banner / Actions Bar */}
      <div className="panel p-5 flex flex-wrap items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h2 className="text-[17px] font-bold tracking-tight text-ink">Vendor Package Builder</h2>
            <span className="text-[11px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full bg-accent/10 text-accent">
              Kanban Allocator
            </span>
            <span className="text-[11px] font-bold px-2 py-0.5 rounded-full bg-black/5 dark:bg-white/10 text-muted">
              {current.packages.length} Active {current.packages.length === 1 ? 'Tier' : 'Tiers'}
            </span>
            {isDirty && (
              <span className="text-[11px] font-bold px-2 py-0.5 rounded-full bg-amber-500/15 text-amber-700 dark:text-amber-400 border border-amber-500/30">
                Unsaved changes
              </span>
            )}
          </div>
          <p className="text-[13px] text-muted font-medium mt-1">
            Create custom package tiers, reorder columns, shuffle features, and set limits. When published, the public marketing site and feature matrix adapt live.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2.5">
          <button
            onClick={() => setIsCreateModalOpen(true)}
            className="btn btn-secondary !py-2 !px-3.5 text-[13px] flex items-center gap-1.5"
            title="Create a brand new package tier"
          >
            <Plus size={15} className="text-accent" />
            <span>New Package</span>
          </button>
          <button
            onClick={handleShufflePackages}
            disabled={current.packages.length <= 1}
            className="btn btn-secondary !py-2 !px-3 text-[13px] flex items-center gap-1.5 disabled:opacity-50"
            title="Rotate / shuffle package display order on the marketing site"
          >
            <Shuffle size={14} />
            <span>Shuffle Order</span>
          </button>
          <button
            onClick={handleReset}
            disabled={!isDirty}
            className="btn !py-2 !px-3.5 text-[13px] flex items-center gap-1.5 disabled:opacity-50"
            style={{ background: 'var(--surface)', borderColor: 'var(--line)' }}
            title="Reset to last saved state"
          >
            <RotateCcw size={14} /> Reset
          </button>
          <button
            onClick={() => draftConfig && saveMutation.mutate(draftConfig)}
            disabled={saveMutation.isPending}
            className="btn !py-2 !px-4 text-[13px] font-bold text-white flex items-center gap-2"
            style={{ background: 'var(--accent)', borderColor: 'var(--accent)' }}
          >
            <Save size={14} />
            {saveMutation.isPending ? 'Publishing...' : 'Publish to Live Website'}
          </button>
        </div>
      </div>

      {statusMessage && (
        <div
          className={`p-3.5 rounded-lg text-[13.5px] font-medium flex items-center gap-2 ${
            statusMessage.type === 'success'
              ? 'bg-emerald-500/10 text-emerald-600 border border-emerald-500/20'
              : 'bg-rose-500/10 text-rose-600 border border-rose-500/20'
          }`}
        >
          {statusMessage.type === 'success' ? <Check size={16} /> : <X size={16} />}
          <span>{statusMessage.text}</span>
        </div>
      )}

      {/* Main Kanban Workspace: Master Pool on Left, Dynamic Package Columns on Right */}
      <div className="flex flex-col xl:flex-row gap-5 items-start">
        {/* Left Column: Master Feature Pool (fixed width, sticky on desktop) */}
        <div className="w-full xl:w-[320px] flex-shrink-0">
          <div
            onDragOver={(e) => handleDragOver(e, 'pool')}
            onDragLeave={(e) => handleDragLeave(e, 'pool')}
            onDrop={(e) => handleDrop(e, 'pool')}
            className={`panel p-4 flex flex-col min-h-[640px] sticky top-4 transition-all duration-200 ${
              activeDropZone === 'pool' ? 'ring-2 ring-accent border-accent' : ''
            }`}
            style={{ background: 'var(--surface)' }}
          >
            <div className="pb-3 border-b border-line mb-3">
              <div className="flex items-center justify-between">
                <span className="text-[12px] font-bold uppercase tracking-wider text-muted">Master Feature Pool</span>
                <span className="text-[11px] font-bold px-2 py-0.5 rounded-full bg-black/5 dark:bg-white/10 text-muted">
                  {filteredPool.length} of {allMasterFeatures.length}
                </span>
              </div>
              <p className="text-[11.5px] text-muted mt-1">Available system capabilities. Drag or click tier to allocate.</p>

              {/* Feature Pool Search */}
              <div className="relative mt-2.5">
                <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted" />
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search capabilities..."
                  className="w-full pl-7 pr-2.5 py-1 text-[11.5px] font-medium bg-black/5 dark:bg-white/5 border border-line rounded-md outline-none focus:border-accent text-ink"
                />
                {searchQuery && (
                  <button
                    onClick={() => setSearchQuery('')}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-muted hover:text-ink"
                  >
                    <X size={12} />
                  </button>
                )}
              </div>

              {/* Category Filter Pills */}
              <div className="flex flex-wrap gap-1 mt-2.5">
                {categories.map((c) => (
                  <button
                    key={c}
                    onClick={() => setFilterCategory(c)}
                    className={`text-[10.5px] font-semibold px-2 py-0.5 rounded-md capitalize transition-colors ${
                      filterCategory === c
                        ? 'bg-accent text-white'
                        : 'bg-black/5 dark:bg-white/5 text-muted hover:text-ink'
                    }`}
                  >
                    {c}
                  </button>
                ))}
              </div>
            </div>

            <div className="space-y-2 flex-1 overflow-y-auto max-h-[720px] pr-1">
              {filteredPool.length === 0 ? (
                <div className="py-12 text-center text-muted">
                  <p className="text-[12px] font-medium">No matching features found.</p>
                </div>
              ) : (
                filteredPool.map((feat) => {
                  const IconComp = FEATURE_ICONS[feat.icon] || Sparkles;
                  const catColor = CATEGORY_COLORS[feat.category] || CATEGORY_COLORS.discovery;
                  const isDragging = draggedFeatureId === feat.id;

                  return (
                    <div
                      key={feat.id}
                      draggable
                      onDragStart={(e) => handleDragStart(e, feat.id, 'pool')}
                      className={`p-2.5 rounded-lg border border-line bg-surface hover:border-accent/40 cursor-grab active:cursor-grabbing transition-all group ${
                        isDragging ? 'opacity-40 scale-95' : 'hover:shadow-sm'
                      }`}
                    >
                      <div className="flex items-start gap-2">
                        <GripVertical size={14} className="text-muted/60 mt-0.5 flex-shrink-0 group-hover:text-accent" />
                        <div className="p-1 rounded bg-black/5 dark:bg-white/5 flex-shrink-0 text-ink">
                          <IconComp size={14} />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between gap-1">
                            <span className="text-[12.5px] font-bold text-ink truncate">{feat.name}</span>
                            <span
                              className="text-[9.5px] font-bold uppercase tracking-wider px-1.5 py-0.2 rounded"
                              style={{
                                background: catColor.bg,
                                color: catColor.text,
                                border: `1px solid ${catColor.border}`,
                              }}
                            >
                              {feat.category}
                            </span>
                          </div>
                          <p className="text-[11px] text-muted line-clamp-2 mt-0.5">{feat.description}</p>
                        </div>
                      </div>

                      {/* Quick-add buttons to all active packages */}
                      <div className="mt-2 pt-2 border-t border-line/50 flex flex-wrap items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <span className="text-[10px] text-muted mr-auto font-medium">Add to:</span>
                        {current.packages.map((pkg) => {
                          const alreadyInPkg = pkg.features.some((f) => f.id === feat.id);
                          return (
                            <button
                              key={pkg.slug}
                              disabled={alreadyInPkg}
                              onClick={() => addFeatureToTier(pkg.slug, feat)}
                              className={`text-[10px] font-bold px-1.5 py-0.5 rounded transition-colors max-w-[120px] truncate ${
                                alreadyInPkg
                                  ? 'bg-black/5 text-muted/50 cursor-not-allowed'
                                  : 'bg-black/5 hover:bg-accent hover:text-white'
                              }`}
                              title={alreadyInPkg ? 'Already included in this tier' : `Add to ${pkg.name}`}
                            >
                              {pkg.name}
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>
        </div>

        {/* Right Section: Dynamically Reorderable Package Columns */}
        <div className="flex-1 min-w-0 w-full overflow-x-auto pb-4">
          <div className="flex gap-5 items-start min-w-max">
            {current.packages.map((pkg, index) => {
              const isHighlighted = pkg.highlighted;
              const isDropOverFeature = activeDropZone === pkg.slug;
              const isDropOverCol = dropTargetPkgSlug === pkg.slug;
              const isDraggingCol = draggedPkgSlug === pkg.slug;

              return (
                <div
                  key={pkg.slug}
                  onDragOver={(e) => {
                    if (draggedPkgSlug) {
                      handleColDragOver(e, pkg.slug);
                    } else {
                      handleDragOver(e, pkg.slug);
                    }
                  }}
                  onDragLeave={(e) => {
                    if (draggedPkgSlug) {
                      if (dropTargetPkgSlug === pkg.slug) setDropTargetPkgSlug(null);
                    } else {
                      handleDragLeave(e, pkg.slug);
                    }
                  }}
                  onDrop={(e) => {
                    if (draggedPkgSlug) {
                      handleColDrop(e, pkg.slug);
                    } else {
                      handleDrop(e, pkg.slug);
                    }
                  }}
                  className={`panel p-4 flex flex-col min-h-[640px] w-[310px] flex-shrink-0 transition-all duration-200 relative ${
                    isDraggingCol ? 'opacity-30 scale-95' : ''
                  } ${
                    isDropOverCol ? 'ring-2 ring-accent border-accent scale-[1.01]' : ''
                  } ${
                    isDropOverFeature ? 'ring-2 ring-accent border-accent scale-[1.01]' : ''
                  } ${isHighlighted ? 'border-amber-500/50 shadow-md ring-1 ring-amber-500/30' : ''}`}
                  style={{ background: 'var(--surface)' }}
                >
                  {/* Highlight ribbon / badge */}
                  {pkg.badge && (
                    <div className="absolute -top-3 left-4">
                      <span
                        className="text-[10.5px] font-bold uppercase tracking-wider px-2.5 py-0.5 rounded-full text-white shadow-sm"
                        style={{ background: isHighlighted ? '#d97706' : '#0d9488' }}
                      >
                        {pkg.badge}
                      </span>
                    </div>
                  )}

                  {/* Tier Header: Reorder Controls, Drag Handle & Actions */}
                  <div className="pb-3 border-b border-line mb-3 pt-1">
                    <div className="flex items-center justify-between gap-1 mb-2">
                      <div
                        draggable
                        onDragStart={(e) => handleColDragStart(e, pkg.slug)}
                        className="flex items-center gap-1 text-muted hover:text-ink cursor-grab active:cursor-grabbing px-2 py-0.5 rounded bg-black/5 dark:bg-white/5 text-[11px] font-semibold"
                        title="Drag column header to reorder tier position"
                      >
                        <GripHorizontal size={13} />
                        <span>Tier {index + 1}</span>
                      </div>

                      <div className="flex items-center gap-0.5">
                        <button
                          type="button"
                          disabled={index === 0}
                          onClick={() => handleMovePackage(pkg.slug, 'left')}
                          className="p-1 rounded text-muted hover:text-ink hover:bg-black/5 dark:hover:bg-white/10 disabled:opacity-30 disabled:hover:bg-transparent"
                          title="Move package left"
                        >
                          <ChevronLeft size={14} />
                        </button>
                        <button
                          type="button"
                          disabled={index === current.packages.length - 1}
                          onClick={() => handleMovePackage(pkg.slug, 'right')}
                          className="p-1 rounded text-muted hover:text-ink hover:bg-black/5 dark:hover:bg-white/10 disabled:opacity-30 disabled:hover:bg-transparent"
                          title="Move package right"
                        >
                          <ChevronRight size={14} />
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDuplicatePackage(pkg)}
                          className="p-1 rounded text-muted hover:text-ink hover:bg-black/5 dark:hover:bg-white/10"
                          title="Duplicate this package tier"
                        >
                          <Copy size={13} />
                        </button>
                        <button
                          type="button"
                          disabled={current.packages.length <= 1}
                          onClick={() => handleDeletePackage(pkg.slug)}
                          className="p-1 rounded text-muted hover:text-rose-600 hover:bg-rose-500/10 disabled:opacity-30 disabled:hover:bg-transparent"
                          title={current.packages.length <= 1 ? 'Cannot delete the only remaining package' : 'Delete this package tier'}
                        >
                          <Trash2 size={13} />
                        </button>
                      </div>
                    </div>

                    {/* Tier Name & Featured Toggle */}
                    <div className="flex items-center justify-between gap-2">
                      <input
                        type="text"
                        value={pkg.name}
                        onChange={(e) => updatePackageField(pkg.slug, 'name', e.target.value)}
                        className="text-[15px] font-bold text-ink bg-transparent outline-none border-b border-transparent hover:border-line focus:border-accent flex-1 pb-0.5"
                        placeholder="Tier name..."
                      />
                      <label className="text-[11px] font-medium text-muted flex items-center gap-1 cursor-pointer flex-shrink-0">
                        <input
                          type="checkbox"
                          checked={pkg.highlighted}
                          onChange={(e) => updatePackageField(pkg.slug, 'highlighted', e.target.checked)}
                          className="rounded"
                        />
                        <span>Featured</span>
                      </label>
                    </div>

                    {/* Badge input */}
                    <div className="mt-2 flex items-center gap-1.5">
                      <span className="text-[10px] font-bold uppercase tracking-wider text-muted flex-shrink-0">Badge:</span>
                      <input
                        type="text"
                        value={pkg.badge || ''}
                        placeholder="e.g. Most Popular"
                        onChange={(e) => updatePackageField(pkg.slug, 'badge', e.target.value)}
                        className="flex-1 text-[11px] font-medium bg-black/5 dark:bg-white/5 rounded px-2 py-0.5 outline-none focus:ring-1 focus:ring-accent text-ink"
                      />
                    </div>

                    {/* Pricing Inputs */}
                    <div className="mt-2 grid grid-cols-2 gap-2">
                      <div className="p-2 rounded bg-black/5 dark:bg-white/5">
                        <span className="text-[10px] font-bold uppercase tracking-wider text-muted block">Monthly</span>
                        <div className="flex items-center gap-0.5 mt-0.5">
                          <span className="text-[13px] font-bold text-ink">$</span>
                          <input
                            type="number"
                            step="0.01"
                            value={pkg.monthlyPriceUsd}
                            onChange={(e) =>
                              updatePackageField(pkg.slug, 'monthlyPriceUsd', parseFloat(e.target.value) || 0)
                            }
                            className="w-full bg-transparent font-bold text-[15px] text-ink outline-none"
                          />
                        </div>
                      </div>

                      <div className="p-2 rounded bg-black/5 dark:bg-white/5">
                        <span className="text-[10px] font-bold uppercase tracking-wider text-muted block">Yearly</span>
                        <div className="flex items-center gap-0.5 mt-0.5">
                          <span className="text-[13px] font-bold text-ink">$</span>
                          <input
                            type="number"
                            step="0.01"
                            value={pkg.yearlyPriceUsd}
                            onChange={(e) =>
                              updatePackageField(pkg.slug, 'yearlyPriceUsd', parseFloat(e.target.value) || 0)
                            }
                            className="w-full bg-transparent font-bold text-[15px] text-ink outline-none"
                          />
                        </div>
                      </div>
                    </div>

                    {/* Subtitle / Tagline input */}
                    <input
                      type="text"
                      value={pkg.tagline}
                      onChange={(e) => updatePackageField(pkg.slug, 'tagline', e.target.value)}
                      className="mt-2.5 w-full text-[11.5px] text-muted font-medium bg-transparent border-b border-line/60 focus:border-accent outline-none pb-0.5"
                      placeholder="Tier tagline..."
                    />

                    {/* System Feature Limits (Editable) */}
                    <div className="mt-3 grid grid-cols-3 gap-1.5 text-center">
                      {/* Offers limit */}
                      <div className="p-1.5 rounded bg-black/5 dark:bg-white/5">
                        <span className="text-[9px] font-bold uppercase tracking-wider text-muted block">Offers</span>
                        <div className="mt-1 flex items-center justify-center gap-1">
                          {pkg.activeOffers === -1 ? (
                            <button
                              type="button"
                              onClick={() => updatePackageField(pkg.slug, 'activeOffers', 1)}
                              className="text-[11px] font-bold text-accent bg-accent/10 px-1.5 py-0.5 rounded"
                              title="Click to set numerical limit"
                            >
                              Unlimited
                            </button>
                          ) : (
                            <div className="flex items-center gap-0.5">
                              <input
                                type="number"
                                min="0"
                                value={pkg.activeOffers}
                                onChange={(e) =>
                                  updatePackageField(pkg.slug, 'activeOffers', parseInt(e.target.value, 10) || 0)
                                }
                                className="w-8 text-center text-[11px] font-bold bg-transparent outline-none border-b border-line"
                              />
                              <button
                                type="button"
                                onClick={() => updatePackageField(pkg.slug, 'activeOffers', -1)}
                                className="text-[10px] text-muted hover:text-accent font-bold"
                                title="Set unlimited"
                              >
                                ∞
                              </button>
                            </div>
                          )}
                        </div>
                      </div>

                      {/* Flash sales limit */}
                      <div className="p-1.5 rounded bg-black/5 dark:bg-white/5">
                        <span className="text-[9px] font-bold uppercase tracking-wider text-muted block">Flash/mo</span>
                        <div className="mt-1 flex items-center justify-center gap-1">
                          {pkg.flashSalesPerMonth === -1 ? (
                            <button
                              type="button"
                              onClick={() => updatePackageField(pkg.slug, 'flashSalesPerMonth', 2)}
                              className="text-[11px] font-bold text-accent bg-accent/10 px-1.5 py-0.5 rounded"
                              title="Click to set numerical limit"
                            >
                              Unlimited
                            </button>
                          ) : (
                            <div className="flex items-center gap-0.5">
                              <input
                                type="number"
                                min="0"
                                value={pkg.flashSalesPerMonth}
                                onChange={(e) =>
                                  updatePackageField(pkg.slug, 'flashSalesPerMonth', parseInt(e.target.value, 10) || 0)
                                }
                                className="w-8 text-center text-[11px] font-bold bg-transparent outline-none border-b border-line"
                              />
                              <button
                                type="button"
                                onClick={() => updatePackageField(pkg.slug, 'flashSalesPerMonth', -1)}
                                className="text-[10px] text-muted hover:text-accent font-bold"
                                title="Set unlimited"
                              >
                                ∞
                              </button>
                            </div>
                          )}
                        </div>
                      </div>

                      {/* Seats limit */}
                      <div className="p-1.5 rounded bg-black/5 dark:bg-white/5">
                        <span className="text-[9px] font-bold uppercase tracking-wider text-muted block">Seats</span>
                        <div className="mt-1 flex items-center justify-center gap-1">
                          {pkg.teamSeats === -1 ? (
                            <button
                              type="button"
                              onClick={() => updatePackageField(pkg.slug, 'teamSeats', 3)}
                              className="text-[11px] font-bold text-accent bg-accent/10 px-1.5 py-0.5 rounded"
                              title="Click to set numerical limit"
                            >
                              Unlimited
                            </button>
                          ) : (
                            <div className="flex items-center gap-0.5">
                              <input
                                type="number"
                                min="1"
                                value={pkg.teamSeats}
                                onChange={(e) =>
                                  updatePackageField(pkg.slug, 'teamSeats', parseInt(e.target.value, 10) || 1)
                                }
                                className="w-8 text-center text-[11px] font-bold bg-transparent outline-none border-b border-line"
                              />
                              <button
                                type="button"
                                onClick={() => updatePackageField(pkg.slug, 'teamSeats', -1)}
                                className="text-[10px] text-muted hover:text-accent font-bold"
                                title="Set unlimited"
                              >
                                ∞
                              </button>
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Features List in this Tier */}
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-[11.5px] font-bold uppercase tracking-wider text-muted">
                      Included Features ({pkg.features.length})
                    </span>
                    <span className="text-[10px] text-muted">Drag to reorder</span>
                  </div>

                  <div
                    className={`space-y-2 flex-1 overflow-y-auto max-h-[520px] pr-1 rounded-lg border-2 border-dashed p-1.5 transition-colors ${
                      isDropOverFeature ? 'border-accent bg-accent/5' : 'border-line/40'
                    }`}
                  >
                    {pkg.features.length === 0 ? (
                      <div className="py-12 text-center text-muted">
                        <p className="text-[12px] font-medium">No features assigned.</p>
                        <p className="text-[10.5px] mt-1">Drag features from the pool to include them.</p>
                      </div>
                    ) : (
                      pkg.features.map((feat) => {
                        const IconComp = FEATURE_ICONS[feat.icon] || Sparkles;
                        const isDragging = draggedFeatureId === feat.id;

                        return (
                          <div
                            key={feat.id}
                            draggable
                            onDragStart={(e) => handleDragStart(e, feat.id, pkg.slug)}
                            onDragOver={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                            }}
                            onDrop={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              const dataRaw = e.dataTransfer.getData('text/plain');
                              if (!dataRaw) return;
                              try {
                                const { featureId, source } = JSON.parse(dataRaw);
                                if (featureId) {
                                  moveFeature(featureId, source, pkg.slug, feat.id);
                                }
                              } catch {}
                            }}
                            className={`p-2.5 rounded-lg border border-line bg-surface hover:border-accent/40 cursor-grab active:cursor-grabbing transition-all group ${
                              isDragging ? 'opacity-40 scale-95' : 'hover:shadow-sm'
                            }`}
                          >
                            <div className="flex items-start gap-2">
                              <GripVertical
                                size={14}
                                className="text-muted/60 mt-0.5 flex-shrink-0 group-hover:text-accent"
                              />
                              <div className="p-1 rounded bg-black/5 dark:bg-white/5 flex-shrink-0 text-ink">
                                <IconComp size={14} />
                              </div>
                              <div className="flex-1 min-w-0">
                                <div className="flex items-center justify-between gap-1">
                                  <span className="text-[12.5px] font-bold text-ink truncate">{feat.name}</span>
                                  <button
                                    onClick={() => removeFeatureFromTier(pkg.slug, feat.id)}
                                    className="text-muted hover:text-rose-600 opacity-0 group-hover:opacity-100 transition-opacity p-0.5"
                                    title="Remove from tier"
                                  >
                                    <X size={13} />
                                  </button>
                                </div>
                                <p className="text-[11px] text-muted line-clamp-2 mt-0.5">{feat.description}</p>
                              </div>
                            </div>
                          </div>
                        );
                      })
                    )}
                  </div>
                </div>
              );
            })}

            {/* Quick Add Tier Placeholder Card */}
            <button
              type="button"
              onClick={() => setIsCreateModalOpen(true)}
              className="w-[280px] min-h-[460px] flex-shrink-0 rounded-2xl border-2 border-dashed border-line hover:border-accent flex flex-col items-center justify-center gap-3 text-muted hover:text-accent transition-all group p-6 text-center cursor-pointer"
              style={{ background: 'var(--surface)' }}
            >
              <div className="w-12 h-12 rounded-full bg-black/5 dark:bg-white/5 flex items-center justify-center group-hover:scale-110 group-hover:bg-accent/10 transition-all">
                <Plus size={24} className="group-hover:text-accent text-muted" />
              </div>
              <div>
                <span className="text-[14px] font-bold block text-ink group-hover:text-accent">Create New Tier</span>
                <span className="text-[12px] text-muted mt-1 block">Add custom subscription tier with tailored limits and capabilities</span>
              </div>
            </button>
          </div>
        </div>
      </div>

      {/* Create New Package Modal */}
      {isCreateModalOpen && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4">
          <div
            className="panel p-6 max-w-lg w-full space-y-5 shadow-2xl relative animate-in fade-in zoom-in-95 duration-150"
            style={{ background: 'var(--surface)' }}
          >
            <div className="flex items-center justify-between pb-3 border-b border-line">
              <div>
                <h3 className="text-[16px] font-bold text-ink">Create New Package Tier</h3>
                <p className="text-[12px] text-muted mt-0.5">Configure pricing, limits, and identifiers for the new tier.</p>
              </div>
              <button
                onClick={() => setIsCreateModalOpen(false)}
                className="p-1 rounded text-muted hover:text-ink"
              >
                <X size={18} />
              </button>
            </div>

            <form onSubmit={handleCreatePackage} className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[11px] font-bold uppercase tracking-wider text-muted block mb-1">
                    Package Name *
                  </label>
                  <input
                    type="text"
                    required
                    value={newPkgName}
                    onChange={(e) => {
                      setNewPkgName(e.target.value);
                      if (!slugManuallyEdited) {
                        const autoSlug = e.target.value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
                        setNewPkgSlug(autoSlug);
                      }
                    }}
                    placeholder="e.g. Enterprise"
                    className="w-full px-3 py-1.5 text-[13px] bg-black/5 dark:bg-white/5 border border-line rounded-lg outline-none focus:border-accent text-ink"
                  />
                </div>

                <div>
                  <label className="text-[11px] font-bold uppercase tracking-wider text-muted block mb-1">
                    Unique Slug *
                  </label>
                  <input
                    type="text"
                    required
                    value={newPkgSlug}
                    onChange={(e) => {
                      setSlugManuallyEdited(true);
                      setNewPkgSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''));
                    }}
                    placeholder="e.g. enterprise"
                    className="w-full px-3 py-1.5 text-[13px] bg-black/5 dark:bg-white/5 border border-line rounded-lg outline-none focus:border-accent text-ink"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[11px] font-bold uppercase tracking-wider text-muted block mb-1">
                    Monthly Price ($ USD)
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    required
                    value={newPkgMonthly}
                    onChange={(e) => {
                      const m = parseFloat(e.target.value) || 0;
                      setNewPkgMonthly(m);
                      // Suggest yearly price with 2 months free (10x monthly)
                      setNewPkgYearly(parseFloat((m * 10).toFixed(2)));
                    }}
                    className="w-full px-3 py-1.5 text-[13px] bg-black/5 dark:bg-white/5 border border-line rounded-lg outline-none focus:border-accent text-ink"
                  />
                </div>

                <div>
                  <label className="text-[11px] font-bold uppercase tracking-wider text-muted block mb-1">
                    Yearly Price ($ USD)
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    required
                    value={newPkgYearly}
                    onChange={(e) => setNewPkgYearly(parseFloat(e.target.value) || 0)}
                    className="w-full px-3 py-1.5 text-[13px] bg-black/5 dark:bg-white/5 border border-line rounded-lg outline-none focus:border-accent text-ink"
                  />
                </div>
              </div>

              <div>
                <label className="text-[11px] font-bold uppercase tracking-wider text-muted block mb-1">
                  Tagline / Pitch
                </label>
                <input
                  type="text"
                  value={newPkgTagline}
                  onChange={(e) => setNewPkgTagline(e.target.value)}
                  placeholder="e.g. For resort chains and multi-location hospitality groups"
                  className="w-full px-3 py-1.5 text-[13px] bg-black/5 dark:bg-white/5 border border-line rounded-lg outline-none focus:border-accent text-ink"
                />
              </div>

              <div className="grid grid-cols-2 gap-3 items-center">
                <div>
                  <label className="text-[11px] font-bold uppercase tracking-wider text-muted block mb-1">
                    Badge Ribbon (Optional)
                  </label>
                  <input
                    type="text"
                    value={newPkgBadge}
                    onChange={(e) => setNewPkgBadge(e.target.value)}
                    placeholder="e.g. VIP Priority"
                    className="w-full px-3 py-1.5 text-[13px] bg-black/5 dark:bg-white/5 border border-line rounded-lg outline-none focus:border-accent text-ink"
                  />
                </div>

                <div className="pt-4">
                  <label className="flex items-center gap-2 cursor-pointer text-[12.5px] font-semibold text-ink">
                    <input
                      type="checkbox"
                      checked={newPkgHighlighted}
                      onChange={(e) => setNewPkgHighlighted(e.target.checked)}
                      className="rounded"
                    />
                    <span>Highlight as Featured Tier</span>
                  </label>
                </div>
              </div>

              <div className="p-3 rounded-lg bg-black/5 dark:bg-white/5 space-y-2">
                <span className="text-[11px] font-bold uppercase tracking-wider text-muted block">
                  Initial Tier Limits (-1 = Unlimited)
                </span>
                <div className="grid grid-cols-3 gap-2">
                  <div>
                    <span className="text-[10px] text-muted block font-medium">Active Offers</span>
                    <input
                      type="number"
                      value={newPkgOffers}
                      onChange={(e) => setNewPkgOffers(parseInt(e.target.value, 10) || 0)}
                      className="w-full px-2 py-1 text-[12px] bg-surface border border-line rounded text-ink"
                    />
                  </div>
                  <div>
                    <span className="text-[10px] text-muted block font-medium">Flash Sales / mo</span>
                    <input
                      type="number"
                      value={newPkgFlash}
                      onChange={(e) => setNewPkgFlash(parseInt(e.target.value, 10) || 0)}
                      className="w-full px-2 py-1 text-[12px] bg-surface border border-line rounded text-ink"
                    />
                  </div>
                  <div>
                    <span className="text-[10px] text-muted block font-medium">Team Seats</span>
                    <input
                      type="number"
                      value={newPkgSeats}
                      onChange={(e) => setNewPkgSeats(parseInt(e.target.value, 10) || 0)}
                      className="w-full px-2 py-1 text-[12px] bg-surface border border-line rounded text-ink"
                    />
                  </div>
                </div>
              </div>

              <div className="pt-3 border-t border-line flex items-center justify-end gap-2.5">
                <button
                  type="button"
                  onClick={() => setIsCreateModalOpen(false)}
                  className="btn !py-2 !px-4 text-[13px]"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary !py-2 !px-5 text-[13px] font-bold"
                >
                  Create Package Tier
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
