# Module: session

- Target source file path: `src/session/store.go`
- Owned APIs:
  - `Create`
  - `Validate`
  - `Destroy`
- Internal logic:
  - generate random session id and signing payload
  - persist session metadata
  - validate signature and expiry
  - rotate last-seen time
- Dependencies:
  - `database.OpenMySQL`
  - `database.WithTx`
  - `logging.Log`
