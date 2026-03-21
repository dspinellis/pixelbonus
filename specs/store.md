Source: `src/store/store.go`
APIs: `Open`, `Close`, `WithTx`, `CreateInstructor`, `ConfirmInstructor`,
`GetInstructorByEmail`, `UpdateInstructorPassword`, `CreatePasswordReset`,
`ConsumePasswordReset`, `CreateCourse`, `UpdateCourse`, `ListCoursesByInstructor`,
`GetCourseByID`, `CreateTokenBatch`, `InsertTokens`, `MarkBatchReady`,
`GetTokenByCode`, `RedeemToken`, `ListCourseRedemptions`,
`CreateDownloadLink`, `GetDownloadLink`

# Persistence

Implement SQLite-backed persistence and schema initialization.

## Responsibilities

- Open the SQLite database file.
- Execute the DDL statements in `schema.sql` during startup so a fresh database
  becomes ready automatically.
- Expose typed CRUD-style operations needed by the application services.
- Provide transaction support for multi-step writes such as token redemption
  and token batch creation.

## Data model requirements

- The schema must support:
  - instructors,
  - instructor confirmation tokens,
  - instructor password reset tokens,
  - instructor sessions,
  - courses,
  - token batches,
  - tokens,
  - redemptions,
  - download links.
- Enforce uniqueness for:
  - instructor email,
  - token code,
  - active session id,
  - download secret.
- Enforce that a token is redeemed at most once.

## Query requirements

- `ListCoursesByInstructor` orders rows by course creation timestamp
  descending.
- `ListCourseRedemptions` returns one row per student id within the course with
  redemption totals needed for grading and public ranking.
- `RedeemToken` must perform lookup and state change atomically to avoid
  double redemption under concurrent requests.

## Error behavior

- Return distinguishable errors for not found, duplicate values, and invalid
  state transitions where the service layer must react differently.
