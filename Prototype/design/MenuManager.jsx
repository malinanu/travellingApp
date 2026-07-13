// MenuManager.jsx — sectioned menu editor for food vendors (Restaurant / Café / Bar).
// Items: name, Rs price, optional description + uploaded photo (object URL, session-only).
// Non-food business types see a pointer to the Profile type picker instead.

const { useState: useStateMM, useRef: useRefMM } = React;

const fmtRs = (n) => 'Rs ' + (Number(n) || 0).toLocaleString('en-US');

// tinted fallback tile when an item has no photo
function MenuThumb({ item, size = 56, radius = 14 }) {
  if (item.img) {
    return <img src={item.img} alt={item.name}
      style={{ width: size, height: size, borderRadius: radius, objectFit: 'cover', flexShrink: 0 }} />;
  }
  return (
    <span className="flex items-center justify-center flex-shrink-0"
      style={{ width: size, height: size, borderRadius: radius, background: 'var(--accent-tint)', color: 'var(--accent)' }}>
      <Icon name={item.icon || 'Utensils'} size={size * 0.42} />
    </span>
  );
}

// ── item editor card (add or edit) ─────────────────────────────
function MenuItemEditor({ accent, sections, initial, targetSection, onSave, onCancel }) {
  const [name, setName] = useStateMM(initial?.name || '');
  const [price, setPrice] = useStateMM(initial ? String(initial.price) : '');
  const [desc, setDesc] = useStateMM(initial?.desc || '');
  const [img, setImg] = useStateMM(initial?.img || null);
  const [sec, setSec] = useStateMM(targetSection);
  const fileRef = useRefMM(null);

  const valid = name.trim() && price;

  function pickFile(e) {
    const f = e.target.files && e.target.files[0];
    if (!f) return;
    if (img) URL.revokeObjectURL(img);
    setImg(URL.createObjectURL(f));
    e.target.value = '';
  }
  function removeImg() {
    if (img) URL.revokeObjectURL(img);
    setImg(null);
  }

  return (
    <div className="rounded-2xl p-5 bg-white space-y-4" style={{ border: `1.5px solid ${accent}` }}>
      <div className="flex items-center gap-2">
        <span className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: 'var(--accent-tint)', color: accent }}>
          <Icon name={initial ? 'Pencil' : 'Plus'} size={16} />
        </span>
        <h3 className="font-extrabold text-[16px] flex-1" style={{ color: 'var(--ink)' }}>{initial ? 'Edit item' : 'New menu item'}</h3>
        <button onClick={onCancel} className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ color: 'var(--muted)' }}>
          <Icon name="X" size={17} />
        </button>
      </div>

      <div className="flex gap-4" style={{ alignItems: 'flex-start' }}>
        {/* photo block */}
        <div className="flex-shrink-0" style={{ width: 128 }}>
          <input ref={fileRef} type="file" accept="image/*" onChange={pickFile} style={{ display: 'none' }} />
          {img ? (
            <div style={{ position: 'relative' }}>
              <img src={img} alt="dish" style={{ width: 128, height: 96, borderRadius: 14, objectFit: 'cover', display: 'block' }} />
              <button onClick={removeImg} title="Remove photo"
                className="absolute -top-2 -right-2 w-6 h-6 rounded-full flex items-center justify-center text-white"
                style={{ background: 'var(--ink)', boxShadow: '0 2px 8px rgba(16,61,58,0.4)' }}>
                <Icon name="X" size={13} strokeWidth={3} />
              </button>
              <button onClick={() => fileRef.current?.click()}
                className="w-full mt-1.5 text-[11.5px] font-extrabold" style={{ color: accent }}>Replace photo</button>
            </div>
          ) : (
            <button onClick={() => fileRef.current?.click()}
              className="w-full flex flex-col items-center justify-center gap-1.5 rounded-xl transition-colors"
              style={{ height: 96, border: '2px dashed var(--line)', background: 'var(--sand)', color: 'var(--muted)' }}>
              <Icon name="ImagePlus" size={20} />
              <span className="text-[11px] font-extrabold">Add photo</span>
            </button>
          )}
          <p className="text-[10.5px] font-semibold mt-1.5 leading-snug text-center" style={{ color: 'var(--muted)' }}>Optional — items without a photo get an icon tile</p>
        </div>

        {/* fields */}
        <div className="flex-1 min-w-0 space-y-3">
          <div className="flex gap-3">
            <label className="block flex-1 min-w-0">
              <span className="block text-[12px] font-extrabold mb-1" style={{ color: 'var(--ink-soft)' }}>Item name</span>
              <input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Jackfruit curry & rice" autoFocus
                className="w-full rounded-xl px-3 py-2.5 text-[14px] font-bold outline-none"
                style={{ background: 'var(--sand)', border: '1.5px solid var(--line)', color: 'var(--ink)' }} />
            </label>
            <label className="block flex-shrink-0" style={{ width: 130 }}>
              <span className="block text-[12px] font-extrabold mb-1" style={{ color: 'var(--ink-soft)' }}>Price</span>
              <div className="flex items-center gap-1.5 rounded-xl px-3" style={{ height: 44, background: 'var(--sand)', border: '1.5px solid var(--line)' }}>
                <span className="font-extrabold text-[13px]" style={{ color: 'var(--ink-soft)' }}>Rs</span>
                <input value={price} inputMode="numeric" placeholder="950"
                  onChange={(e) => setPrice(e.target.value.replace(/\D/g, ''))}
                  className="flex-1 bg-transparent outline-none text-[14px] font-bold w-full" style={{ color: 'var(--ink)' }} />
              </div>
            </label>
          </div>
          <label className="block">
            <span className="block text-[12px] font-extrabold mb-1" style={{ color: 'var(--ink-soft)' }}>Description <span className="font-semibold" style={{ color: 'var(--muted)' }}>· optional</span></span>
            <input value={desc} onChange={(e) => setDesc(e.target.value)} placeholder="Short and appetising — one line"
              className="w-full rounded-xl px-3 py-2.5 text-[13.5px] font-semibold outline-none"
              style={{ background: 'var(--sand)', border: '1.5px solid var(--line)', color: 'var(--ink)' }} />
          </label>
          <div>
            <span className="block text-[12px] font-extrabold mb-1.5" style={{ color: 'var(--ink-soft)' }}>Section</span>
            <div className="flex gap-1.5 flex-wrap">
              {sections.map((s) => {
                const on = sec === s.id;
                return (
                  <button key={s.id} onClick={() => setSec(s.id)}
                    className="rounded-full px-3 py-1.5 text-[12.5px] font-extrabold transition-all"
                    style={{ background: on ? accent : 'var(--sand)', color: on ? '#fff' : 'var(--ink-soft)',
                             border: `1.5px solid ${on ? accent : 'var(--line)'}` }}>{s.name}</button>
                );
              })}
            </div>
          </div>
        </div>
      </div>

      <div className="flex gap-2.5 pt-1">
        <button onClick={() => valid && onSave({ name: name.trim(), price: parseInt(price, 10) || 0, desc: desc.trim(), img, sec })}
          disabled={!valid}
          className="rounded-xl px-5 py-2.5 font-extrabold text-[13.5px] text-white flex items-center gap-2 transition-all active:scale-[0.98]"
          style={{ background: accent, opacity: valid ? 1 : 0.45, boxShadow: valid ? '0 8px 18px -8px var(--accent-shadow)' : 'none' }}>
          <Icon name="Check" size={16} strokeWidth={2.5} /> {initial ? 'Save changes' : 'Add to menu'}
        </button>
        <button onClick={onCancel} className="rounded-xl px-4 py-2.5 font-extrabold text-[13.5px]"
          style={{ background: 'var(--sand)', border: '1.5px solid var(--line)', color: 'var(--ink)' }}>Cancel</button>
      </div>
    </div>
  );
}

