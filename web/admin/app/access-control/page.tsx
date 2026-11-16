'use client';

// Command Center User Access Control — Super Admin Permission Manager & Breaker.
// Exclusive access control management for Command Center features and capabilities.
// Features:
// 1. Lists all Command Center features in the master catalog.
// 2. Super Admins can break features down into granular To-Do sub-permissions (via button or dragging into Column 2).
// 3. Complete bidirectional To-Do list Kanban board with native HTML5 drag-and-drop options:
//    - Column 1: Feature Catalog & Available Pool (Drop to un-grant / reset)
//    - Column 2: Broken Down Granular Tasks (Drop features here to break down; drop tasks here to return to to-do)
//    - Column 3: Granted Access Controls (Drop features or tasks here to grant; drag out to revoke)
// 4. Quick presets, accessible click fallbacks, search/category filters, and audit log reason tracking.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Activity,
  AlertCircle,
  AlertTriangle,
  ArrowLeft,
  ArrowRight,
  Check,
  CheckSquare,
  ChevronDown,
  ChevronRight,
  Copy,
  CreditCard,
  Filter,
  GripVertical,
  History,
  Info,
  KeyRound,
  Layers,
  Lock,
  LucideIcon,
  Map,
  Minus,
  Newspaper,
  RotateCcw,
  Save,
  Scissors,
  Search,
  Shapes,
  Shield,
  ShieldAlert,
  ShieldCheck,
  Sliders,
  Sparkles,
  Split,
  Square,
  Store,
  Trash2,
  Unlock,
  UserCheck,
  Users,
  X,
  Zap,
} from 'lucide-react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import React, { Suspense, useMemo, useState } from 'react';

import type {
  AdminAccessUser,
  AdminGroup,
  CommandCenterFeature,
  CommandCenterPermission,
  PermissionRisk,
  User,
} from '@spotto/shared';
import Shell, { useMe } from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

const FEATURE_ICONS: Record<string, LucideIcon> = {
  UserCheck,
  Store,
  KeyRound,
  Users,
  Activity,
  Map,
  CreditCard,
  Layers,
  Newspaper,
  Sparkles,
  Shapes,
  History,
  ShieldAlert,
  ShieldCheck,
};

const CATEGORY_META: Record<string, { label: string; bg: string; text: string; border: string }> = {
  operations: { label: 'Operations', bg: 'rgba(3,105,161,0.12)', text: '#0284c7', border: 'rgba(3,105,161,0.25)' },
  security: { label: 'Security & Access', bg: 'rgba(225,29,72,0.12)', text: '#e11d48', border: 'rgba(225,29,72,0.25)' },
  monetization: { label: 'Monetization', bg: 'rgba(246,184,30,0.14)', text: '#d97706', border: 'rgba(246,184,30,0.3)' },
  editorial: { label: 'Editorial & CMS', bg: 'rgba(13,148,136,0.12)', text: '#0d9488', border: 'rgba(13,148,136,0.25)' },
  intelligence: { label: 'AI & Telemetry', bg: 'rgba(124,58,237,0.12)', text: '#7c3aed', border: 'rgba(124,58,237,0.25)' },
};

const RISK_META: Record<PermissionRisk, { label: string; bg: string; text: string }> = {
  low: { label: 'Low Risk', bg: 'rgba(22,163,74,0.12)', text: '#16a34a' },
  medium: { label: 'Medium', bg: 'rgba(246,184,30,0.14)', text: '#d97706' },
  high: { label: 'High Risk', bg: 'rgba(225,29,72,0.12)', text: '#e11d48' },
  critical: { label: 'Critical', bg: 'rgba(190,18,60,0.2)', text: '#be123c' },
};

export default function AccessControlPage() {
  return (
    <Shell>
      <Suspense fallback={<div className="panel p-8 text-center text-muted font-semibold">Loading access control manager…</div>}>
        <AccessControlManager />
      </Suspense>
    </Shell>
  );
}

