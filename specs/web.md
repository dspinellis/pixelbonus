Source: `src/web/web.go`
APIs: `NewServer`, `Routes`, `RenderPrivacy`, `SetLanguage`,
`RequireInstructor`, `CurrentSession`

# Web delivery and localization

Implement HTTP routing, page rendering, localization, and request/response
integration.

## Responsibilities

- Register all public and instructor routes described in
  `specs/integration.md`.
- Render HTML pages using server-side templates.
- Expose a privacy page suitable for Greek and EU GDPR requirements.
- Resolve interface language from cookie, browser preferences, and default
  English behavior.
- Offer language switching between English and Greek.
- Manage secure session cookies and authenticated route protection by using
  the `auth` package.
- Add CSRF validation to state-changing forms.
- Render user-facing success and error messages for redemption, login,
  registration, password reset, and instructor actions.

## Page-level requirements

- Every page uses title `PixelBonus`.
- The header and footer elements required by `specs/global.md` appear on all
  pages.
- The home page presents separate student and instructor sections with the
  supplied descriptions.
- Instructor course list rows expose actions for:
  - generating QR coupons,
  - viewing grades,
  - downloading CSV,
  - viewing the public page,
  - opening settings.

## Response requirements

- Form validation errors are displayed near the relevant action.
- Successful POST requests use redirect-after-post where appropriate.
- CSV export returns `text/csv`.
- PDF download returns `application/pdf`.