// ── traveller-side preview (matches BusinessDrawer's menu) ─────
function MenuPreview({ menu, accent }) {
  const total = menu.reduce((a, s) => a + s.items.length, 0);
  return (
    <div className="rounded-2xl p-5 bg-white sticky top-2" style={{ border: '1px solid var(--line)' }}>
      <div className="text-[12px] font-extrabold uppercase tracking-wide flex items-center gap-1.5" style={{ color: 'var(--muted)' }}>
        <Icon name="Smartphone" size={14} /> Traveler preview
      </div>
      <div className="mt-3 rounded-2xl p-3.5" style={{ border: '1px solid var(--line)', background: 'var(--sand)' }}>
        <div className="flex items-center justify-between">
          <span className="text-[11px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>Menu</span>
          <span className="text-[11px] font-bold" style={{ color: 'var(--muted)' }}>{total} items</span>
        </div>
        {menu.map((s) => s.items.length > 0 && (
          <div key={s.id} className="mt-2.5">
            <div className="text-[12px] font-extrabold" style={{ color: accent }}>{s.name}</div>
            {s.items.slice(0, 3).map((it) => (
              <div key={it.id} className="flex items-center gap-2.5 mt-2">
                <MenuThumb item={it} size={34} radius={10} />
                <div className="flex-1 min-w-0">
                  <div className="font-extrabold text-[12.5px] truncate" style={{ color: 'var(--ink)' }}>{it.name}</div>
                  {it.desc && <div className="text-[10.5px] font-semibold truncate" style={{ color: 'var(--muted)' }}>{it.desc}</div>}
                </div>
                <span className="font-extrabold text-[12px] whitespace-nowrap" style={{ color: 'var(--ink)' }}>{fmtRs(it.price)}</span>
              </div>
            ))}
          </div>
        ))}
      </div>
      <p className="text-[11.5px] font-semibold mt-3 leading-snug" style={{ color: 'var(--muted)' }}>
        This is how your menu appears on your spot's detail sheet in the traveler app.
      </p>
    </div>
  );
}

