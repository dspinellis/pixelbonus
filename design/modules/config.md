# Module: config

- Target source file path: `src/config/config.go`
- Owned APIs:
  - `Load`
- Internal logic:
  - parse environment map
  - parse optional config file as `key=value`
  - apply defaults
  - load `PrivacyPolicyURLs` by language ID (`privacy_policy_urls.en` and
    `privacy_policy_urls.el`), defaulting each entry to `/privacy`
  - validate port and normalize directories
- Dependencies:
  - none beyond `design/api.md`
