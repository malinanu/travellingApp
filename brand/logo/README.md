# Spotto logo — usage guide

The Spotto mark is a **passport stamp**: a sun rising over the horizon above "SPOTTO" — travel, warmth, a place
you've been. Masters are clean vectors; colours follow the app's "Sunny Postcard" palette.

## Files

| File | Use |
|---|---|
| `spotto-stamp.svg` | **Primary logo.** Website hero, splash/onboarding, marketing, merch |
| `spotto-horizontal.svg` | Headers, nav bars, email signatures, documents |
| `spotto-symbol.svg` | Symbol only, ≥ 48 px (headers next to text, avatars) |
| `spotto-symbol-small.svg` | Symbol below 48 px — favicons, tiny UI (two heavier lines, bigger sun) |
| `spotto-app-icon.svg` / `-small.svg` | Cream tile app icon (full / small-size cut) |
| `*-reversed.svg` | On dark backgrounds (ink `#103D3A` or darker) |
| `*-ink.svg`, `*-white.svg` | One-colour: stamps, embossing, single-colour print |
| `web/` | favicon.ico/svg, PNG icons, apple-touch-icon, webmanifest, `<head>` snippet |
| `png/` | High-res PNGs of stamp, horizontal and the 1024 app icon |

## Colours

| Name | HEX | RGB | Used for |
|---|---|---|---|
| Ink | `#103D3A` | 16 61 58 | Frame, horizon, lettering, stars |
| Ocean teal | `#0B6F86` | 11 111 134 | Sea lines (on light) |
| Lagoon | `#0E8C8A` | 14 140 138 | Sea lines (reversed, on dark) |
| Sun | `#F6B81E` | 246 184 30 | The sun only |
| Cream | `#FBF6ED` | 251 246 237 | Backgrounds, app-icon tile, reversed lettering |

## Rules

- **Clear space:** keep empty space around the logo at least the height of the sun (stamp) or the height of the
  wordmark (horizontal).
- **Minimum sizes:** stamp 96 px / 25 mm wide (lettering gets too small below); horizontal 120 px; symbol 48 px;
  below that use `spotto-symbol-small.svg`.
- **Backgrounds:** full colour on cream or white; reversed on ink or darker; one-colour white on photos.
- **Don't:** recolour the sun, stretch or rotate, add shadows/gradients/outlines, re-type "SPOTTO" in a font,
  put the full-colour logo on busy photos, or add a rough "inked" texture to the master.

## Notes

- Lettering is custom-drawn geometry (no font licence involved). Strokes are kept as SVG strokes; expand them to
  outlines in a vector editor before sending to print/embroidery vendors.
- Derived from an AI-generated concept; run a trademark search before registering.
- Regenerate the SVG masters: `python build_logo.py .` (from this folder). Edit geometry/colours there, not in the SVGs.
