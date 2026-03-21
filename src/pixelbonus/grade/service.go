package grade

import (
	"context"
	"database/sql"
	"math"
	"sort"

	"pixelbonus/src/pixelbonus/course"
)

type Service struct {
	db *sql.DB
}

type GradeRow struct {
	StudentID   string
	Redemptions int
	Grade       int
	Rank        int
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CourseGrades(ctx context.Context, instructorID int64, courseID int64) (course.Course, []GradeRow, error) {
	var c course.Course
	err := s.db.QueryRowContext(ctx, `
		SELECT id, instructor_id, name, grading_method, created_at, updated_at
		FROM courses
		WHERE id = ? AND instructor_id = ?
	`, courseID, instructorID).Scan(&c.ID, &c.InstructorID, &c.Name, &c.GradingMethod, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return course.Course{}, nil, course.ErrNotFound
	}
	if err != nil {
		return course.Course{}, nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT redeemed_by_student_id, COUNT(*)
		FROM coupons
		WHERE course_id = ? AND redeemed_by_student_id IS NOT NULL
		GROUP BY redeemed_by_student_id
	`, courseID)
	if err != nil {
		return course.Course{}, nil, err
	}
	defer rows.Close()

	var grades []GradeRow
	var counts []int
	for rows.Next() {
		var row GradeRow
		if err := rows.Scan(&row.StudentID, &row.Redemptions); err != nil {
			return course.Course{}, nil, err
		}
		grades = append(grades, row)
		counts = append(counts, row.Redemptions)
	}
	if err := rows.Err(); err != nil {
		return course.Course{}, nil, err
	}

	sort.Ints(counts)
	q1 := quartile(counts, 0.25)
	q2 := quartile(counts, 0.50)
	q3 := quartile(counts, 0.75)
	q4 := 0
	if len(counts) > 0 {
		q4 = counts[len(counts)-1]
	}

	sort.Slice(grades, func(i, j int) bool {
		if grades[i].Redemptions == grades[j].Redemptions {
			return grades[i].StudentID < grades[j].StudentID
		}
		return grades[i].Redemptions > grades[j].Redemptions
	})
	for i := range grades {
		if i == 0 {
			grades[i].Rank = 1
		} else if grades[i].Redemptions == grades[i-1].Redemptions {
			grades[i].Rank = grades[i-1].Rank
		} else {
			grades[i].Rank = i + 1
		}
		grades[i].Grade = computeGrade(c.GradingMethod, grades[i].Redemptions, q1, q2, q3, q4)
	}
	sort.Slice(grades, func(i, j int) bool { return grades[i].StudentID < grades[j].StudentID })
	return c, grades, nil
}

func quartile(values []int, p float64) int {
	if len(values) == 0 {
		return 0
	}
	index := int(math.Ceil(p*float64(len(values)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func computeGrade(method string, count int, q1 int, q2 int, q3 int, q4 int) int {
	score := 0.0
	switch method {
	case "zero_to_q2_10":
		score = linear(count, 0, 0, q2, 10)
	case "zero_to_q3_10":
		score = linear(count, 0, 0, q3, 10)
	case "zero_to_q4_10":
		score = linear(count, 0, 0, q4, 10)
	case "q2_5_to_q3_10":
		score = piecewise(count, q2, 5, q3, 10)
	case "q2_5_to_q4_10":
		score = piecewise(count, q2, 5, q4, 10)
	case "q1_5_to_q2_10":
		score = piecewise(count, q1, 5, q2, 10)
	case "q1_5_to_q3_10":
		score = piecewise(count, q1, 5, q3, 10)
	case "q1_5_to_q4_10":
		score = piecewise(count, q1, 5, q4, 10)
	}
	if score > 10 {
		score = 10
	}
	return int(math.Floor(score + 0.5))
}

func linear(x int, x1 int, y1 float64, x2 int, y2 float64) float64 {
	if x2 <= x1 {
		if x >= x2 {
			return y2
		}
		return y1
	}
	if x <= x1 {
		return y1
	}
	if x >= x2 {
		return y2
	}
	return y1 + (float64(x-x1)/float64(x2-x1))*(y2-y1)
}

func piecewise(x int, x1 int, y1 float64, x2 int, y2 float64) float64 {
	if x <= 0 {
		return 0
	}
	if x <= x1 {
		return linear(x, 0, 0, x1, y1)
	}
	return linear(x, x1, y1, x2, y2)
}
