Source: `src/token/token.go`
APIs: `NewService`, `CreateBatch`, `NormalizeCode`, `FormatCode`,
`EncodeRedeemURL`, `ResolveDownload`

# Token generation and instructor downloads

Implement generation of coupon tokens, printable formatting, PDF production,
and instructor download links.

## Data handled by this module

- Token batches with:
  - numeric id,
  - course id,
  - creation timestamp,
  - requested count,
  - generated PDF path,
  - ready timestamp.
- Tokens with:
  - numeric id,
  - batch id,
  - course id,
  - normalized 40-hex-character code,
  - unused or redeemed state.
- Download links with:
  - random secret,
  - batch id,
  - PDF path,
  - creation timestamp.

## Functional requirements

- Generate an instructor-requested number of new coupon tokens for a course.
- Each token must contain 160 random bits and be represented internally as
  40 uppercase hexadecimal characters.
- Store generated tokens as unused and associated with the selected course and
  the newly created batch.
- Produce an A4 PDF containing 20 coupons per page arranged in 10 rows and
  two columns with cut marks.
- Each coupon must contain:
  - a QR code whose payload is the redeem URL;
  - the token printed as ten groups of four hexadecimal characters in a
    fixed-width font;
  - the text `pixelbonus.com` beneath the printed code in a larger font.
- After the PDF is ready, create a permanent download secret and notify the
  instructor by email with the download URL.

## Validation

- Requested token count must be an integer from 1 through 10000.
- `NormalizeCode` must:
  - remove spaces,
  - uppercase letters,
  - require exactly 40 hexadecimal characters,
  - reject any other characters.
- `FormatCode` must output ten groups of four characters separated by spaces.

## Download behavior

- Any holder of the secret URL may download the file.
- The link does not expire automatically.
- Missing secrets return a not-found result.
