Source: `src/pdf/pdf.go`
APIs: `NewGenerator`, `RenderCoupons`

# PDF generation

Implement printable coupon PDF generation.

## Responsibilities

- Generate an A4 PDF file for a token batch.
- Lay out 20 coupons per page in 10 rows by 2 columns.
- Draw cut marks suitable for cutting the coupons apart.
- Render each coupon with:
  - a QR code image for the redeem URL,
  - the formatted coupon code in a fixed-width font,
  - the text `pixelbonus.com` below the code in a larger font.

## Output requirements

- Return the generated file path and metadata needed by the caller.
- Use readable font sizing and spacing so the coupon contents remain legible
  after printing.
