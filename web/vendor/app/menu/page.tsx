'use client';

// Menu manager — multiple named preset menus (owner picks which one is
// currently active/live), each a sectioned menu editor with an optional
// per-day availability restriction on sections/items, whole-document PUT
// /vendor/menu on save, with a live traveler preview. Port of the Menu
// mockup / MenuManager.jsx, extended with presets + day toggles.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Coffee, Cookie, EggFried, Fish, GlassWater, ImagePlus, Loader2, Martini, Pencil, Plus, Save,
  Sandwich, Smartphone, Soup, Trash2, UtensilsCrossed, Wine, X, LucideIcon,
} from 'lucide-react';
import { useEffect, useRef, useState } from 'react';

import { resolveMediaUrl, uploadMedia, validateFile, WEEKDAYS, type MenuPreset, type MenuSection, type StoredMenuDoc, type Weekday } from '@spotto/shared';
import Shell, { RoleLockedCard, useBusiness, useMe } from '../../components/shell';
import { api } from '../../lib/api';

const ITEM_ICONS: Record<string, LucideIcon> = {
  EggFried, Sandwich, Soup, Fish, Coffee, GlassWater, Cookie, Wine, Martini, UtensilsCrossed,
};

const DAY_LETTERS: Record<Weekday, string> = {
  monday: 'M', tuesday: 'T', wednesday: 'W', thursday: 'T', friday: 'F', saturday: 'S', sunday: 'S',
};

let nextId = 1;
const newId = (p: string) => `${p}_${Date.now().toString(36)}${nextId++}`;

// Any vendor who hasn't resaved since presets shipped may still have the
// pre-preset shape ({} or {sections:[...]}) — wrap it into one implicit
// preset so nothing is lost, and it's written back in the new shape on
// their next save.
function normalizeMenuDoc(doc: StoredMenuDoc | null | undefined): { presets: MenuPreset[]; activePresetId: string } {
  if (doc?.presets && doc.presets.length > 0) {
    const activeId = doc.presets.some((p) => p.id === doc.activePresetId) ? doc.activePresetId! : doc.presets[0].id;
    return { presets: doc.presets, activePresetId: activeId };
  }
  const main: MenuPreset = { id: 'preset_main', name: 'Main menu', sections: doc?.sections ?? [] };
  return { presets: [main], activePresetId: main.id };
}

function availableDaysLabel(days: Partial<Record<Weekday, boolean>> | undefined): string | null {
  if (!days) return null;
  const on = WEEKDAYS.filter((d) => days[d]);
  if (on.length === 0) return 'No days selected — hidden from travelers';
  return on.map((d) => d.slice(0, 1).toUpperCase() + d.slice(1, 3)).join(', ');
}

function DayChips({
  value, onChange,
}: {
  value: Partial<Record<Weekday, boolean>> | undefined;
  onChange: (next: Partial<Record<Weekday, boolean>> | undefined) => void;
}) {
  if (!value) {
    return (
      <button type="button" className="text-[11px] font-extrabold text-muted hover:text-accent" onClick={() => onChange({})}>
        Every day — customize
      </button>
    );
  }
  return (
    <div className="flex items-center gap-1">
      {WEEKDAYS.map((day) => {
        const on = !!value[day];
        return (
          <button key={day} type="button" title={day}
            className={`w-6 h-6 rounded-md text-[10px] font-extrabold transition-colors border ${
              on ? 'border-accent bg-accent text-white' : 'border-line bg-white text-ink hover:border-accent'
            }`}
            onClick={() => onChange({ ...value, [day]: !on })}>
            {DAY_LETTERS[day]}
          </button>
        );
      })}
      <button type="button" aria-label="Clear day restriction — available every day again"
        className="rounded-md p-1 text-muted hover:text-coral" onClick={() => onChange(undefined)}>
        <X size={13} />
      </button>
    </div>
  );
}

