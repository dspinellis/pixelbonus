# Module: redemption

- Target source file path: `src/redemption/service.go`
- Owned APIs:
  - `Redeem`
  - `LookupRedeemPage`
- Internal logic:
  - normalize manual and QR token formats
  - validate student identifier
  - atomically mark token redeemed once
  - insert redemption audit row
  - return deterministic status object
- Dependencies:
  - `database.OpenMySQL`
  - `database.WithTx`
  - `logging.Log`
  - `tokens.GetTokenByCode`
