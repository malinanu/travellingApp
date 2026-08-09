'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import * as Icons from 'lucide-react';
import { useState } from 'react';
import { VendorCategory, VENDOR_SEGMENTS, VendorSegment } from '@spotto/shared';
import Shell from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

export default function CategoriesPage() {
  const qc = useQueryClient();

  const { data: categories, error, isLoading } = useQuery<VendorCategory[]>({
    queryKey: ['admin_categories'],
    queryFn: () => api.get<VendorCategory[]>('/admin/categories'),
  });

  const [editingCategory, setEditingCategory] = useState<VendorCategory | 'new' | null>(null);

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.del(`/admin/categories/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin_categories'] });
    },
  });

  const grouped = (categories || []).reduce((acc, cat) => {
    if (!acc[cat.segment]) acc[cat.segment] = [];
    acc[cat.segment].push(cat);
    return acc;
  }, {} as Record<string, VendorCategory[]>);

  return (
    <Shell>
      {error && <InlineErrorBanner message="Failed to load categories." />}

      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold">Vendor Categories</h1>
        <button className="btn btn-primary text-sm px-4 py-2" onClick={() => setEditingCategory('new')}>
          + New Category
        </button>
      </div>

      {isLoading ? (
        <div className="flex justify-center p-10"><Icons.Loader2 className="animate-spin text-muted" size={24} /></div>
      ) : (
        <div className="flex flex-col gap-8">
          {VENDOR_SEGMENTS.map(segment => {
            const segCats = grouped[segment.id] || [];
            if (segCats.length === 0) return null;

            return (
              <div key={segment.id}>
                <h2 className="text-[13px] font-bold uppercase tracking-wide text-muted mb-4 pb-2 border-b border-border">
                  {segment.label}
                </h2>
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
                  {segCats.map(cat => {
                    const IconComp = (Icons as any)[cat.icon] || Icons.CircleDashed;
                    return (
                      <div key={cat.id} className="panel p-4 flex flex-col gap-3 group relative">
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-full bg-accent/30 flex items-center justify-center shrink-0">
                            <IconComp size={18} className="text-primary" />
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="font-bold text-[15px] truncate">{cat.label}</div>
                            <div className="text-[12px] font-mono text-muted truncate">{cat.id}</div>
                          </div>
                        </div>

                        {cat.bookingType && (
                          <div className="text-[11.5px] font-semibold text-muted bg-accent/30 rounded px-2 py-1 self-start">
                            {cat.bookingType} booking
                          </div>
                        )}

                        <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity flex gap-1">
                          <button
                            className="w-7 h-7 rounded bg-background hover:bg-accent flex items-center justify-center text-muted hover:text-primary transition-colors border border-border"
                            title="Edit"
                            onClick={() => setEditingCategory(cat)}
                          >
                            <Icons.Pencil size={12} />
                          </button>
                          <button
                            className="w-7 h-7 rounded bg-background hover:bg-red-500/10 flex items-center justify-center text-muted hover:text-red-500 transition-colors border border-border"
                            title="Delete"
                            onClick={() => {
                              if (window.confirm(`Delete category ${cat.label}? Vendors using this category might break.`)) {
                                deleteMutation.mutate(cat.id);
                              }
                            }}
                          >
                            <Icons.Trash2 size={12} />
                          </button>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {editingCategory && (
        <CategoryModal
          category={editingCategory === 'new' ? undefined : editingCategory}
          onClose={() => setEditingCategory(null)}
          onSaved={() => {
            setEditingCategory(null);
            qc.invalidateQueries({ queryKey: ['admin_categories'] });
          }}
        />
      )}
    </Shell>
  );
}

function CategoryModal({ category, onClose, onSaved }: { category?: VendorCategory; onClose: () => void; onSaved: () => void }) {
  const isNew = !category;
  
  const mutation = useMutation({
    mutationFn: (data: VendorCategory) => {
      if (isNew) {
        return api.post<VendorCategory>('/admin/categories', data);
      } else {
        return api.put<VendorCategory>(`/admin/categories/${data.id}`, data);
      }
    },
    onSuccess: () => onSaved(),
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-card w-full max-w-lg rounded-xl shadow-2xl border border-border flex flex-col max-h-full">
        <div className="flex items-center justify-between p-5 border-b border-border shrink-0">
          <h2 className="text-lg font-bold">{isNew ? 'New Category' : 'Edit Category'}</h2>
          <button onClick={onClose} className="text-muted hover:text-foreground">
            <Icons.X size={20} />
          </button>
        </div>
        
        <form 
          className="flex flex-col overflow-y-auto p-5 gap-5"
          onSubmit={(e) => {
            e.preventDefault();
            const fd = new FormData(e.currentTarget);
            mutation.mutate({
              id: fd.get('id') as string,
              label: fd.get('label') as string,
              segment: fd.get('segment') as VendorSegment,
              icon: fd.get('icon') as string,
              bookingType: (fd.get('bookingType') as any) || undefined,
            });
          }}
        >
          {mutation.isError && (
            <div className="bg-red-500/10 text-red-500 p-3 rounded text-sm font-semibold">
              Failed to save category. Check your inputs.
            </div>
          )}

          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-bold text-muted uppercase tracking-wide">ID (slug)</label>
            <input 
              name="id" 
              required 
              pattern="[a-z0-9\-]+" 
              title="Lowercase alphanumeric and hyphens only"
              defaultValue={category?.id} 
              disabled={!isNew}
              className="input disabled:opacity-50" 
              placeholder="e.g. coffee-shop"
            />
            {isNew && <span className="text-[11px] text-muted-foreground">Cannot be changed later.</span>}
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-bold text-muted uppercase tracking-wide">Label</label>
            <input name="label" required defaultValue={category?.label} className="input" placeholder="e.g. Coffee Shop" />
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-bold text-muted uppercase tracking-wide">Segment</label>
            <select name="segment" required defaultValue={category?.segment || 'hospitality'} className="input">
              {VENDOR_SEGMENTS.map(s => (
                <option key={s.id} value={s.id}>{s.label}</option>
              ))}
            </select>
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-bold text-muted uppercase tracking-wide">Lucide Icon Name</label>
            <input name="icon" required defaultValue={category?.icon} className="input" placeholder="e.g. Coffee" />
            <span className="text-[11px] text-muted-foreground">Must be a valid React-Lucide icon name (PascalCase).</span>
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-bold text-muted uppercase tracking-wide">Booking Type (Optional)</label>
            <select name="bookingType" defaultValue={category?.bookingType || ''} className="input">
              <option value="">None (Not Bookable)</option>
              <option value="table">Table (Restaurants, Cafes)</option>
              <option value="stay">Stay (Hotels, Hostels)</option>
              <option value="session">Session (Classes, Tours)</option>
              <option value="rental">Rental (Bikes, Surfboards)</option>
            </select>
          </div>

          <div className="flex justify-end gap-3 mt-4 pt-4 border-t border-border">
            <button type="button" onClick={onClose} className="btn bg-accent text-accent-foreground px-4 py-2 hover:bg-accent/80">Cancel</button>
            <button type="submit" disabled={mutation.isPending} className="btn btn-primary px-4 py-2 flex items-center gap-2">
              {mutation.isPending && <Icons.Loader2 size={14} className="animate-spin" />}
              {isNew ? 'Create' : 'Save Changes'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
