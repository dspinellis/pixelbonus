# Module Specification: coupon

- Target source file path: `src/pixelbonus/coupon/service.go`
- Owned APIs:
  - `NewService`
  - `QueueGeneration`
  - `RedeemByCode`
  - `LookupCoupon`
  - `ClaimNextJob`
  - `CompleteJob`
  - `FailJob`
  - `FileByNonce`
- Internal logic:
  - Define `GenerationJob`, `CouponToken`, `StoredFile`, `RedeemResult`, and `CouponLookup`.
  - Quantity validation: positive multiple of 20.
  - Generate coupon tokens as 20 random bytes.
  - Store raw bytes and normalized uppercase hex string.
  - Normalization removes ASCII spaces and uppercases hex.
  - Redemption validates student identifier rules from `design/global.md`.
  - Atomic redemption:
    1. start transaction
    2. load coupon by normalized code
    3. if missing -> invalid
    4. if already redeemed -> already redeemed result
    5. update coupon with student identifier and timestamp
    6. commit
  - File nonce is 16 random bytes encoded as lowercase hex.
- Dependencies ONLY via `design/api.md`:
  - none
