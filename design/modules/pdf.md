# Module Specification: pdf

- Target source file path: `src/pixelbonus/pdf/render.go`
- Owned APIs:
  - `RenderCoupons`
- Internal logic:
  - Produce a valid PDF file without introducing a heavy framework.
  - Page size A4, 1 cm margins, 10 rows by 2 columns.
  - Calculate coupon box dimensions from the printable area.
  - Draw cut marks at coupon boundaries.
  - For each coupon:
    - render QR code image for `{baseURL}/r?c={url-safe-code}`
    - print grouped hex code in fixed-width text on the right
    - print `pixelbonus.com` below in a larger font
  - Use QR error correction level L.
  - Create parent directories if missing.
- Dependencies ONLY via `design/api.md`:
  - none
