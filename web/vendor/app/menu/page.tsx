'use client';

// Menu manager — sectioned menu editor (food categories only), whole-document
// PUT /vendor/menu on save. Port of MenuManager.jsx.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus, Save, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';

import type { MenuSection } from '@spotto/shared';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

interface MenuDoc {
  sections?: MenuSection[];
}

let nextId = 1;
const newId = (p: string) => `${p}_${Date.now().toString(36)}${nextId++}`;

export default function MenuPage() {
  const qc = useQueryClient();
  const { data } = useQuery<MenuDoc>({
    queryKey: ['menu'],
    queryFn: () => api.get<MenuDoc>('/vendor/menu'),
  });
  const [sections, setSections] = useState<MenuSection[]>([]);
  const [dirty, setDirty] = useState(false);

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

  return (
    <Shell>
      <div className="flex items-center justify-between mb-5">
        <p className="text-muted font-semibold text-[14px]">
          {sections.reduce((a, s) => a + s.items.length, 0)} items in {sections.length} sections
        </p>
        <div className="flex gap-2">
          <button
            className="rounded-xl border border-line bg-white px-4 py-2.5 font-extrabold text-[13.5px] flex items-center gap-2 hover:border-accent"
            onClick={() => mutate((s) => [...s, { id: newId('s'), name: 'New section', items: [] }])}
          >
            <Plus size={16} /> Section
          </button>
          <button className="btn-accent px-4 py-2.5 flex items-center gap-2 text-[13.5px]"
            disabled={!dirty || save.isPending} onClick={() => save.mutate()}>
            <Save size={16} /> {save.isPending ? 'Saving…' : dirty ? 'Save menu' : 'Saved'}
          </button>
        </div>
      </div>

      <div className="space-y-4">
        {sections.map((s) => (
          <div key={s.id} className="card p-5">
            <div className="flex items-center gap-3 mb-3">
              <input
                className="font-extrabold text-[16px] bg-transparent outline-none border-b border-transparent focus:border-accent"
                value={s.name}
                onChange={(e) => mutate((secs) => secs.map((x) => (x.id === s.id ? { ...x, name: e.target.value } : x)))}
              />
              <button
                aria-label="Remove section"
                className="ml-auto rounded-lg border border-line p-2 text-coral hover:bg-[#fde4d8]"
                onClick={() => mutate((secs) => secs.filter((x) => x.id !== s.id))}
              >
                <Trash2 size={14} />
              </button>
            </div>
            <div className="space-y-2">
              {s.items.map((i) => (
                <div key={i.id} className="grid grid-cols-[1fr_2fr_110px_36px] gap-2 items-center group">
                  <input className="input !py-2 text-[13.5px]" value={i.name}
                    onChange={(e) => setItem(s.id, i.id, 'name', e.target.value)} placeholder="Item" />
                  <input className="input !py-2 text-[13.5px]" value={i.desc}
                    onChange={(e) => setItem(s.id, i.id, 'desc', e.target.value)} placeholder="Description" />
                  <div className="relative">
                    <span className="absolute left-3 top-1/2 -translate-y-1/2 text-[12px] font-extrabold text-muted">Rs</span>
                    <input className="input !py-2 !pl-9 text-[13.5px]" value={i.price}
                      onChange={(e) => setItem(s.id, i.id, 'price', e.target.value)} inputMode="numeric" />
                  </div>
                  <button
                    aria-label="Remove item"
                    className="rounded-lg border border-line p-2 text-coral opacity-0 group-hover:opacity-100 transition-opacity hover:bg-[#fde4d8]"
                    onClick={() =>
                      mutate((secs) =>
                        secs.map((x) => (x.id === s.id ? { ...x, items: x.items.filter((y) => y.id !== i.id) } : x)),
                      )
                    }
                  >
                    <Trash2 size={13} />
                  </button>
                </div>
              ))}
              <button
                className="text-[13px] font-extrabold text-accent flex items-center gap-1.5 mt-1 hover:underline"
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
                <Plus size={14} /> Add item
              </button>
            </div>
          </div>
        ))}
      </div>
    </Shell>
  );
}
