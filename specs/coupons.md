Source: `src/coupons/coupons.go`
APIs: `RegisterCouponRoutes`, `CouponService`

Implement coupon token batch generation, PDF production, and secure file
download.

Routes and behavior

- GET/POST route for an instructor to request a number of QR code tokens for a
  selected owned course.
- Generate 160-bit random token identifiers.
- Store each token associated with the course, the batch creation timestamp, and
  an unused state.
- Start asynchronous A4 PDF generation after a batch request is accepted.
- Email the instructor a secure long random-string download URL once the file is
  ready.
- GET download route that serves the generated PDF to anybody possessing the
  secret URL, with no expiry.

PDF output

- 20 coupons per page arranged in 10 rows by 2 columns.
- Include cut marks appropriate for manual splitting.
- Each coupon contains:
  - QR code encoding `https://pixelbonus.com/r?c=CODE`
  - printed code in ten space-separated groups of four hexadecimal characters in
    a fixed-width font
  - `pixelbonus.com` below the code in a larger font
- Use no QR error correction.

Operational notes

- Persist download-link metadata in the database.
- Persist generated files in a local directory so the secure link works across
  process restarts.