// ── main screen ────────────────────────────────────────────────
function MenuManager({ accent, menu, setMenu, isFood, onNav }) {
  const [editing, setEditing] = useStateMM(null);        // { sectionId, itemId|null } | null
  const [confirmDel, setConfirmDel] = useStateMM(null);  // sectionId pending two-tap delete
  const [newSection, setNewSection] = useStateMM('');
  const [renaming, setRenaming] = useStateMM(null);      // sectionId being renamed

  if (!isFood) {
    return (
      <div className="rounded-2xl bg-white p-10 flex flex-col items-center text-center" style={{ border: '1px solid var(--line)', maxWidth: 560, margin: '40px auto' }}>
        <span className="w-14 h-14 rounded-2xl flex items-center justify-center mb-4" style={{ background: 'var(--accent-tint)', color: accent }}>
          <Icon name="UtensilsCrossed" size={26} />
        </span>
        <h2 className="font-extrabold text-[20px]" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Menus are for food businesses</h2>
        <p className="text-[14px] font-semibold mt-2 leading-relaxed" style={{ color: 'var(--muted)' }}>
          Set your business type to Restaurant, Café or Bar in your profile and the menu tools unlock here.
        </p>
        <button onClick={() => onNav && onNav('profile')}
          className="mt-5 rounded-xl px-5 py-2.5 font-extrabold text-[14px] text-white active:scale-[0.98] transition-transform"
          style={{ background: accent, boxShadow: '0 8px 18px -8px var(--accent-shadow)' }}>
          Change business type
        </button>
      </div>
    );
  }

  const totalItems = menu.reduce((a, s) => a + s.items.length, 0);

  function saveItem(data) {
    const item = { id: editing.itemId || 'm' + Date.now(), name: data.name, price: data.price, desc: data.desc, img: data.img,
                   icon: (editing.itemId && menu.flatMap((s) => s.items).find((i) => i.id === editing.itemId)?.icon) || 'Utensils' };
    setMenu(menu.map((s) => {
      const without = { ...s, items: s.items.filter((i) => i.id !== item.id) };
      return without.id === data.sec ? { ...without, items: [...without.items, item] } : without;
    }));
    setEditing(null);
  }
  function deleteItem(sectionId, itemId) {
    setMenu(menu.map((s) => s.id === sectionId ? { ...s, items: s.items.filter((i) => i.id !== itemId) } : s));
  }
  function deleteSection(id) {
    if (confirmDel !== id) { setConfirmDel(id); setTimeout(() => setConfirmDel(null), 2400); return; }
    setMenu(menu.filter((s) => s.id !== id));
    setConfirmDel(null);
  }
  function addSection() {
    const name = newSection.trim();
    if (!name) return;
    setMenu([...menu, { id: 's' + Date.now(), name, items: [] }]);
    setNewSection('');
  }
  function renameSection(id, name) {
    setMenu(menu.map((s) => s.id === id ? { ...s, name: name.trim() || s.name } : s));
    setRenaming(null);
  }

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="font-extrabold tracking-tight" style={{ fontSize: 28, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Your menu</h1>
          <p className="text-[14.5px] font-semibold mt-1" style={{ color: 'var(--muted)' }}>
            {menu.length} sections · {totalItems} items — travelers see this on your spot's page.
          </p>
        </div>
        {!editing && menu.length > 0 && (
          <button onClick={() => setEditing({ sectionId: menu[0].id, itemId: null })}
            className="rounded-xl px-5 py-2.5 font-extrabold text-[14px] flex items-center gap-2 text-white active:scale-[0.98] transition-transform flex-shrink-0 whitespace-nowrap"
            style={{ background: accent, boxShadow: '0 8px 18px -8px var(--accent-shadow)' }}>
            <Icon name="Plus" size={17} strokeWidth={2.5} /> Add item
          </button>
        )}
      </div>

      <div className="grid gap-6" style={{ gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 340px)' }}>
        <div className="space-y-4 min-w-0">
          {editing && (
            <MenuItemEditor accent={accent} sections={menu}
              targetSection={editing.sectionId}
              initial={editing.itemId ? menu.flatMap((s) => s.items).find((i) => i.id === editing.itemId) : null}
              onSave={saveItem} onCancel={() => setEditing(null)} />
          )}

          {menu.map((s) => (
            <div key={s.id} className="rounded-2xl bg-white overflow-hidden" style={{ border: '1px solid var(--line)' }}>
              {/* section header */}
              <div className="flex items-center gap-2.5 px-5 py-3.5" style={{ borderBottom: '1px solid var(--line)', background: 'var(--sand)' }}>
                {renaming === s.id ? (
                  <input defaultValue={s.name} autoFocus
                    onBlur={(e) => renameSection(s.id, e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && renameSection(s.id, e.target.value)}
                    className="rounded-lg px-2 py-1 text-[15px] font-extrabold outline-none"
                    style={{ background: '#fff', border: `1.5px solid ${accent}`, color: 'var(--ink)' }} />
                ) : (
                  <h3 className="font-extrabold text-[15.5px]" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>{s.name}</h3>
                )}
                <span className="text-[12px] font-bold" style={{ color: 'var(--muted)' }}>· {s.items.length} items</span>
                <div className="ml-auto flex items-center gap-1">
                  <button onClick={() => setRenaming(s.id)} title="Rename section"
                    className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ color: 'var(--muted)' }}>
                    <Icon name="Pencil" size={15} />
                  </button>
                  <button onClick={() => deleteSection(s.id)} title="Delete section"
                    className="h-8 rounded-lg flex items-center justify-center px-2 text-[12px] font-extrabold transition-all"
                    style={{ color: confirmDel === s.id ? '#fff' : 'var(--muted)', background: confirmDel === s.id ? '#e11d48' : 'transparent' }}>
                    {confirmDel === s.id ? 'Sure?' : <Icon name="Trash2" size={15} />}
                  </button>
                </div>
              </div>
              {/* items */}
              {s.items.length === 0 ? (
                <div className="px-5 py-5 text-[13px] font-semibold" style={{ color: 'var(--muted)' }}>No items yet — add the first one.</div>
              ) : s.items.map((it, i) => (
                <div key={it.id} className="flex items-center gap-3.5 px-5 py-3" style={{ borderTop: i ? '1px solid var(--line)' : 'none' }}>
                  <MenuThumb item={it} />
                  <div className="flex-1 min-w-0">
                    <div className="font-extrabold text-[14.5px] truncate" style={{ color: 'var(--ink)' }}>{it.name}</div>
                    {it.desc && <div className="text-[12.5px] font-semibold truncate mt-0.5" style={{ color: 'var(--muted)' }}>{it.desc}</div>}
                  </div>
                  <span className="font-extrabold text-[14.5px] whitespace-nowrap" style={{ color: 'var(--ink)' }}>{fmtRs(it.price)}</span>
                  <div className="flex items-center gap-0.5 ml-1">
                    <button onClick={() => setEditing({ sectionId: s.id, itemId: it.id })} title="Edit item"
                      className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ color: accent }}>
                      <Icon name="Pencil" size={15} />
                    </button>
                    <button onClick={() => deleteItem(s.id, it.id)} title="Delete item"
                      className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ color: 'var(--muted)' }}>
                      <Icon name="Trash2" size={15} />
                    </button>
                  </div>
                </div>
              ))}
              {/* add item to this section */}
              <button onClick={() => setEditing({ sectionId: s.id, itemId: null })}
                className="w-full flex items-center gap-2 px-5 py-3 text-[13px] font-extrabold"
                style={{ borderTop: '1px solid var(--line)', color: accent }}>
                <Icon name="Plus" size={15} strokeWidth={2.5} /> Add item to {s.name}
              </button>
            </div>
          ))}

          {/* add section */}
          <div className="rounded-2xl px-4 py-3.5 flex items-center gap-2.5" style={{ border: '2px dashed var(--line)', background: 'transparent' }}>
            <Icon name="ListPlus" size={18} style={{ color: 'var(--muted)' }} />
            <input value={newSection} placeholder="New section — e.g. Desserts"
              onChange={(e) => setNewSection(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && addSection()}
              className="flex-1 bg-transparent outline-none text-[14px] font-bold" style={{ color: 'var(--ink)' }} />
            <button onClick={addSection} disabled={!newSection.trim()}
              className="rounded-lg px-3.5 py-2 text-[12.5px] font-extrabold text-white transition-all"
              style={{ background: accent, opacity: newSection.trim() ? 1 : 0.4 }}>Add section</button>
          </div>
        </div>

        <MenuPreview menu={menu} accent={accent} />
      </div>
    </div>
  );
}

Object.assign(window, { MenuManager });
