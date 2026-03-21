Source: `src/courses/courses.go`
APIs: `RegisterCourseRoutes`, `CourseService`

Implement instructor course management.

Routes and behavior

- GET instructor dashboard listing the instructor's existing courses, most
  recently created first, with icons/links for:
  - new QR coupon batch
  - grades
  - CSV export
  - public redemption page
  - settings
- GET/POST new course creation with course name and grading method.
- GET/POST course settings update with course name and grading method.

Data rules

- A course belongs to one instructor.
- Course names are free text, with guidance to include year or semester.
- Supported grading methods are exactly the quartile-based methods listed in the
  original specifications.
- The course service must also provide a public read-only lookup by course id
  for use by public reporting and redemption displays.

UI

- Instructor dashboard and forms are localized.
- Course action links use readable URLs containing the course id.
