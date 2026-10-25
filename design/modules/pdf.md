# Module: pdf

- Target source file path: `src/pdf/service.go`
- Owned APIs:
  - `GenerateCoupons`
- Internal logic:
  - create `/var/tmp/pixelbonus` if missing
  - paginate in 10x2 coupon grid on A4 with 1 cm margins
  - draw cut marks
  - render QR code with error correction level L
  - render grouped hex code and footer label
- Dependencies:
  - `logging.Log`