export default function MenuPage() {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const isStaff = me?.vendorRole === 'staff';
  const { data: business } = useBusiness();
  const { data } = useQuery<StoredMenuDoc>({
    queryKey: ['menu'],
    queryFn: () => api.get<StoredMenuDoc>('/vendor/menu'),
    enabled: !isStaff,
  });

  const [presets, setPresets] = useState<MenuPreset[]>([]);
  const [activePresetId, setActivePresetId] = useState('');
  const [editingPresetId, setEditingPresetId] = useState('');
  const [dirty, setDirty] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [newSectionName, setNewSectionName] = useState('');
  const [newPresetName, setNewPresetName] = useState('');
  const [renamingPresetId, setRenamingPresetId] = useState<string | null>(null);
  const [editingItem, setEditingItem] = useState<string | null>(null);
  const [editingSection, setEditingSection] = useState<string | null>(null);

  useEffect(() => {
    if (data && !dirty) {
      const { presets: p, activePresetId: a } = normalizeMenuDoc(data);
      setPresets(p);
      setActivePresetId(a);
      setEditingPresetId((prev) => (prev && p.some((x) => x.id === prev) ? prev : a));
    }
  }, [data, dirty]);

  const save = useMutation({
    mutationFn: () => api.put('/vendor/menu', { presets, activePresetId }),
    onSuccess: () => {
      setDirty(false);
      setSaveError(null);
      qc.invalidateQueries({ queryKey: ['menu'] });
    },
    onError: (err) => setSaveError(err instanceof Error ? err.message : 'Something went wrong — try again.'),
  });

  const editingPreset = presets.find((p) => p.id === editingPresetId) ?? presets[0];
  const sections = editingPreset?.sections ?? [];

  // Every edit only marks the document dirty — persisting happens
  // exclusively via the explicit "Save menu" button below, which always
  // reads the current render's presets/activePresetId. (An earlier version
  // of this page called save.mutate() immediately after some edits, which
  // raced React's state batching and could silently persist the
  // *previous* state — this is the actual cause of "can't add a menu"
  // reports: the new section appeared to save, then vanished on refetch.)
  const mutateSections = (fn: (s: MenuSection[]) => MenuSection[]) => {
    setPresets((ps) => ps.map((p) => (p.id !== editingPresetId ? p : { ...p, sections: fn(p.sections) })));
    setDirty(true);
  };

  const setItem = (sid: string, iid: string, field: 'name' | 'desc' | 'price', value: string) =>
    mutateSections((secs) =>
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

  const setItemImg = (sid: string, iid: string, img: string | null) =>
    mutateSections((secs) =>
      secs.map((s) =>
        s.id !== sid ? s : { ...s, items: s.items.map((i) => (i.id !== iid ? i : { ...i, img })) },
      ),
    );

  // Per-item photo upload straight to R2; on success the public URL becomes
  // the item's img (persisted with the next menu save).
  const [uploadingItem, setUploadingItem] = useState<string | null>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const uploadItemPhoto = async (sid: string, iid: string, file: File | undefined) => {
    if (!file || !business?.id) return;
    const invalid = validateFile(file, 'image');
    if (invalid) {
      setUploadError(invalid);
      return;
    }
    setUploadError(null);
    setUploadingItem(iid);
    try {
      const m = await uploadMedia(api, file, { ownerType: 'vendor', ownerId: business.id, role: 'menu_item' });
      setItemImg(sid, iid, m.url);
    } catch (e) {
      setUploadError(e instanceof Error ? e.message : 'Upload failed — try again.');
    } finally {
      setUploadingItem(null);
    }
  };

  const setItemAvailableDays = (sid: string, iid: string, days: Partial<Record<Weekday, boolean>> | undefined) =>
    mutateSections((secs) =>
      secs.map((s) =>
        s.id !== sid ? s : { ...s, items: s.items.map((i) => (i.id !== iid ? i : { ...i, availableDays: days })) },
      ),
    );

  const setSectionAvailableDays = (sid: string, days: Partial<Record<Weekday, boolean>> | undefined) =>
    mutateSections((secs) => secs.map((s) => (s.id !== sid ? s : { ...s, availableDays: days })));

  const addSection = () => {
    if (!newSectionName.trim()) return;
    mutateSections((s) => [...s, { id: newId('s'), name: newSectionName.trim(), items: [] }]);
    setNewSectionName('');
  };

  const addPreset = () => {
    if (!newPresetName.trim()) return;
    const preset: MenuPreset = { id: newId('p'), name: newPresetName.trim(), sections: [] };
    setPresets((ps) => [...ps, preset]);
    setEditingPresetId(preset.id);
    setNewPresetName('');
    setDirty(true);
  };

  const renamePreset = (id: string, name: string) => {
    setPresets((ps) => ps.map((p) => (p.id === id ? { ...p, name } : p)));
    setDirty(true);
  };

  const deletePreset = (id: string) => {
    if (presets.length <= 1) return;
    const fallbackId = presets.find((p) => p.id !== id)?.id ?? id;
    setPresets((ps) => ps.filter((p) => p.id !== id));
    setActivePresetId((prev) => (prev === id ? fallbackId : prev));
    setEditingPresetId((prev) => (prev === id ? fallbackId : prev));
    setDirty(true);
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
          <div className="flex items-start justify-between mb-4">
            <div>
              <h1 className="text-2xl font-extrabold font-display">Your menu</h1>
              <p className="text-muted font-semibold mt-0.5">
                {sections.length} sections · {totalItems} items — travelers see whichever menu is Active.
              </p>
            </div>
            <button
              className="btn-accent flex items-center gap-2 px-4 py-2.5"
              onClick={() =>
                mutateSections((secs) =>
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

          <div className="flex items-center gap-2 mb-5 flex-wrap">
            {presets.map((p) => {
              const isEditing = p.id === editingPresetId;
              const isActive = p.id === activePresetId;
              return (
                <div key={p.id}
                  className={`flex items-center gap-1.5 rounded-xl border px-3 py-2 ${isEditing ? 'border-accent bg-accent-tint' : 'border-line bg-white'}`}>
                  {renamingPresetId === p.id ? (
                    <input
                      className="font-extrabold text-[13.5px] bg-transparent outline-none border-b border-accent w-28"
                      autoFocus
                      value={p.name}
                      onChange={(e) => renamePreset(p.id, e.target.value)}
                      onBlur={() => setRenamingPresetId(null)}
                      onKeyDown={(e) => e.key === 'Enter' && setRenamingPresetId(null)}
                    />
                  ) : (
                    <button className="font-extrabold text-[13.5px]" onClick={() => setEditingPresetId(p.id)}>{p.name}</button>
                  )}
                  {isActive ? (
                    <span className="rounded-full bg-accent text-white text-[10px] font-extrabold px-1.5 py-0.5">Active</span>
                  ) : (
                    <button className="text-[11px] font-extrabold text-accent whitespace-nowrap"
                      onClick={() => { setActivePresetId(p.id); setDirty(true); }}>
                      Set active
                    </button>
                  )}
                  <button aria-label="Rename menu" className="rounded-lg p-1 text-muted hover:text-accent transition-colors"
                    onClick={() => setRenamingPresetId(p.id)}>
                    <Pencil size={12} />
                  </button>
                  {presets.length > 1 && (
                    <button aria-label="Delete menu" className="rounded-lg p-1 text-coral transition-colors"
                      onClick={() => deletePreset(p.id)}>
                      <Trash2 size={12} />
                    </button>
                  )}
                </div>
              );
            })}
            <div className="flex items-center gap-1.5 rounded-xl border-2 border-dashed border-line px-3 py-2">
              <input className="bg-transparent outline-none text-[13px] font-semibold w-28 placeholder:text-muted"
                placeholder="New menu — e.g. Weekend Brunch"
                value={newPresetName}
                onChange={(e) => setNewPresetName(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && addPreset()} />
              <button className="rounded-lg bg-line text-ink px-2 py-1 font-extrabold disabled:opacity-50"
                disabled={!newPresetName.trim()} onClick={addPreset}>
                <Plus size={13} />
              </button>
            </div>
          </div>

          <div className="space-y-4 mb-4">
            {sections.map((s) => {
              const sectionEditing = editingSection === s.id;
              const sectionDaysLabel = availableDaysLabel(s.availableDays);
              return (
              <div key={s.id} className="card p-0 overflow-hidden">
                <div className="px-5 py-3.5 bg-sand">
                  <div className="flex items-center gap-2">
                    {sectionEditing ? (
                      <input
                        className="font-extrabold text-[15px] bg-transparent outline-none border-b border-accent flex-1"
                        autoFocus
                        value={s.name}
                        onChange={(e) => mutateSections((secs) => secs.map((x) => (x.id === s.id ? { ...x, name: e.target.value } : x)))}
                        onKeyDown={(e) => e.key === 'Enter' && setEditingSection(null)}
                      />
                    ) : (
                      <p className="font-extrabold text-[15px]">
                        {s.name} <span className="text-muted font-semibold">· {s.items.length} items</span>
                        {sectionDaysLabel && <span className="text-muted font-semibold"> · {sectionDaysLabel}</span>}
                      </p>
                    )}
                    <div className="ml-auto flex items-center gap-1">
                      <button aria-label={sectionEditing ? 'Done editing section' : 'Edit section'}
                        className="rounded-lg p-1.5 text-muted hover:text-accent hover:bg-white transition-colors"
                        onClick={() => setEditingSection(sectionEditing ? null : s.id)}>
                        <Pencil size={14} />
                      </button>
                      <button aria-label="Delete section" className="rounded-lg p-1.5 text-coral hover:bg-white transition-colors"
                        onClick={() => mutateSections((secs) => secs.filter((x) => x.id !== s.id))}>
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>
                  {sectionEditing && (
                    <div className="flex items-center gap-2 mt-2">
                      <span className="text-[11px] font-extrabold text-muted uppercase">Available</span>
                      <DayChips value={s.availableDays} onChange={(days) => setSectionAvailableDays(s.id, days)} />
                    </div>
                  )}
                </div>

                <div className="p-3">
                  {s.items.map((i) => {
                    const Icon = ITEM_ICONS[i.icon] ?? UtensilsCrossed;
                    const editing = editingItem === i.id;
                    const itemDaysLabel = availableDaysLabel(i.availableDays);
                    return (
                      <div key={i.id} className="px-2 py-2.5 border-b border-line last:border-0">
                        <div className="flex items-center gap-3">
                          <MenuItemThumb
                            img={i.img}
                            Icon={Icon}
                            editable={editing}
                            busy={uploadingItem === i.id}
                            onPick={(f) => uploadItemPhoto(s.id, i.id, f)}
                            onRemove={() => setItemImg(s.id, i.id, null)}
                          />
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
                              <p className="text-[12.5px] text-muted font-semibold truncate">
                                {i.desc}{itemDaysLabel && (i.desc ? ' · ' : '') + itemDaysLabel}
                              </p>
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
                              onClick={() => setEditingItem(editing ? null : i.id)}>
                              <Pencil size={14} />
                            </button>
                            <button aria-label="Delete item" className="rounded-lg p-1.5 text-coral hover:bg-[#fde4d8] transition-colors"
                              onClick={() =>
                                mutateSections((secs) =>
                                  secs.map((x) => (x.id === s.id ? { ...x, items: x.items.filter((y) => y.id !== i.id) } : x)),
                                )
                              }>
                              <Trash2 size={14} />
                            </button>
                          </div>
                        </div>
                        {editing && (
                          <div className="flex items-center gap-2 mt-2 pl-12">
                            <span className="text-[11px] font-extrabold text-muted uppercase">Available</span>
                            <DayChips value={i.availableDays} onChange={(days) => setItemAvailableDays(s.id, i.id, days)} />
                          </div>
                        )}
                      </div>
                    );
                  })}
                  <button
                    className="text-[13px] font-extrabold text-accent flex items-center gap-1.5 mt-2 px-2 hover:underline"
                    onClick={() =>
                      mutateSections((secs) =>
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
              );
            })}
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

          {(saveError || uploadError) && <p className="text-[13.5px] font-bold text-coral mt-4">{saveError || uploadError}</p>}

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
          {editingPresetId !== activePresetId && (
            <p className="text-[11.5px] font-bold mb-3" style={{ color: '#d97706' }}>
              Previewing &ldquo;{editingPreset?.name}&rdquo; — not the currently active menu.
            </p>
          )}
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
                    {i.img ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={resolveMediaUrl(i.img)} alt="" className="w-8 h-8 rounded-lg object-cover shrink-0" />
                    ) : (
                      <span className="w-8 h-8 rounded-lg bg-accent-tint text-accent flex items-center justify-center shrink-0">
                        {(() => {
                          const Icon = ITEM_ICONS[i.icon] ?? UtensilsCrossed;
                          return <Icon size={14} />;
                        })()}
                      </span>
                    )}
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

// MenuItemThumb shows a dish photo when set, otherwise the item's icon. While
// the row is being edited it doubles as an uploader (click to pick, with a
// remove affordance once a photo exists).
function MenuItemThumb({
  img, Icon, editable, busy, onPick, onRemove,
}: {
  img?: string | null; Icon: LucideIcon; editable: boolean; busy: boolean;
  onPick: (f?: File) => void; onRemove: () => void;
}) {
  const ref = useRef<HTMLInputElement>(null);
  return (
    <div className="relative w-9 h-9 shrink-0 group">
      {img ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={resolveMediaUrl(img)} alt="" className="w-9 h-9 rounded-xl object-cover" />
      ) : (
        <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center">
          <Icon size={17} />
        </span>
      )}
      {editable && (
        <>
          <button type="button" disabled={busy} onClick={() => ref.current?.click()}
            className="absolute inset-0 rounded-xl flex items-center justify-center bg-black/45 text-white opacity-0 group-hover:opacity-100 transition-opacity disabled:opacity-100"
            aria-label="Upload item photo">
            {busy ? <Loader2 size={15} className="animate-spin" /> : <ImagePlus size={15} />}
          </button>
          {img && !busy && (
            <button type="button" onClick={onRemove}
              className="absolute -right-1.5 -top-1.5 rounded-full p-0.5 bg-coral text-white opacity-0 group-hover:opacity-100 transition-opacity"
              aria-label="Remove item photo">
              <X size={11} />
            </button>
          )}
          <input ref={ref} type="file" accept="image/*" className="hidden"
            onChange={(e) => { onPick(e.target.files?.[0]); e.target.value = ''; }} />
        </>
      )}
    </div>
  );
}
