Source: `src/course/course.go`
APIs: `NewService`, `CreateCourse`, `UpdateCourse`, `ListCourses`,
`GetCourse`, `CalculateGrades`, `ExportGradesCSV`, `PublicRedemptionBoard`

# Course management and grading

Implement instructor course management and grade calculation.

## Data handled by this module

- Courses with:
  - numeric id,
  - instructor id,
  - course name,
  - grading method identifier,
  - creation timestamp,
  - update timestamp.
- Derived student participation summaries per course.

## Functional requirements

- Create a course for a logged-in instructor.
- List the instructor's courses ordered by creation time descending.
- Load one course owned by an instructor.
- Update course name and grading method.
- Produce calculated course grades ordered by student identifier ascending.
- Export the same grade data in CSV format with a header row.
- Produce a public redemption board ordered by student identifier ascending.

## Course fields

- Course name is required and limited to 120 characters.
- Course name entry should suggest including a year or semester in the UI, but
  the service only stores the provided value.
- Supported grading method identifiers:
  - `ge_q2_10`
  - `ge_q3_10`
  - `q4_10`
  - `q2_5_ge_q3_10`
  - `q2_5_q4_10`
  - `q1_5_ge_q2_10`
  - `q1_5_ge_q3_10`
  - `q1_5_q4_10`

## Grade calculation

- Input data is the count of redeemed coupons for each student within the
  course.
- Rank is `1` for the highest redemption count, with standard competition
  ranking for ties.
- Quartiles are computed over the ordered redemption counts ascending using
  nearest-rank boundaries:
  - `Q1` is the value at `ceil(0.25 * n)`,
  - `Q2` is the value at `ceil(0.50 * n)`,
  - `Q3` is the value at `ceil(0.75 * n)`,
  where `n` is the number of students with at least one redemption.
- `Q4` means the set of students whose redemption count equals the maximum
  course redemption count.
- A student with zero redemptions does not appear in the grade output.
- The grading method applies these rules:
  - `ge_q2_10`: grade 10 when count >= Q2, else 0.
  - `ge_q3_10`: grade 10 when count >= Q3, else 0.
  - `q4_10`: grade 10 when count == max count, else 0.
  - `q2_5_ge_q3_10`: grade 10 when count >= Q3, grade 5 when count == Q2,
    else 0.
  - `q2_5_q4_10`: grade 10 when count == max count, grade 5 when count == Q2,
    else 0.
  - `q1_5_ge_q2_10`: grade 10 when count >= Q2, grade 5 when count == Q1,
    else 0.
  - `q1_5_ge_q3_10`: grade 10 when count >= Q3, grade 5 when count == Q1,
    else 0.
  - `q1_5_q4_10`: grade 10 when count == max count, grade 5 when count == Q1,
    else 0.
- Final grade values are integers from 0 through 10.

## Outputs

- Grade rows contain:
  - student id,
  - redemption count,
  - grade,
  - rank.
- Public rows contain:
  - student id,
  - redemption count.
