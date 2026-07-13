// icons.jsx — thin React wrapper over the Lucide UMD icon set.
// window.lucide.icons[Name] = array of [tagName, attrs] children.

const _ALIASES = {
  Palmtree: ['Palmtree', 'TreePalm'],
  UtensilsCrossed: ['UtensilsCrossed', 'Utensils'],
  Backpack: ['Backpack', 'Briefcase'],
  Flower2: ['Flower2', 'Flower'],
};

function _resolve(name) {
  const lib = (window.lucide && window.lucide.icons) || {};
  const tries = _ALIASES[name] || [name];
  for (const t of tries) if (lib[t]) return lib[t];
  // last resort: any icon so we never crash
  return lib.Circle || null;
}

function Icon({ name, size = 24, strokeWidth = 2, className = '', style = {}, ...rest }) {
  const node = _resolve(name);
  // lucide node = ["svg", svgAttrs, [ [tag, attrs], ... ]]; older builds may be just the children array
  const kids = Array.isArray(node) && node[0] === 'svg' ? node[2] : node;
  const children = (kids || []).map((entry, i) => {
    const tag = Array.isArray(entry) ? entry[0] : entry.tag;
    const attrs = Array.isArray(entry) ? entry[1] : entry.attrs;
    return React.createElement(tag, { key: i, ...attrs });
  });
  return (
    <svg
      width={size} height={size} viewBox="0 0 24 24" fill="none"
      stroke="currentColor" strokeWidth={strokeWidth}
      strokeLinecap="round" strokeLinejoin="round"
      className={className} style={style} aria-hidden="true" {...rest}
    >
      {children}
    </svg>
  );
}

Object.assign(window, { Icon });
