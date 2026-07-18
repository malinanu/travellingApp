'use client';

// Menu manager — sectioned menu editor (food categories only), whole-document
// PUT /vendor/menu on save, with a live traveler preview. Port of the Menu
// mockup / MenuManager.jsx.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Coffee, Cookie, EggFried, Fish, GlassWater, Martini, Pencil, Plus, Save,
  Sandwich, Smartphone, Soup, Trash2, UtensilsCrossed, Wine, LucideIcon,
} from 'lucide-react';
import { useEffect, useState } from 'react';

import type { MenuSection } from '@spotto/shared';
import Shell, { RoleLockedCard, useBusiness, useMe } from '../../components/shell';
import { api } from '../../lib/api';

const ITEM_ICONS: Record<string, LucideIcon> = {
  EggFried, Sandwich, Soup, Fish, Coffee, GlassWater, Cookie, Wine, Martini, UtensilsCrossed,
};

interface MenuDoc {
  sections?: MenuSection[];
}

let nextId = 1;
const newId = (p: string) => `${p}_${Date.now().toString(36)}${nextId++}`;

export default function MenuPage() {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const isStaff = me?.vendorRole === 'staff';
  const { data: business } = useBusiness();
  const { data } = useQuery<MenuDoc>({
    queryKey: ['menu'],
    queryFn: () => api.get<MenuDoc>('/vendor/menu'),
    enabled: !isStaff,
  });
  const [sections, setSections] = useState<MenuSection[]>([]);
  const [dirty, setDirty] = useState(false);
  const [newSectionName, setNewSectionName] = useState('');
  const [editingItem, setEditingItem] = useState<string | null>(null);
  const [editingSection, setEditingSection] = useState<string | null>(null);

  useEffect(() => {
    if (data?.sections && !dirty) setSections(data.sections);
  }, [data, dirty]);

  const save = useMutation({
    mutationFn: () => api.put('/vendor/menu', { sections }),
    onSuccess: () => {
      setDirty(false);
      qc.invalidateQueries({ queryKey: ['menu'] });
    },
  });

  const mutate = (fn: (s: MenuSection[]) => MenuSection[]) => {
    setSections(fn);
    setDirty(true);
  };

  const setItem = (sid: string, iid: string, field: 'name' | 'desc' | 'price', value: string) =>
    mutate((secs) =>
      secs.map((s) =>
        s.id !== sid
          ? s
          : {
              ...s,
              items: s.items.map((i) =>
                i.id !== iid ? i : { ...i, [field]: field === 'price' ? Number(value) || 0 : value },
              ),
            },
      ),
    );

  const addSection = () => {
    if (!newSectionName.trim()) return;
    mutate((s) => [...s, { id: newId('s'), name: newSectionName.trim(), items: [] }]);
    setNewSectionName('');
    save.mutate();
  };

  const totalItems = sections.reduce((a, s) => a + s.items.length, 0);

  if (isStaff) {
    return (
      <Shell>
        <div className="mb-5">
          <h1 className="text-2xl font-extrabold font-display">Your menu</h1>
        </div>
        <RoleLockedCard what="Menu" />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="grid lg:grid-cols-[1fr_320px] gap-5 items-start">
        <div>
          <div className="flex items-start justify-between mb-5">
            <div>
              <h1 className="text-2xl font-extrabold font-display">Your menu</h1>
              <p className="text-muted font-semibold mt-0.5">
                {sections.length} sections · {totalItems} items — travelers see this on your spot&rsquo;s page.
              </p>
            </div>
            <button
              className="btn-accent flex items-center gap-2 px-4 py-2.5"
              onClick={() =>
                mutate((secs) =>
                  secs.length === 0
                    ? secs
                    : secs.map((s, i) =>
                        i === 0
                          ? { ...s, items: [...s.items, { id: newId('m'), name: '', price: 0, desc: '', icon: 'UtensilsCrossed' }] }
                          : s,
                      ),
                )
              }
              disabled={sections.length === 0}
            >
              <Plus size={16} /> Add item
            </button>
          </div>

          <div className="space-y-4 mb-4">
            {sections.map((s) => (
              <div key={s.id} className="card p-0 overflow-hidden">
                <div className="flex items-center gap-2 px-5 py-3.5 bg-sand">
                  {editingSection === s.id ? (
                    <input
                      className="font-extrabold text-[15px] bg-transparent outline-none border-b border-accent flex-1"
                      autoFocus
                      value={s.name}
                      onChange={(e) => mutate((secs) => secs.map((x) => (x.id === s.id ? { ...x, name: e.target.value } : x)))}
                      onBlur={() => setEditingSection(null)}
                      onKeyDown={(e) => e.key === 'Enter' && setEditingSection(null)}
                    />
                  ) : (
                    <p className="font-extrabold text-[15px]">
                      {s.name} <span className="text-muted font-semibold">· {s.items.length} items</span>
                    </p>
                  )}
                  <div className="ml-auto flex items-center gap-1">
                    <button aria-label="Edit section" className="rounded-lg p-1.5 text-muted hover:text-accent hover:bg-white transition-colors"
                      onClick={() => setEditingSection(s.id)}>
                      <Pencil size={14} />
                    </button>
                    <button aria-label="Delete section" className="rounded-lg p-1.5 text-coral hover:bg-white transition-colors"
                      onClick={() => mutate((secs) => secs.filter((x) => x.id !== s.id))}>
                      <Trash2 size={14} />
                    </button>
                  </div>
                </div>

                <div className="p-3">
                  {s.items.map((i) => {
                    const Icon = ITEM_ICONS[i.icon] ?? UtensilsCrossed;
                    const editing = editingItem === i.id;
                    return (
                      <div key={i.id} className="flex items-center gap-3 px-2 py-2.5 border-b border-line last:border-0">
                        <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center shrink-0">
                          <Icon size={17} />
                        </span>
                        {editing ? (
                          <div className="flex-1 grid grid-cols-[1fr_2fr] gap-2">
                            <input className="input !py-1.5 text-[13.5px]" value={i.name}
                              onChange={(e) => setItem(s.id, i.id, 'name', e.target.value)} placeholder="Item" autoFocus />
                            <input className="input !py-1.5 text-[13.5px]" value={i.desc}
                              onChange={(e) => setItem(s.id, i.id, 'desc', e.target.value)} placeholder="Description" />
                          </div>
                        ) : (
                          <div className="flex-1 min-w-0">
                            <p className="font-extrabold text-[14px] truncate">{i.name || 'Untitled item'}</p>
                            <p className="text-[12.5px] text-muted font-semibold truncate">{i.desc}</p>
                          </div>
                        )}
                        {editing ? (
                          <div className="relative w-24 shrink-0">
                            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-[12px] font-extrabold text-muted">Rs</span>
                            <input className="input !py-1.5 !pl-8 text-[13.5px]" value={i.price}
                              onChange={(e) => setItem(s.id, i.id, 'price', e.target.value)} inputMode="numeric" />
                          </div>
                        ) : (
                          <p className="font-extrabold text-[14px] shrink-0">Rs {i.price.toLocaleString()}</p>
                        )}
                        <div className="flex items-center gap-1 shrink-0">
                          <button aria-label={editing ? 'Done editing' : 'Edit item'}
                            className="rounded-lg p-1.5 text-muted hover:text-accent hover:bg-sand transition-colors"
                            onClick={() => {
                              setEditingItem(editing ? null : i.id);
                              if (editing) save.mutate();
                            }}>
                            <Pencil size={14} />
                          </button>
                          <button aria-label="Delete item" className="rounded-lg p-1.5 text-coral hover:bg-[#fde4d8] transition-colors"
                            onClick={() =>
                              mutate((secs) =>
                                secs.map((x) => (x.id === s.id ? { ...x, items: x.items.filter((y) => y.id !== i.id) } : x)),
                              )
                            }>
                            <Trash2 size={14} />
                          </button>
                        </div>
                      </div>
                    );
                  })}
                  <button
                    className="text-[13px] font-extrabold text-accent flex items-center gap-1.5 mt-2 px-2 hover:underline"
                    onClick={() =>
                      mutate((secs) =>
                        secs.map((x) =>
                          x.id === s.id
                            ? { ...x, items: [...x.items, { id: newId('m'), name: '', price: 0, desc: '', icon: 'UtensilsCrossed' }] }
                            : x,
                        ),
                      )
                    }
                  >
                    <Plus size={14} /> Add item to {s.name}
                  </button>
                </div>
              </div>
            ))}
          </div>

          <div className="rounded-2xl border-2 border-dashed border-line p-4 flex items-center gap-3">
            <input
              className="flex-1 bg-transparent outline-none text-[14px] font-semibold placeholder:text-muted"
              placeholder="New section — e.g. Desserts"
              value={newSectionName}
              onChange={(e) => setNewSectionName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && addSection()}
            />
            <button className="rounded-xl bg-line text-ink px-4 py-2 font-extrabold text-[13px] disabled:opacity-50"
              disabled={!newSectionName.trim()} onClick={addSection}>
              Add section
            </button>
          </div>

          {dirty && (
            <div className="flex justify-end mt-4">
              <button className="btn-accent px-4 py-2.5 flex items-center gap-2 text-[13.5px]"
                disabled={save.isPending} onClick={() => save.mutate()}>
                <Save size={16} /> {save.isPending ? 'Saving…' : 'Save menu'}
              </button>
            </div>
          )}
        </div>

        <div className="card p-5 lg:sticky lg:top-5">
          <p className="text-[11px] font-extrabold uppercase text-muted mb-3 flex items-center gap-1.5">
            <Smartphone size={12} /> Traveler preview
          </p>
          <div className="rounded-2xl border border-line p-4">
            <div className="flex items-center justify-between mb-3">
              <p className="font-extrabold text-[14px]">Menu</p>
              <span className="text-[11.5px] text-muted font-bold">{totalItems} items</span>
            </div>
            {sections.map((s) => (
              <div key={s.id} className="mb-3 last:mb-0">
                <p className="text-[11.5px] font-extrabold text-accent uppercase mb-1.5">{s.name}</p>
                {s.items.map((i) => (
                  <div key={i.id} className="flex items-center gap-2 mb-2">
                    <span className="w-8 h-8 rounded-lg bg-accent-tint text-accent flex items-center justify-center shrink-0">
                      {(() => {
                        const Icon = ITEM_ICONS[i.icon] ?? UtensilsCrossed;
                        return <Icon size={14} />;
                      })()}
                    </span>
                    <div className="flex-1 min-w-0">
                      <p className="text-[12.5px] font-extrabold truncate">{i.name || 'Untitled item'}</p>
                      <p className="text-[11px] text-muted font-semibold truncate">{i.desc}</p>
                    </div>
                    <p className="text-[12.5px] font-extrabold shrink-0">Rs {i.price.toLocaleString()}</p>
                  </div>
                ))}
              </div>
            ))}
          </div>
          <p className="text-[11.5px] text-muted font-semibold mt-3">
            This is how your menu appears on {business?.name ?? 'your spot'}&rsquo;s detail sheet in the traveler app.
          </p>
        </div>
      </div>
    </Shell>
  );
}