function AccessControlManager() {
  const qc = useQueryClient();
  const searchParams = useSearchParams();
  const initialAdminId = searchParams.get('admin') || searchParams.get('user') || '';

  const { data: me } = useMe();
  const isSuperAdmin = !!me?.isSuperAdmin;

  // Load feature catalog
  const { data: catalogData, isLoading: isLoadingCatalog } = useQuery<{ features: CommandCenterFeature[] }>({
    queryKey: ['admin-access-control-catalog'],
    queryFn: () => api.get<{ features: CommandCenterFeature[] }>('/admin/access-control/catalog'),
  });

  // Load admin accounts
  const { data: admins = [], isLoading: isLoadingAdmins } = useQuery<AdminAccessUser[]>({
    queryKey: ['admin-access-control-admins'],
    queryFn: () => api.get<AdminAccessUser[]>('/admin/access-control/admins'),
  });

  // Load access control groups
  const { data: groups = [] } = useQuery<AdminGroup[]>({
    queryKey: ['admin-access-control-groups'],
    queryFn: () => api.get<AdminGroup[]>('/admin/access-control/groups'),
  });

  const [selectedAdminId, setSelectedAdminId] = useState<string>('');

  // Modals state for users & groups
  const [isAddUserModalOpen, setIsAddUserModalOpen] = useState(false);
  const [newUserEmail, setNewUserEmail] = useState('');
  const [newUserName, setNewUserName] = useState('');
  const [newUserGroupId, setNewUserGroupId] = useState('vendor_approvals');
  const [newUserReason, setNewUserReason] = useState('');

  const [isGroupsModalOpen, setIsGroupsModalOpen] = useState(false);
  const [isCreateGroupModalOpen, setIsCreateGroupModalOpen] = useState(false);
  const [newGroupName, setNewGroupName] = useState('');
  const [newGroupDesc, setNewGroupDesc] = useState('');
  const [newGroupIcon, setNewGroupIcon] = useState('Shield');
  const [newGroupPerms, setNewGroupPerms] = useState<string[]>([]);
  const [newGroupReason, setNewGroupReason] = useState('');

  // Synchronize initial admin
  React.useEffect(() => {
    if (admins.length > 0) {
      if (initialAdminId && admins.some((a) => a.id === initialAdminId)) {
        setSelectedAdminId(initialAdminId);
      } else if (!selectedAdminId) {
        setSelectedAdminId(admins[0].id);
      }
    }
  }, [admins, initialAdminId, selectedAdminId]);

  const selectedAdmin = useMemo(() => {
    return admins.find((a) => a.id === selectedAdminId) || null;
  }, [admins, selectedAdminId]);

  // Working draft of permissions for the selected admin
  const [draftPermissions, setDraftPermissions] = useState<string[]>([]);
  const [lastSyncedAdminId, setLastSyncedAdminId] = useState<string>('');

  React.useEffect(() => {
    if (selectedAdmin && selectedAdmin.id !== lastSyncedAdminId) {
      setDraftPermissions(selectedAdmin.adminPermissions || []);
      setLastSyncedAdminId(selectedAdmin.id);
    }
  }, [selectedAdmin, lastSyncedAdminId]);

  // Broken-down features tracking: which features are currently "broken" into granular items
  const [brokenFeatureIds, setBrokenFeatureIds] = useState<Set<string>>(new Set());

  // Search & category filter
  const [searchQuery, setSearchQuery] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('all');

  // View mode: compact checklist (default) vs 3-column Kanban board
  const [viewMode, setViewMode] = useState<'checklist' | 'kanban'>('checklist');
  const [collapsedFeatureIds, setCollapsedFeatureIds] = useState<Set<string>>(new Set());

  const toggleFeatureCollapsed = (id: string) => {
    setCollapsedFeatureIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  // Drag-and-drop state
  const [draggedItem, setDraggedItem] = useState<{ type: 'feature' | 'permission'; id: string; featureId?: string } | null>(null);
  const [activeDropZone, setActiveDropZone] = useState<string | null>(null);

  // Save Modal
  const [isSaveModalOpen, setIsSaveModalOpen] = useState(false);
  const [saveReason, setSaveReason] = useState('');
  const [toastMessage, setToastMessage] = useState<{ text: string; type: 'success' | 'error' } | null>(null);

  const features = catalogData?.features || [];

  const isDirty = useMemo(() => {
    if (!selectedAdmin) return false;
    const original = new Set(selectedAdmin.adminPermissions || []);
    const current = new Set(draftPermissions);
    if (original.size !== current.size) return true;
    for (const p of original) {
      if (!current.has(p)) return true;
    }
    return false;
  }, [selectedAdmin, draftPermissions]);

  const saveMutation = useMutation({
    mutationFn: ({ adminId, permissions, reason }: { adminId: string; permissions: string[]; reason: string }) =>
      api.put(`/admin/access-control/users/${encodeURIComponent(adminId)}`, { permissions, reason }),
    onSuccess: (res: any) => {
      qc.invalidateQueries({ queryKey: ['admin-access-control-admins'] });
      qc.invalidateQueries({ queryKey: ['admin-user', selectedAdminId] });
      setIsSaveModalOpen(false);
      setSaveReason('');
      setToastMessage({ text: 'Command Center access controls updated and audit log recorded.', type: 'success' });
      setTimeout(() => setToastMessage(null), 5000);
    },
    onError: (err: any) => {
      setToastMessage({ text: err?.message || 'Failed to update access controls.', type: 'error' });
      setTimeout(() => setToastMessage(null), 6000);
    },
  });

  const addUserMutation = useMutation({
    mutationFn: (vars: { email: string; name: string; groupId?: string; reason: string }) =>
      api.post('/admin/access-control/users', vars),
    onSuccess: (newUser: any) => {
      qc.invalidateQueries({ queryKey: ['admin-access-control-admins'] });
      setIsAddUserModalOpen(false);
      setNewUserEmail('');
      setNewUserName('');
      setNewUserReason('');
      if (newUser?.id) setSelectedAdminId(newUser.id);
      setToastMessage({ text: `Operator ${newUser.email} added to Command Center.`, type: 'success' });
      setTimeout(() => setToastMessage(null), 5000);
    },
    onError: (err: any) => {
      setToastMessage({ text: err?.message || 'Failed to add operator.', type: 'error' });
      setTimeout(() => setToastMessage(null), 6000);
    },
  });

  const setUserGroupMutation = useMutation({
    mutationFn: ({ adminId, groupId, reason }: { adminId: string; groupId: string; reason: string }) =>
      api.put(`/admin/access-control/users/${encodeURIComponent(adminId)}/group`, { groupId, reason }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-access-control-admins'] });
      qc.invalidateQueries({ queryKey: ['admin-access-control-groups'] });
      setToastMessage({ text: 'Operator group assigned successfully.', type: 'success' });
      setTimeout(() => setToastMessage(null), 4000);
    },
    onError: (err: any) => {
      setToastMessage({ text: err?.message || 'Failed to update group.', type: 'error' });
      setTimeout(() => setToastMessage(null), 5000);
    },
  });

  const createGroupMutation = useMutation({
    mutationFn: (vars: { name: string; description: string; icon: string; permissions: string[]; reason: string }) =>
      api.post('/admin/access-control/groups', vars),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-access-control-groups'] });
      setIsCreateGroupModalOpen(false);
      setNewGroupName('');
      setNewGroupDesc('');
      setNewGroupPerms([]);
      setNewGroupReason('');
      setToastMessage({ text: 'Access Control Group created successfully.', type: 'success' });
      setTimeout(() => setToastMessage(null), 5000);
    },
    onError: (err: any) => {
      setToastMessage({ text: err?.message || 'Failed to create group.', type: 'error' });
      setTimeout(() => setToastMessage(null), 6000);
    },
  });

  const deleteGroupMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.del(`/admin/access-control/groups/${encodeURIComponent(id)}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-access-control-groups'] });
      qc.invalidateQueries({ queryKey: ['admin-access-control-admins'] });
      setToastMessage({ text: 'Access Control Group deleted.', type: 'success' });
      setTimeout(() => setToastMessage(null), 4000);
    },
    onError: (err: any) => {
      setToastMessage({ text: err?.message || 'Failed to delete group.', type: 'error' });
      setTimeout(() => setToastMessage(null), 5000);
    },
  });

  // Feature breaking helpers
  const toggleBreakFeature = (featureId: string) => {
    if (!isSuperAdmin) return;
    setBrokenFeatureIds((prev) => {
      const next = new Set(prev);
      if (next.has(featureId)) {
        next.delete(featureId);
      } else {
        next.add(featureId);
      }
      return next;
    });
  };

  const breakAllFeatures = () => {
    if (!isSuperAdmin) return;
    setBrokenFeatureIds(new Set(features.map((f) => f.id)));
  };

  const collapseAllFeatures = () => {
    if (!isSuperAdmin) return;
    setBrokenFeatureIds(new Set());
  };

  // Permission mutation helpers with parent/sub-permission coherence
  const grantPermission = (permId: string) => {
    if (!isSuperAdmin) return;
    setDraftPermissions((prev) => {
      const next = prev.includes(permId) ? prev : [...prev, permId];
      // Check if this completes all sub-permissions for a feature; if so, also grant feature.id
      const parentFeature = features.find((f) => f.subPermissions.some((sp) => sp.id === permId));
      if (parentFeature) {
        const allSubsGranted = parentFeature.subPermissions.every((sp) => next.includes(sp.id));
        if (allSubsGranted && !next.includes(parentFeature.id)) {
          return [...next, parentFeature.id];
        }
      }
      return next;
    });
  };

  const revokePermission = (permId: string) => {
    if (!isSuperAdmin) return;
    setDraftPermissions((prev) => {
      // Find parent feature of this sub-permission
      const parentFeature = features.find((f) => f.subPermissions.some((sp) => sp.id === permId));
      // Remove permId AND parent feature ID if present (since not all subpermissions are granted anymore)
      return prev.filter((p) => p !== permId && (!parentFeature || p !== parentFeature.id));
    });
  };

  const grantWholeFeature = (feature: CommandCenterFeature) => {
    if (!isSuperAdmin) return;
    const subIds = feature.subPermissions.map((p) => p.id);
    setDraftPermissions((prev) => Array.from(new Set([...prev, feature.id, ...subIds])));
  };

  const revokeWholeFeature = (feature: CommandCenterFeature) => {
    if (!isSuperAdmin) return;
    const subIds = new Set(feature.subPermissions.map((p) => p.id));
    setDraftPermissions((prev) => prev.filter((p) => p !== feature.id && !subIds.has(p)));
  };

  // Preset Handlers
  const applyPreset = (preset: 'super' | 'ops' | 'content' | 'support' | 'security' | 'auditor' | 'reset') => {
    if (!isSuperAdmin || !features.length) return;
    if (preset === 'reset') {
      setDraftPermissions([]);
      return;
    }

    if (preset === 'super') {
      const all: string[] = [];
      for (const f of features) {
        all.push(f.id);
        for (const p of f.subPermissions) all.push(p.id);
      }
      setDraftPermissions(all);
      return;
    }

    let targetFeatures: string[] = [];
    let targetPermissions: string[] = [];

    if (preset === 'ops') {
      targetFeatures = ['queue', 'vendors', 'categories'];
    } else if (preset === 'content') {
      targetFeatures = ['content', 'local_guide'];
    } else if (preset === 'support') {
      targetFeatures = ['users', 'vendor_logins', 'access_grants'];
    } else if (preset === 'security') {
      targetFeatures = ['access_control', 'audit_log', 'settings'];
    } else if (preset === 'auditor') {
      targetPermissions = features.flatMap((f) =>
        f.subPermissions.filter((p) => p.id.endsWith('.view')).map((p) => p.id)
      );
    }

    if (preset === 'auditor') {
      setDraftPermissions(targetPermissions);
    } else {
      const perms: string[] = [];
      for (const fid of targetFeatures) {
        const feat = features.find((f) => f.id === fid);
        if (feat) {
          perms.push(feat.id);
          for (const sp of feat.subPermissions) perms.push(sp.id);
        }
      }
      setDraftPermissions(perms);
    }
  };

  // Drag and drop handlers
  const handleDragStart = (e: React.DragEvent, item: { type: 'feature' | 'permission'; id: string; featureId?: string }) => {
    if (!isSuperAdmin) return;
    setDraggedItem(item);
    e.dataTransfer.setData('text/plain', JSON.stringify(item));
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleDragOver = (e: React.DragEvent, zone: string) => {
    if (!isSuperAdmin) return;
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

  const handleDrop = (e: React.DragEvent, targetZone: 'available' | 'break' | 'granted') => {
    if (!isSuperAdmin) return;
    e.preventDefault();
    setActiveDropZone(null);
    const raw = e.dataTransfer.getData('text/plain');
    if (!raw) return;

    try {
      const data = JSON.parse(raw);
      if (!data?.id) return;

      if (data.type === 'feature') {
        const feat = features.find((f) => f.id === data.id);
        if (!feat) return;

        if (targetZone === 'granted') {
          grantWholeFeature(feat);
        } else if (targetZone === 'break') {
          // Break feature down into granular To-Do checklist tasks!
          setBrokenFeatureIds((prev) => new Set([...prev, feat.id]));
        } else {
          // targetZone === 'available'
          revokeWholeFeature(feat);
        }
      } else if (data.type === 'permission') {
        if (targetZone === 'granted') {
          grantPermission(data.id);
        } else if (targetZone === 'break') {
          // Return permission to broken down To-Do checklist
          revokePermission(data.id);
          const parentFeature = features.find((f) => f.subPermissions.some((sp) => sp.id === data.id));
          if (parentFeature) {
            setBrokenFeatureIds((prev) => new Set([...prev, parentFeature.id]));
          }
        } else {
          // targetZone === 'available'
          revokePermission(data.id);
        }
      }
    } catch {
      // ignore invalid data
    } finally {
      setDraggedItem(null);
    }
  };

  // Filtered features
  const filteredFeatures = useMemo(() => {
    return features.filter((f) => {
      const matchesCategory = categoryFilter === 'all' || f.category === categoryFilter;
      const q = searchQuery.trim().toLowerCase();
      const matchesSearch =
        !q ||
        f.name.toLowerCase().includes(q) ||
        f.description.toLowerCase().includes(q) ||
        f.id.toLowerCase().includes(q) ||
        f.subPermissions.some((sp) => sp.name.toLowerCase().includes(q) || sp.id.toLowerCase().includes(q));
      return matchesCategory && matchesSearch;
    });
  }, [features, categoryFilter, searchQuery]);

  if (isLoadingCatalog || isLoadingAdmins) {
    return <div className="panel p-10 text-center text-muted font-semibold">Loading Command Center access controls…</div>;
  }

  return (
    <div className="space-y-6">
      {/* Top Header Strip */}
      <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg flex items-center justify-center bg-rose-500/10 text-rose-500 border border-rose-500/20">
              <ShieldCheck size={18} />
            </div>
            <div>
              <h1 className="text-xl font-bold tracking-tight">Command Center User Access Control</h1>
              <p className="text-[12.5px] text-muted font-semibold">
                Super Admin access control breaker and To-Do list allocator for Command Center operators
              </p>
            </div>
          </div>
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-2 self-stretch md:self-auto justify-end">
          <button
            onClick={() => setIsGroupsModalOpen(true)}
            className="btn flex items-center gap-2 py-1.5 px-3 text-[13px] border border-line hover:border-muted"
          >
            <Shield size={14} />
            <span>Role Groups ({groups.length})</span>
          </button>

          {isDirty && (
            <span className="text-[11.5px] font-bold text-amber-500 bg-amber-500/10 border border-amber-500/20 px-2.5 py-1 rounded-full">
              Unsaved Access Changes
            </span>
          )}

          {isSuperAdmin ? (
            <button
              onClick={() => setIsSaveModalOpen(true)}
              disabled={!isDirty || saveMutation.isPending}
              className="btn btn-primary flex items-center gap-2 py-1.5 px-4 text-[13px] disabled:opacity-50"
            >
              <Save size={14} />
              {saveMutation.isPending ? 'Publishing…' : 'Save Access Controls'}
            </button>
          ) : (
            <div className="flex items-center gap-1.5 text-[12px] font-bold text-muted bg-line/20 px-3 py-1.5 rounded-lg border border-line">
              <Lock size={13} /> View Only (Super Admin required to edit)
            </div>
          )}
        </div>
      </div>

      {!isSuperAdmin && (
        <div className="p-3.5 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400 text-[12.5px] font-semibold flex items-center gap-2.5">
          <AlertCircle size={16} className="shrink-0" />
          <span>
            <strong>Read-Only Mode:</strong> You are currently viewing Command Center access controls. Only Super Admins can break down features or alter operator permissions.
          </span>
        </div>
      )}

      {toastMessage && (
        <div
          className={`p-3.5 rounded-xl border text-[13px] font-bold flex items-center justify-between gap-3 ${
            toastMessage.type === 'success'
              ? 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20'
              : 'bg-rose-500/10 text-rose-600 border-rose-500/20'
          }`}
        >
          <span>{toastMessage.text}</span>
          <button onClick={() => setToastMessage(null)} className="opacity-60 hover:opacity-100">
            <X size={14} />
          </button>
        </div>
      )}

      {/* Operator Account Selector Bar */}
      <div className="panel p-4">
        <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-4">
          <div className="flex items-center gap-3 w-full lg:w-auto">
            <span className="text-[11.5px] font-bold uppercase tracking-wider text-muted shrink-0">Operator:</span>
            <div className="flex items-center gap-1.5 overflow-x-auto py-1 max-w-full">
              {admins.map((admin) => {
                const isSelected = admin.id === selectedAdminId;
                const isItemSuper = admin.isSuperAdmin;
                return (
                  <button
                    key={admin.id}
                    onClick={() => {
                      if (isDirty) {
                        if (!confirm('You have unsaved access control changes for this operator. Switch anyway?')) {
                          return;
                        }
                      }
                      setSelectedAdminId(admin.id);
                    }}
                    className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-[12.5px] font-bold transition-all shrink-0 border ${
                      isSelected
                        ? 'bg-[var(--accent)] text-white border-[var(--accent)] shadow-sm'
                        : 'bg-[var(--surface)] text-[var(--ink)] border-line hover:border-muted'
                    }`}
                  >
                    <div className="w-5 h-5 rounded-full flex items-center justify-center text-[10px] bg-black/20 text-white font-mono">
                      {admin.name ? admin.name[0].toUpperCase() : admin.email[0].toUpperCase()}
                    </div>
                    <span>{admin.name || admin.email}</span>
                    {isItemSuper ? (
                      <span className="text-[9px] uppercase px-1.5 py-0.2 rounded bg-amber-400 text-black font-extrabold">
                        Super
                      </span>
                    ) : admin.adminGroupName ? (
                      <span className="text-[9px] font-bold px-1.5 py-0.2 rounded bg-black/10 text-inherit opacity-80">
                        {admin.adminGroupName}
                      </span>
                    ) : null}
                  </button>
                );
              })}
              {isSuperAdmin && (
                <button
                  onClick={() => setIsAddUserModalOpen(true)}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[12px] font-bold text-[var(--accent)] border border-dashed border-[var(--accent)] hover:bg-[var(--accent)]/10 shrink-0"
                >
                  <Users size={13} />
                  <span>+ Add Operator</span>
                </button>
              )}
            </div>
          </div>

          {/* Quick Presets Menu */}
          {isSuperAdmin && (
            <div className="flex items-center gap-1.5 shrink-0 self-end lg:self-auto text-[11.5px] flex-wrap">
              <span className="font-bold text-muted mr-1">Presets:</span>
              <button
                onClick={() => applyPreset('super')}
                className="btn !py-1 !px-2 text-[11.5px] font-semibold"
                title="Grant every Command Center feature and sub-permission"
              >
                All Access
              </button>
              <button
                onClick={() => applyPreset('ops')}
                className="btn !py-1 !px-2 text-[11.5px] font-semibold"
                title="Queue, Vendors, Categories"
              >
                Operations
              </button>
              <button
                onClick={() => applyPreset('content')}
                className="btn !py-1 !px-2 text-[11.5px] font-semibold"
                title="Content Studio and Local Guide AI"
              >
                Content Lead
              </button>
              <button
                onClick={() => applyPreset('support')}
                className="btn !py-1 !px-2 text-[11.5px] font-semibold"
                title="Users, Vendor Logins, Access Grants"
              >
                Support
              </button>
              <button
                onClick={() => applyPreset('security')}
                className="btn !py-1 !px-2 text-[11.5px] font-semibold"
                title="Access Control, Audit Log, Settings"
              >
                Security
              </button>
              <button
                onClick={() => applyPreset('auditor')}
                className="btn !py-1 !px-2 text-[11.5px] font-semibold"
                title="View-only access across all sections"
              >
                Read-Only
              </button>
              <button
                onClick={() => applyPreset('reset')}
                className="btn !py-1 !px-2 text-rose-500 hover:text-rose-600 font-semibold"
                title="Clear all granted permissions"
              >
                Clear
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Selected Admin Overview & Feature Break Toolbar */}
      {selectedAdmin && (
        <div className="panel p-4 bg-line/10 border-line flex flex-col md:flex-row items-start md:items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-slate-700 to-slate-900 text-white flex items-center justify-center font-bold text-sm">
              {selectedAdmin.name ? selectedAdmin.name.slice(0, 2).toUpperCase() : selectedAdmin.email.slice(0, 2).toUpperCase()}
            </div>
            <div>
              <div className="flex items-center gap-2 flex-wrap">
                <h3 className="font-bold text-[14px]">{selectedAdmin.name || selectedAdmin.email}</h3>
                <span className="text-[11px] font-mono text-muted">{selectedAdmin.email}</span>
                {selectedAdmin.isSuperAdmin ? (
                  <span className="px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-amber-500/20 text-amber-500 border border-amber-500/30">
                    Super-Admin (Holds All Powers)
                  </span>
                ) : (
                  <span className="px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-slate-500/15 text-slate-400 border border-slate-500/20">
                    Standard Admin
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2 mt-1 flex-wrap">
                <span className="text-[11px] font-bold text-muted">Role Group:</span>
                <select
                  disabled={!isSuperAdmin || setUserGroupMutation.isPending}
                  value={selectedAdmin.adminGroupId || ''}
                  onChange={(e) => {
                    const newGid = e.target.value;
                    const g = groups.find((x) => x.id === newGid);
                    const reason = prompt(
                      `Reason for changing ${selectedAdmin.email}'s group to ${g ? g.name : 'Custom / None'}?`,
                      'Role update'
                    );
                    if (reason && reason.trim()) {
                      setUserGroupMutation.mutate({
                        adminId: selectedAdmin.id,
                        groupId: newGid,
                        reason: reason.trim(),
                      });
                    }
                  }}
                  className="input !py-0.5 !px-2 text-[11px] font-semibold bg-white border-line rounded"
                >
                  <option value="">No Group (Custom Permissions Only)</option>
                  {groups.map((g) => (
                    <option key={g.id} value={g.id}>
                      {g.name} {g.isSystem ? '· System' : '· Custom'}
                    </option>
                  ))}
                </select>
                <span className="text-[11px] text-muted">·</span>
                <span className="text-[11.5px] text-muted font-medium">
                  Active permissions: <strong className="text-[var(--ink)]">{draftPermissions.length}</strong> ·{' '}
                  {selectedAdmin.activeSessions} device session{selectedAdmin.activeSessions === 1 ? '' : 's'} ·{' '}
                  <Link
                    href={`/users?user=${encodeURIComponent(selectedAdmin.id)}`}
                    className="font-semibold underline hover:text-[var(--accent)]"
                  >
                    View profile →
                  </Link>
                </span>
              </div>
            </div>
          </div>

          {isSuperAdmin ? (
            <div className="flex items-center gap-2 shrink-0">
              <button
                onClick={breakAllFeatures}
                className="btn !py-1.5 !px-3 text-[12px] flex items-center gap-1.5 font-bold"
                title="Break down all features into atomic To-Do tasks"
              >
                <Scissors size={13} className="text-rose-500" />
                Break All Features
              </button>
              <button
                onClick={collapseAllFeatures}
                className="btn !py-1.5 !px-3 text-[12px] flex items-center gap-1.5 font-semibold text-muted"
              >
                <RotateCcw size={13} />
                Collapse All
              </button>
            </div>
          ) : (
            <div className="flex items-center gap-1 text-[11px] text-muted font-semibold bg-line/20 px-2.5 py-1 rounded-lg">
              <Lock size={12} /> Read-only mode
            </div>
          )}
        </div>
      )}

      {/* Filter, Search Bar, and View Mode Switcher */}
      <div className="flex flex-col lg:flex-row items-stretch lg:items-center justify-between gap-3">
        <div className="flex items-center gap-3 flex-1 flex-wrap sm:flex-nowrap">
          <div className="relative w-full sm:w-72">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
            <input
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search features or permissions…"
              className="input w-full pl-9 py-1.5 text-[13px]"
            />
          </div>

          <div className="flex items-center gap-1.5 overflow-x-auto w-full sm:w-auto">
            <button
              onClick={() => setCategoryFilter('all')}
              className={`px-3 py-1 rounded-lg text-[12px] font-bold border transition-colors ${
                categoryFilter === 'all'
                  ? 'bg-[var(--accent)] text-white border-[var(--accent)]'
                  : 'bg-[var(--surface)] text-muted border-line'
              }`}
            >
              All Categories ({features.length})
            </button>
            {Object.entries(CATEGORY_META).map(([catKey, meta]) => (
              <button
                key={catKey}
                onClick={() => setCategoryFilter(catKey)}
                className={`px-2.5 py-1 rounded-lg text-[11.5px] font-bold border transition-colors whitespace-nowrap ${
                  categoryFilter === catKey
                    ? 'bg-[var(--accent)] text-white border-[var(--accent)]'
                    : 'bg-[var(--surface)] text-muted border-line'
                }`}
              >
                {meta.label}
              </button>
            ))}
          </div>
        </div>

        {/* View Mode Switcher */}
        <div className="flex items-center gap-1 bg-line/20 p-1 rounded-xl border border-line shrink-0 self-start lg:self-auto">
          <button
            onClick={() => setViewMode('checklist')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[12px] font-bold transition-all ${
              viewMode === 'checklist'
                ? 'bg-[var(--accent)] text-white shadow-sm'
                : 'text-muted hover:text-[var(--ink)]'
            }`}
          >
            <CheckSquare size={13} />
            <span>Checklist View</span>
          </button>
          <button
            onClick={() => setViewMode('kanban')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[12px] font-bold transition-all ${
              viewMode === 'kanban'
                ? 'bg-[var(--accent)] text-white shadow-sm'
                : 'text-muted hover:text-[var(--ink)]'
            }`}
          >
            <Layers size={13} />
            <span>Kanban Board</span>
          </button>
        </div>
      </div>

      {viewMode === 'checklist' ? (
        <div className="panel p-5 space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between pb-3.5 border-b border-line gap-2">
            <div>
              <h2 className="font-bold text-[15px] flex items-center gap-2">
                <CheckSquare size={16} style={{ color: 'var(--accent)' }} />
                <span>Privilege Checklist</span>
              </h2>
              <p className="text-[12px] text-muted font-medium mt-0.5">
                Each privilege is listed with a check mark directly in front for quick, compact access configuration.
              </p>
            </div>
            <div className="flex items-center gap-2 self-end sm:self-auto text-[12px]">
              <span className="font-semibold text-muted">
                <strong className="text-[var(--ink)]">{draftPermissions.length}</strong> permissions granted
              </span>
              {isSuperAdmin && (
                <>
                  <button
                    type="button"
                    onClick={() => applyPreset('super')}
                    className="text-[11.5px] font-bold text-[var(--accent)] hover:underline ml-2"
                  >
                    Select All
                  </button>
                  <span className="text-muted">·</span>
                  <button
                    type="button"
                    onClick={() => applyPreset('reset')}
                    className="text-[11.5px] font-bold text-rose-500 hover:underline"
                  >
                    Clear All
                  </button>
                </>
              )}
            </div>
          </div>

          {filteredFeatures.length === 0 ? (
            <div className="p-8 text-center text-muted font-semibold text-[13px]">
              No features match your current filter.
            </div>
          ) : (
            <div className="space-y-3">
              {filteredFeatures.map((feat) => {
                const Icon = FEATURE_ICONS[feat.icon] || Shield;
                const catMeta = CATEGORY_META[feat.category] || { label: feat.category, bg: '#eee', text: '#333', border: '#ccc' };
                const grantedCount = feat.subPermissions.filter((sp) => draftPermissions.includes(sp.id)).length;
                const isAllGranted = feat.subPermissions.length > 0 && grantedCount === feat.subPermissions.length;
                const isPartiallyGranted = grantedCount > 0 && !isAllGranted;
                const isCollapsed = collapsedFeatureIds.has(feat.id);

                return (
                  <div
                    key={feat.id}
                    className={`rounded-xl border transition-all overflow-hidden ${
                      isAllGranted
                        ? 'border-emerald-500/30 bg-emerald-500/[0.02]'
                        : isPartiallyGranted
                        ? 'border-amber-500/30 bg-amber-500/[0.02]'
                        : 'border-line bg-[var(--surface)]'
                    }`}
                  >
                    {/* Feature Master Row */}
                    <div className="p-3.5 flex items-start sm:items-center justify-between gap-3 bg-[var(--surface)] border-b border-line/60">
                      <div className="flex items-start sm:items-center gap-3 min-w-0">
                        {/* Check mark directly in front of the feature name */}
                        <button
                          type="button"
                          disabled={!isSuperAdmin}
                          onClick={() => {
                            if (isAllGranted) {
                              revokeWholeFeature(feat);
                            } else {
                              grantWholeFeature(feat);
                            }
                          }}
                          className={`w-6 h-6 rounded-md flex items-center justify-center shrink-0 border transition-all mt-0.5 sm:mt-0 ${
                            isAllGranted
                              ? 'bg-emerald-500 border-emerald-600 text-white shadow-sm'
                              : isPartiallyGranted
                              ? 'bg-amber-500 border-amber-600 text-white shadow-sm'
                              : 'border-line hover:border-muted bg-transparent text-transparent hover:text-muted/30'
                          } ${!isSuperAdmin ? 'cursor-default opacity-80' : 'cursor-pointer hover:scale-105'}`}
                          title={
                            isAllGranted
                              ? 'Click to revoke all permissions in this feature'
                              : 'Click to grant all permissions in this feature'
                          }
                        >
                          {isAllGranted && <Check size={14} className="stroke-[3]" />}
                          {isPartiallyGranted && <Minus size={14} className="stroke-[3]" />}
                        </button>

                        <div
                          className="w-7 h-7 rounded-lg flex items-center justify-center shrink-0 hidden sm:flex"
                          style={{ background: catMeta.bg, color: catMeta.text }}
                        >
                          <Icon size={14} />
                        </div>

                        <div className="min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="font-bold text-[13.5px] leading-snug">{feat.name}</span>
                            <span
                              className="text-[9.5px] font-bold px-1.5 py-0.2 rounded uppercase border"
                              style={{ background: catMeta.bg, color: catMeta.text, borderColor: catMeta.border }}
                            >
                              {catMeta.label}
                            </span>
                          </div>
                          <p className="text-[11.5px] text-muted font-medium mt-0.5 line-clamp-1 sm:line-clamp-none">
                            {feat.description}
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center gap-2 shrink-0">
                        <span
                          className={`text-[11px] font-bold px-2 py-0.5 rounded-full ${
                            isAllGranted
                              ? 'bg-emerald-500/15 text-emerald-600'
                              : isPartiallyGranted
                              ? 'bg-amber-500/15 text-amber-600'
                              : 'bg-line text-muted'
                          }`}
                        >
                          {grantedCount} / {feat.subPermissions.length}
                        </span>
                        <button
                          type="button"
                          onClick={() => toggleFeatureCollapsed(feat.id)}
                          className="p-1 rounded text-muted hover:text-[var(--ink)]"
                          title={isCollapsed ? 'Expand permissions' : 'Collapse permissions'}
                        >
                          {isCollapsed ? <ChevronRight size={16} /> : <ChevronDown size={16} />}
                        </button>
                      </div>
                    </div>

                    {/* Granular Sub-permissions Checklist */}
                    {!isCollapsed && (
                      <div className="divide-y divide-line/40 bg-[var(--surface-subtle,rgba(0,0,0,0.01))]">
                        {feat.subPermissions.map((sp) => {
                          const isGranted = draftPermissions.includes(sp.id);
                          const risk = RISK_META[sp.risk] || RISK_META.low;

                          return (
                            <div
                              key={sp.id}
                              className={`px-4 sm:px-6 py-2.5 flex items-start sm:items-center justify-between gap-3 hover:bg-[var(--line)]/15 transition-colors ${
                                isGranted ? 'bg-emerald-500/[0.015]' : ''
                              }`}
                            >
                              <div className="flex items-start sm:items-center gap-3 min-w-0">
                                {/* Sub-permission Checkmark Button */}
                                <button
                                  type="button"
                                  disabled={!isSuperAdmin}
                                  onClick={() => {
                                    if (isGranted) {
                                      revokePermission(sp.id);
                                    } else {
                                      grantPermission(sp.id);
                                    }
                                  }}
                                  className={`w-5 h-5 rounded flex items-center justify-center shrink-0 border transition-all mt-0.5 sm:mt-0 ${
                                    isGranted
                                      ? 'bg-emerald-500 border-emerald-600 text-white shadow-sm'
                                      : 'border-line hover:border-muted bg-transparent text-transparent hover:text-muted/30'
                                  } ${!isSuperAdmin ? 'cursor-default opacity-80' : 'cursor-pointer hover:scale-105'}`}
                                  title={isGranted ? 'Click to revoke' : 'Click to grant'}
                                >
                                  {isGranted && <Check size={12} className="stroke-[3]" />}
                                </button>

                                <div className="min-w-0">
                                  <div className="flex items-center gap-2 flex-wrap">
                                    <span
                                      className={`text-[12.5px] ${
                                        isGranted ? 'font-bold text-[var(--ink)]' : 'font-medium text-muted'
                                      }`}
                                    >
                                      {sp.name}
                                    </span>
                                    <span
                                      className="text-[9px] font-bold px-1.5 py-0.2 rounded uppercase"
                                      style={{ background: risk.bg, color: risk.text }}
                                    >
                                      {risk.label}
                                    </span>
                                  </div>
                                  <p className="text-[11px] text-muted font-normal mt-0.5">
                                    {sp.description}
                                  </p>
                                </div>
                              </div>

                              <span className="text-[10.5px] font-mono text-muted/70 shrink-0 hidden md:inline">
                                {sp.id}
                              </span>
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      ) : (
        /* To-Do List Style 3-Column Kanban Board with Bidirectional Drag and Drop */
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-5 items-start">
        {/* Column 1: Feature Catalog & Available Pool (The Feature Catalog) */}
        <div
          onDragOver={(e) => handleDragOver(e, 'available')}
          onDragLeave={(e) => handleDragLeave(e, 'available')}
          onDrop={(e) => handleDrop(e, 'available')}
          className={`panel p-4 flex flex-col transition-all min-h-[620px] ${
            activeDropZone === 'available' ? 'ring-2 ring-[var(--accent)] bg-[var(--accent-tint)]/10' : ''
          }`}
        >
          <div className="flex items-center justify-between pb-3 mb-3 border-b border-line">
            <div className="flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full bg-slate-400" />
              <h2 className="font-bold text-[14px]">1. Feature Catalog & Pool</h2>
            </div>
            <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-line font-bold text-muted">
              {filteredFeatures.length} features
            </span>
          </div>
          <p className="text-[11.5px] text-muted font-medium mb-3">
            All Command Center features listed first. Click <strong className="text-[var(--ink)]">Break</strong> or drag into Column 2 to break into To-Do tasks.
          </p>

          <div className="space-y-3 flex-1 overflow-y-auto max-h-[750px] pr-1">
            {filteredFeatures.map((feat) => {
              const Icon = FEATURE_ICONS[feat.icon] || Shield;
              const isBroken = brokenFeatureIds.has(feat.id);
              const catMeta = CATEGORY_META[feat.category] || { label: feat.category, bg: '#eee', text: '#333', border: '#ccc' };
              const grantedCount = feat.subPermissions.filter((sp) => draftPermissions.includes(sp.id)).length;
              const isAllGranted = grantedCount === feat.subPermissions.length;

              return (
                <div
                  key={feat.id}
                  draggable={isSuperAdmin}
                  onDragStart={(e) => handleDragStart(e, { type: 'feature', id: feat.id })}
                  className={`p-3.5 rounded-xl border transition-all select-none ${
                    isSuperAdmin ? 'cursor-grab active:cursor-grabbing hover:shadow-md' : 'cursor-default'
                  } ${
                    isAllGranted
                      ? 'bg-emerald-500/5 border-emerald-500/25'
                      : 'bg-[var(--surface)] border-line hover:border-muted'
                  }`}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-start gap-2.5 min-w-0">
                      {isSuperAdmin && (
                        <GripVertical size={14} className="text-muted/60 hover:text-muted shrink-0 mt-0.5" />
                      )}
                      <div
                        className="w-7 h-7 rounded-lg flex items-center justify-center shrink-0"
                        style={{ background: catMeta.bg, color: catMeta.text }}
                      >
                        <Icon size={14} />
                      </div>
                      <div className="min-w-0">
                        <div className="flex items-center gap-1.5 flex-wrap">
                          <h4 className="font-bold text-[13px] leading-snug">{feat.name}</h4>
                          <span
                            className="text-[9.5px] font-bold px-1.5 py-0.2 rounded uppercase border"
                            style={{ background: catMeta.bg, color: catMeta.text, borderColor: catMeta.border }}
                          >
                            {catMeta.label}
                          </span>
                        </div>
                        <p className="text-[11.5px] text-muted font-medium mt-1 leading-relaxed">
                          {feat.description}
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Feature Footer & Actions */}
                  <div className="mt-3 pt-2.5 border-t border-line/60 flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      {isSuperAdmin ? (
                        <button
                          onClick={() => toggleBreakFeature(feat.id)}
                          className={`text-[11px] font-bold px-2.5 py-1 rounded-lg flex items-center gap-1 transition-colors border ${
                            isBroken
                              ? 'bg-rose-500/10 text-rose-500 border-rose-500/20'
                              : 'bg-line/40 hover:bg-line text-[var(--ink)] border-line'
                          }`}
                          title={isBroken ? 'Collapse sub-permissions back into feature' : 'Break down this feature into granular To-Do items (or drag to column 2)'}
                        >
                          <Scissors size={12} className={isBroken ? 'rotate-90 text-rose-500' : ''} />
                          {isBroken ? 'Collapse' : `Break (${feat.subPermissions.length} tasks)`}
                        </button>
                      ) : (
                        <span className="text-[11px] font-bold text-muted px-2 py-0.5 rounded bg-line/40">
                          {feat.subPermissions.length} tasks
                        </span>
                      )}

                      <span className="text-[10.5px] text-muted font-semibold">
                        {grantedCount}/{feat.subPermissions.length} granted
                      </span>
                    </div>

                    {isSuperAdmin && (
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => (isAllGranted ? revokeWholeFeature(feat) : grantWholeFeature(feat))}
                          className={`btn !py-1 !px-2 text-[11px] font-bold flex items-center gap-1 ${
                            isAllGranted ? 'text-rose-500 hover:text-rose-600' : 'btn-primary'
                          }`}
                        >
                          {isAllGranted ? 'Revoke All' : 'Grant All →'}
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Column 2: Broken Down Granular Tasks (To-Do Checklist & Break Target) */}
        <div
          onDragOver={(e) => handleDragOver(e, 'break')}
          onDragLeave={(e) => handleDragLeave(e, 'break')}
          onDrop={(e) => handleDrop(e, 'break')}
          className={`panel p-4 flex flex-col min-h-[620px] transition-all border-dashed border-2 ${
            activeDropZone === 'break'
              ? 'ring-2 ring-amber-500 bg-amber-500/10 border-amber-500'
              : 'bg-slate-500/5 border-line'
          }`}
        >
          <div className="flex items-center justify-between pb-3 mb-3 border-b border-line">
            <div className="flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full bg-amber-500" />
              <h2 className="font-bold text-[14px]">2. Broken Down Granular Tasks</h2>
            </div>
            <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-line font-bold text-muted">
              {brokenFeatureIds.size} broken features
            </span>
          </div>

          <p className="text-[11.5px] text-muted font-medium mb-3">
            To-Do style task checklist. Drag features here to break them down, check items off, or drag tasks to <strong className="text-[var(--ink)]">Granted Controls</strong>.
          </p>

          {brokenFeatureIds.size === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center p-8 text-center border border-line rounded-xl bg-[var(--surface)] text-muted">
              <Scissors size={28} className="text-muted/40 mb-2" />
              <p className="text-[13px] font-bold">No features broken down yet</p>
              <p className="text-[11.5px] text-muted font-medium mt-1 max-w-[240px]">
                Drag any feature card here or click &ldquo;Break&rdquo; in Column 1 to split into atomic To-Do checklist tasks.
              </p>
              {isSuperAdmin && (
                <button
                  onClick={breakAllFeatures}
                  className="btn btn-primary !py-1.5 !px-3 text-[12px] mt-4 flex items-center gap-1.5"
                >
                  <Scissors size={13} />
                  Break All Features Now
                </button>
              )}
            </div>
          ) : (
            <div className="space-y-4 flex-1 overflow-y-auto max-h-[750px] pr-1">
              {features
                .filter((f) => brokenFeatureIds.has(f.id))
                .map((feat) => {
                  const Icon = FEATURE_ICONS[feat.icon] || Shield;
                  return (
                    <div key={feat.id} className="p-3 rounded-xl bg-[var(--surface)] border border-line space-y-2">
                      <div className="flex items-center justify-between pb-2 border-b border-line/50">
                        <div className="flex items-center gap-2">
                          <Icon size={14} className="text-muted" />
                          <h4 className="font-bold text-[12.5px]">{feat.name}</h4>
                        </div>
                        {isSuperAdmin && (
                          <button
                            onClick={() => toggleBreakFeature(feat.id)}
                            className="text-[10px] font-bold text-muted hover:text-[var(--ink)] underline"
                          >
                            Collapse
                          </button>
                        )}
                      </div>

                      {/* Granular Sub-Permission Tasks Checklist */}
                      <div className="space-y-1.5 pt-1">
                        {feat.subPermissions.map((sp) => {
                          const isGranted = draftPermissions.includes(sp.id);
                          const riskMeta = RISK_META[sp.risk] || RISK_META.low;

                          return (
                            <div
                              key={sp.id}
                              draggable={isSuperAdmin}
                              onDragStart={(e) => handleDragStart(e, { type: 'permission', id: sp.id, featureId: feat.id })}
                              className={`p-2.5 rounded-lg border transition-all flex items-start justify-between gap-2 text-left select-none ${
                                isSuperAdmin ? 'cursor-grab active:cursor-grabbing hover:shadow-xs' : 'cursor-default'
                              } ${
                                isGranted
                                  ? 'bg-emerald-500/10 border-emerald-500/30'
                                  : 'bg-line/20 border-line hover:border-muted'
                              }`}
                            >
                              <div className="flex items-start gap-2 min-w-0">
                                {isSuperAdmin && (
                                  <GripVertical size={13} className="text-muted/60 hover:text-muted shrink-0 mt-0.5" />
                                )}
                                {isSuperAdmin ? (
                                  <button
                                    onClick={() => (isGranted ? revokePermission(sp.id) : grantPermission(sp.id))}
                                    className="mt-0.5 shrink-0 text-muted hover:text-[var(--ink)]"
                                    title={isGranted ? 'Click to revoke task' : 'Click to grant task'}
                                  >
                                    {isGranted ? (
                                      <CheckSquare size={15} className="text-emerald-500" />
                                    ) : (
                                      <Square size={15} className="text-muted" />
                                    )}
                                  </button>
                                ) : (
                                  <div className="mt-0.5 shrink-0">
                                    {isGranted ? (
                                      <CheckSquare size={15} className="text-emerald-500" />
                                    ) : (
                                      <Square size={15} className="text-muted" />
                                    )}
                                  </div>
                                )}
                                <div className="min-w-0">
                                  <div className="flex items-center gap-1.5 flex-wrap">
                                    <span className={`font-bold text-[12px] leading-tight ${isGranted ? 'text-emerald-700 dark:text-emerald-400' : ''}`}>
                                      {sp.name}
                                    </span>
                                    <span
                                      className="text-[9px] font-bold px-1.5 py-0.2 rounded"
                                      style={{ background: riskMeta.bg, color: riskMeta.text }}
                                    >
                                      {riskMeta.label}
                                    </span>
                                    {isGranted && (
                                      <span className="text-[9px] font-bold px-1 py-0.2 rounded bg-emerald-500/20 text-emerald-600">
                                        Granted
                                      </span>
                                    )}
                                  </div>
                                  <p className="text-[11px] text-muted font-medium mt-0.5">{sp.description}</p>
                                  <span className="text-[9.5px] font-mono text-muted/80">{sp.id}</span>
                                </div>
                              </div>

                              {isSuperAdmin && (
                                <button
                                  onClick={() => (isGranted ? revokePermission(sp.id) : grantPermission(sp.id))}
                                  className={`btn !py-0.5 !px-1.5 text-[10.5px] font-bold shrink-0 ${
                                    isGranted ? 'text-rose-500 hover:text-rose-600' : 'btn-primary'
                                  }`}
                                >
                                  {isGranted ? 'Revoke' : 'Grant →'}
                                </button>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  );
                })}
            </div>
          )}
        </div>

        {/* Column 3: Active / Granted Access Controls (Drop Target & Draggable Outward) */}
        <div
          onDragOver={(e) => handleDragOver(e, 'granted')}
          onDragLeave={(e) => handleDragLeave(e, 'granted')}
          onDrop={(e) => handleDrop(e, 'granted')}
          className={`panel p-4 flex flex-col transition-all min-h-[620px] ${
            activeDropZone === 'granted' ? 'ring-2 ring-emerald-500 bg-emerald-500/10' : ''
          }`}
        >
          <div className="flex items-center justify-between pb-3 mb-3 border-b border-line">
            <div className="flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full bg-emerald-500" />
              <h2 className="font-bold text-[14px]">3. Granted Access Controls</h2>
            </div>
            <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-600 font-bold">
              {draftPermissions.length} active
            </span>
          </div>

          <p className="text-[11.5px] text-muted font-medium mb-3">
            Active powers for <strong className="text-[var(--ink)]">{selectedAdmin?.name || selectedAdmin?.email}</strong>. Drag items here to grant; drag out to revoke.
          </p>

          {draftPermissions.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center p-8 text-center border border-line rounded-xl bg-[var(--surface)] text-muted">
              <ShieldAlert size={28} className="text-muted/40 mb-2" />
              <p className="text-[13px] font-bold">No access controls granted</p>
              <p className="text-[11.5px] text-muted font-medium mt-1 max-w-[240px]">
                This operator has zero extra Command Center privileges. Drag features or tasks here to grant access.
              </p>
            </div>
          ) : (
            <div className="space-y-2.5 flex-1 overflow-y-auto max-h-[750px] pr-1">
              {/* Group granted permissions by feature for neat review */}
              {features.map((feat) => {
                const grantedSub = feat.subPermissions.filter((sp) => draftPermissions.includes(sp.id));
                const isCoarseOnly = draftPermissions.includes(feat.id) && grantedSub.length === 0;
                if (grantedSub.length === 0 && !draftPermissions.includes(feat.id)) return null;

                const Icon = FEATURE_ICONS[feat.icon] || Shield;
                const catMeta = CATEGORY_META[feat.category] || { label: feat.category, bg: '#eee', text: '#333', border: '#ccc' };

                return (
                  <div
                    key={feat.id}
                    draggable={isSuperAdmin}
                    onDragStart={(e) => handleDragStart(e, { type: 'feature', id: feat.id })}
                    className={`p-3 rounded-xl border border-emerald-500/30 bg-emerald-500/5 space-y-2 select-none ${
                      isSuperAdmin ? 'cursor-grab active:cursor-grabbing hover:shadow-xs' : 'cursor-default'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2 min-w-0">
                        {isSuperAdmin && (
                          <GripVertical size={13} className="text-muted/60 hover:text-muted shrink-0" />
                        )}
                        <div
                          className="w-6 h-6 rounded-md flex items-center justify-center shrink-0"
                          style={{ background: catMeta.bg, color: catMeta.text }}
                        >
                          <Icon size={12} />
                        </div>
                        <h4 className="font-bold text-[12.5px] truncate">{feat.name}</h4>
                      </div>

                      {isSuperAdmin && (
                        <button
                          onClick={() => revokeWholeFeature(feat)}
                          className="text-[11px] font-bold text-rose-500 hover:text-rose-600 underline shrink-0"
                        >
                          Revoke All
                        </button>
                      )}
                    </div>

                    {/* Granted Sub-items Pills */}
                    <div className="space-y-1">
                      {isCoarseOnly && (
                        <div className="flex items-center justify-between gap-2 px-2.5 py-1.5 rounded bg-[var(--surface)] border border-amber-500/30 text-[11.5px]">
                          <div className="flex items-center gap-1.5 min-w-0">
                            <Check size={12} className="text-emerald-500 shrink-0" />
                            <span className="font-bold truncate text-amber-600">Full Feature Access ({feat.id})</span>
                          </div>
                          {isSuperAdmin && (
                            <button
                              onClick={() => grantWholeFeature(feat)}
                              className="text-[10.5px] font-bold text-[var(--accent)] underline shrink-0"
                            >
                              Expand Tasks
                            </button>
                          )}
                        </div>
                      )}

                      {grantedSub.map((sp) => (
                        <div
                          key={sp.id}
                          draggable={isSuperAdmin}
                          onDragStart={(e) => handleDragStart(e, { type: 'permission', id: sp.id, featureId: feat.id })}
                          className={`flex items-center justify-between gap-2 px-2.5 py-1.5 rounded bg-[var(--surface)] border border-line text-[11.5px] transition-all select-none ${
                            isSuperAdmin ? 'cursor-grab active:cursor-grabbing hover:border-emerald-500/50 hover:shadow-xs' : 'cursor-default'
                          }`}
                        >
                          <div className="flex items-center gap-1.5 min-w-0">
                            {isSuperAdmin && (
                              <GripVertical size={13} className="text-muted/60 hover:text-muted shrink-0" />
                            )}
                            <Check size={12} className="text-emerald-500 shrink-0" />
                            <span className="font-bold truncate">{sp.name}</span>
                          </div>
                          {isSuperAdmin && (
                            <button
                              onClick={() => revokePermission(sp.id)}
                              className="text-muted hover:text-rose-500 shrink-0 p-0.5 rounded hover:bg-rose-500/10"
                              title="Click or drag out to revoke this control"
                            >
                              <X size={12} />
                            </button>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
      )}

      {/* Save Confirmation Modal with Audit Log Reason */}
      {isSaveModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
          <div className="panel max-w-md w-full p-6 shadow-2xl border-line animate-in fade-in zoom-in-95">
            <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
              <div className="flex items-center gap-2">
                <ShieldCheck size={18} className="text-[var(--accent)]" />
                <h3 className="font-bold text-[15px]">Confirm Access Controls Update</h3>
              </div>
              <button onClick={() => setIsSaveModalOpen(false)} className="text-muted hover:text-[var(--ink)]">
                <X size={16} />
              </button>
            </div>

            <p className="text-[12.5px] text-muted font-medium mb-3">
              You are about to modify Command Center access controls for operator{' '}
              <strong className="text-[var(--ink)]">{selectedAdmin?.name || selectedAdmin?.email}</strong>.
            </p>

            <div className="p-3 rounded-xl bg-line/20 border border-line mb-4 text-[12px] font-semibold space-y-1">
              <div>Total permissions allocated: <strong className="text-emerald-600">{draftPermissions.length}</strong></div>
              <div>Target Operator ID: <span className="font-mono text-muted">{selectedAdmin?.id}</span></div>
            </div>

            <form
              onSubmit={(e) => {
                e.preventDefault();
                if (!selectedAdminId || !saveReason.trim()) return;
                saveMutation.mutate({
                  adminId: selectedAdminId,
                  permissions: draftPermissions,
                  reason: saveReason.trim(),
                });
              }}
              className="space-y-4"
            >
              <label className="block">
                <span className="text-[12px] font-bold block mb-1">
                  Reason for access change <span className="text-rose-500">*</span>
                </span>
                <input
                  autoFocus
                  required
                  value={saveReason}
                  onChange={(e) => setSaveReason(e.target.value)}
                  placeholder="e.g. Assigned Operations Lead privileges for merchant onboarding"
                  className="input w-full py-2 text-[13px]"
                />
                <span className="text-[11px] text-muted mt-1 block">
                  This reason is permanently recorded in the immutable Security Audit Log.
                </span>
              </label>

              <div className="flex items-center justify-end gap-2 pt-2 border-t border-line">
                <button
                  type="button"
                  onClick={() => setIsSaveModalOpen(false)}
                  className="btn py-1.5 px-3 text-[12.5px]"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={saveMutation.isPending || !saveReason.trim()}
                  className="btn btn-primary py-1.5 px-4 text-[12.5px] flex items-center gap-1.5"
                >
                  <Save size={13} />
                  {saveMutation.isPending ? 'Saving to Audit Log…' : 'Confirm & Save'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add Operator Modal (No .env needed) */}
      {isAddUserModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
          <div className="panel w-full max-w-md p-6 bg-[var(--surface)] shadow-2xl border border-line">
            <div className="flex items-start justify-between gap-3 mb-4">
              <div>
                <h3 className="font-bold text-lg">Add Operator to Command Center</h3>
                <p className="text-[12px] text-muted mt-0.5">
                  Grant Command Center access to a new or existing user without modifying environment variables.
                </p>
              </div>
              <button
                onClick={() => setIsAddUserModalOpen(false)}
                className="text-muted hover:text-[var(--ink)] p-1 rounded-lg border border-transparent hover:border-line"
              >
                <X size={16} />
              </button>
            </div>

            <form
              onSubmit={(e) => {
                e.preventDefault();
                if (!newUserEmail.trim() || !newUserReason.trim()) return;
                addUserMutation.mutate({
                  email: newUserEmail.trim(),
                  name: newUserName.trim(),
                  groupId: newUserGroupId || undefined,
                  reason: newUserReason.trim(),
                });
              }}
              className="space-y-3"
            >
              <label className="block">
                <span className="text-[12px] font-bold block mb-1">
                  Operator Email <span className="text-rose-500">*</span>
                </span>
                <input
                  type="email"
                  required
                  autoFocus
                  value={newUserEmail}
                  onChange={(e) => setNewUserEmail(e.target.value)}
                  placeholder="e.g. reviewer@spotto.com"
                  className="input w-full py-2 text-[13px]"
                />
              </label>

              <label className="block">
                <span className="text-[12px] font-bold block mb-1">Full Name (optional)</span>
                <input
                  value={newUserName}
                  onChange={(e) => setNewUserName(e.target.value)}
                  placeholder="e.g. Kasun Fernando"
                  className="input w-full py-2 text-[13px]"
                />
              </label>

              <label className="block">
                <span className="text-[12px] font-bold block mb-1">Role / Access Control Group</span>
                <select
                  value={newUserGroupId}
                  onChange={(e) => setNewUserGroupId(e.target.value)}
                  className="input w-full py-2 text-[13px] bg-white"
                >
                  {groups.map((g) => (
                    <option key={g.id} value={g.id}>
                      {g.name} {g.isSystem ? '· System' : '· Custom'}
                    </option>
                  ))}
                </select>
                <span className="text-[11px] text-muted mt-1 block">
                  The user will immediately inherit all permissions configured for this group.
                </span>
              </label>

              <label className="block">
                <span className="text-[12px] font-bold block mb-1">
                  Reason for Audit Log <span className="text-rose-500">*</span>
                </span>
                <input
                  required
                  value={newUserReason}
                  onChange={(e) => setNewUserReason(e.target.value)}
                  placeholder="e.g. Joined Sri Lanka merchant onboarding team"
                  className="input w-full py-2 text-[13px]"
                />
              </label>

              <div className="flex items-center justify-end gap-2 pt-3 border-t border-line">
                <button
                  type="button"
                  onClick={() => setIsAddUserModalOpen(false)}
                  className="btn py-1.5 px-3 text-[12.5px]"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={addUserMutation.isPending || !newUserEmail.trim() || !newUserReason.trim()}
                  className="btn btn-primary py-1.5 px-4 text-[12.5px] flex items-center gap-1.5"
                >
                  <UserCheck size={13} />
                  {addUserMutation.isPending ? 'Adding Operator…' : 'Add Operator'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Role Groups Modal */}
      {isGroupsModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
          <div className="panel w-full max-w-2xl max-h-[85vh] flex flex-col p-6 bg-[var(--surface)] shadow-2xl border border-line">
            <div className="flex items-start justify-between gap-3 mb-4 shrink-0">
              <div>
                <h3 className="font-bold text-lg">Command Center Role Groups</h3>
                <p className="text-[12px] text-muted mt-0.5">
                  Organize team permissions into reusable operational roles.
                </p>
              </div>
              <div className="flex items-center gap-2">
                {isSuperAdmin && (
                  <button
                    onClick={() => {
                      setIsGroupsModalOpen(false);
                      setIsCreateGroupModalOpen(true);
                    }}
                    className="btn btn-primary text-[12px] py-1 px-3 flex items-center gap-1.5"
                  >
                    <span>+ New Group</span>
                  </button>
                )}
                <button
                  onClick={() => setIsGroupsModalOpen(false)}
                  className="text-muted hover:text-[var(--ink)] p-1 rounded-lg border border-transparent hover:border-line"
                >
                  <X size={16} />
                </button>
              </div>
            </div>

            <div className="overflow-y-auto space-y-3 pr-1 flex-1">
              {groups.map((g) => (
                <div key={g.id} className="p-4 rounded-xl border border-line bg-white/50 hover:bg-white transition-all space-y-2">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <div className="w-7 h-7 rounded-lg bg-[var(--accent)]/10 text-[var(--accent)] flex items-center justify-center">
                        <Shield size={15} />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <h4 className="font-bold text-[14px]">{g.name}</h4>
                          {g.isSystem ? (
                            <span className="text-[9.5px] uppercase font-bold tracking-wider px-1.5 py-0.2 rounded bg-blue-500/10 text-blue-600 border border-blue-500/20">
                              System
                            </span>
                          ) : (
                            <span className="text-[9.5px] uppercase font-bold tracking-wider px-1.5 py-0.2 rounded bg-purple-500/10 text-purple-600 border border-purple-500/20">
                              Custom
                            </span>
                          )}
                        </div>
                        <p className="text-[12px] text-muted font-medium">{g.description}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <span className="text-[11.5px] font-bold text-muted bg-line/30 px-2 py-0.5 rounded-full">
                        {g.memberCount ?? 0} {g.memberCount === 1 ? 'member' : 'members'}
                      </span>
                      {!g.isSystem && isSuperAdmin && (
                        <button
                          onClick={() => {
                            const reason = prompt(`Reason for deleting group ${g.name}?`, 'Obsolete role');
                            if (reason && reason.trim()) {
                              deleteGroupMutation.mutate({ id: g.id, reason: reason.trim() });
                            }
                          }}
                          className="text-rose-500 hover:text-rose-700 p-1 rounded hover:bg-rose-50"
                          title="Delete custom group"
                        >
                          <Trash2 size={14} />
                        </button>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center gap-1.5 flex-wrap pt-1 border-t border-line/40">
                    <span className="text-[10.5px] font-bold text-muted uppercase tracking-wider">Granted:</span>
                    {g.permissions.includes('*') ? (
                      <span className="text-[10.5px] font-bold px-2 py-0.5 rounded-full bg-amber-500/15 text-amber-700">
                        All Features (*)
                      </span>
                    ) : (
                      g.permissions.map((p) => (
                        <span key={p} className="text-[10.5px] font-mono px-2 py-0.5 rounded bg-line/20 text-muted">
                          {p}
                        </span>
                      ))
                    )}
                  </div>
                </div>
              ))}
            </div>

            <div className="flex justify-end pt-3 border-t border-line mt-4">
              <button
                onClick={() => setIsGroupsModalOpen(false)}
                className="btn py-1.5 px-4 text-[12.5px]"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create Custom Group Modal */}
      {isCreateGroupModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
          <div className="panel w-full max-w-xl max-h-[90vh] flex flex-col p-6 bg-[var(--surface)] shadow-2xl border border-line">
            <div className="flex items-start justify-between gap-3 mb-4 shrink-0">
              <div>
                <h3 className="font-bold text-lg">Create Access Control Group</h3>
                <p className="text-[12px] text-muted mt-0.5">
                  Define a new team role with bundled permissions.
                </p>
              </div>
              <button
                onClick={() => setIsCreateGroupModalOpen(false)}
                className="text-muted hover:text-[var(--ink)] p-1 rounded-lg border border-transparent hover:border-line"
              >
                <X size={16} />
              </button>
            </div>

            <form
              onSubmit={(e) => {
                e.preventDefault();
                if (!newGroupName.trim() || !newGroupReason.trim()) return;
                createGroupMutation.mutate({
                  name: newGroupName.trim(),
                  description: newGroupDesc.trim(),
                  icon: newGroupIcon || 'Shield',
                  permissions: newGroupPerms,
                  reason: newGroupReason.trim(),
                });
              }}
              className="space-y-4 overflow-y-auto pr-1 flex-1"
            >
              <label className="block">
                <span className="text-[12px] font-bold block mb-1">
                  Group Name <span className="text-rose-500">*</span>
                </span>
                <input
                  required
                  autoFocus
                  value={newGroupName}
                  onChange={(e) => setNewGroupName(e.target.value)}
                  placeholder="e.g. Southern Coast Reviewers"
                  className="input w-full py-2 text-[13px]"
                />
              </label>

              <label className="block">
                <span className="text-[12px] font-bold block mb-1">Description</span>
                <input
                  value={newGroupDesc}
                  onChange={(e) => setNewGroupDesc(e.target.value)}
                  placeholder="e.g. Regional team managing Galle, Weligama and Mirissa merchants"
                  className="input w-full py-2 text-[13px]"
                />
              </label>

              <div>
                <span className="text-[12px] font-bold block mb-1">Select Granted Features / Permissions</span>
                <div className="space-y-2 max-h-48 overflow-y-auto p-2 bg-line/10 rounded-xl border border-line">
                  {features.map((f) => {
                    const isFeatureSelected = newGroupPerms.includes(f.id);
                    return (
                      <div key={f.id} className="p-2 rounded-lg bg-white/60 border border-line/60">
                        <label className="flex items-center gap-2 cursor-pointer font-bold text-[12.5px]">
                          <input
                            type="checkbox"
                            checked={isFeatureSelected}
                            onChange={(e) => {
                              const checked = e.target.checked;
                              const subIds = f.subPermissions.map((p) => p.id);
                              if (checked) {
                                setNewGroupPerms((prev) => Array.from(new Set([...prev, f.id, ...subIds])));
                              } else {
                                setNewGroupPerms((prev) => prev.filter((p) => p !== f.id && !subIds.includes(p)));
                              }
                            }}
                          />
                          <span>{f.name}</span>
                        </label>
                        <div className="flex items-center gap-1 flex-wrap mt-1.5 ml-5">
                          {f.subPermissions.map((sp) => {
                            const isPermChecked = newGroupPerms.includes(sp.id);
                            return (
                              <button
                                key={sp.id}
                                type="button"
                                onClick={() => {
                                  if (isPermChecked) {
                                    setNewGroupPerms((prev) => prev.filter((p) => p !== sp.id));
                                  } else {
                                    setNewGroupPerms((prev) => [...prev, sp.id]);
                                  }
                                }}
                                className={`text-[11px] px-2 py-0.5 rounded border transition-all ${
                                  isPermChecked
                                    ? 'bg-[var(--accent)] text-white border-[var(--accent)]'
                                    : 'bg-white text-muted border-line hover:border-muted'
                                }`}
                              >
                                {sp.name}
                              </button>
                            );
                          })}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              <label className="block">
                <span className="text-[12px] font-bold block mb-1">
                  Reason for Audit Log <span className="text-rose-500">*</span>
                </span>
                <input
                  required
                  value={newGroupReason}
                  onChange={(e) => setNewGroupReason(e.target.value)}
                  placeholder="e.g. Setting up regional operations role"
                  className="input w-full py-2 text-[13px]"
                />
              </label>

              <div className="flex items-center justify-end gap-2 pt-3 border-t border-line">
                <button
                  type="button"
                  onClick={() => setIsCreateGroupModalOpen(false)}
                  className="btn py-1.5 px-3 text-[12.5px]"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createGroupMutation.isPending || !newGroupName.trim() || !newGroupReason.trim()}
                  className="btn btn-primary py-1.5 px-4 text-[12.5px]"
                >
                  {createGroupMutation.isPending ? 'Creating…' : 'Create Group'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
