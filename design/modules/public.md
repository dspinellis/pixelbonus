# Module: public

- Target source file path: `src/public/service.go`
- Owned APIs:
  - `ListPublicRedemptions`
- Internal logic:
  - resolve public nonce
  - aggregate redemptions by student id
  - order rows by student id
- Dependencies:
  - `database.OpenMySQL`
  - `logging.Log`
