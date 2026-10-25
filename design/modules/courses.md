# Module: courses

- Target source file path: `src/courses/service.go`
- Owned APIs:
  - `CreateCourse`
  - `ListCourses`
  - `GetCourse`
  - `UpdateCourse`
- Internal logic:
  - validate course name
  - validate grading method against the fixed enum
  - create public nonce
  - enforce instructor ownership
- Dependencies:
  - `database.OpenMySQL`
  - `database.WithTx`
  - `logging.Log`
