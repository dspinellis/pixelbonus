# Module Specification: grade

- Target source file path: `src/pixelbonus/grade/service.go`
- Owned APIs:
  - `NewService`
  - `CourseGrades`
- Internal logic:
  - Load owned course and per-student redemption counts.
  - Ignore zero-redemption students because they are absent from data.
  - Compute quartiles with nearest-rank as defined in `design/global.md`.
  - Compute grades using piecewise linear interpolation for the selected grading method, cap at 10, and round to nearest integer with `.5` rounding up.
  - Compute competition ranks on redemption counts descending.
  - Sort output by student identifier ascending.
- Dependencies ONLY via `design/api.md`:
  - `course.GetCourse`
