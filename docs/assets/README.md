# Artwork

| File | Use |
|---|---|
| `logo.svg` | Source logo; shown at the top of the project README |
| `logo.png` | 512 × 512 transparent export of `logo.svg` |
| `social-preview.svg` | Source for the GitHub social preview |
| `social-preview.png` | 1280 × 640 export uploaded as the social preview |

The logo is a cumulative depth chart: bids (green, `#12A37A`) and asks
(red, `#E5484D`) step outward from a dashed mid line (`#94A3B8`), with
fixed-width bands underneath. It uses no fonts.

`social-preview.svg` copies the logo paths and sets text in
Helvetica Neue, so its rendering depends on installed fonts. The PNG is
the canonical export.

## Regenerating the PNGs

On macOS, `sips` renders the SVGs directly:

```sh
cd docs/assets
sips -s format png social-preview.svg --out social-preview.png
sed 's/width="256" height="256"/width="512" height="512"/' logo.svg > /tmp/logo-512.svg
sips -s format png /tmp/logo-512.svg --out logo.png
```

Edit the logo paths in both SVGs together if the mark changes.
