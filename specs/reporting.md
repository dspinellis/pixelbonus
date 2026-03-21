Source: `src/reporting/reporting.go`
APIs: `RegisterReportingRoutes`, `CalculateGrades`

Implement instructor and public reporting.

Routes and behavior

- GET instructor course grades page ordered by student id and containing:
  - student id
  - number of redemptions
  - grade
  - rank
- GET instructor CSV export with the same ordered columns.
- GET public course redemption page ordered by student id and containing:
  - student id
  - number of redemptions

Grading rules

- Calculate grades on a 0-10 integer scale from the selected quartile-based
  grading method.
- Grades are capped at 10.
- Rank is based on redemption count, with higher counts ranking first.
- Quartile handling must be deterministic for any class size.

Visibility rules

- Instructor reports require ownership of the course.
- Public reports expose no authentication and no grade data.
