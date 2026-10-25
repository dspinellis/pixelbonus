# Module: grades

- Target source file path: `src/grades/service.go`
- Owned APIs:
  - `ComputeCourseGrades`
  - `RenderCSV`
- Internal logic:
  - aggregate redemptions by student id
  - compute quartiles
  - map counts to grades for each grading method
  - compute ranks
  - order rows by student id
  - render deterministic CSV
- Dependencies:
  - `courses.GetCourse`
  - `database.OpenMySQL`
  - `logging.Log`
