package grades

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"math"
	"sort"

	"pixelbonus/src/courses"
)

type Service struct {
	DB      *sql.DB
	Courses *courses.Service
}

type GradeRow struct {
	StudentID   string
	Redemptions int
	Grade       int
	Rank        int
}

type CourseGrades struct {
	Course courses.Course
	Rows   []GradeRow
	Q1     int
	Q2     int
	Q3     int
	Q4     int
}

type point struct{ x, y int }

func (s *Service) ComputeCourseGrades(ctx context.Context, instructorID int64, courseID int64) (CourseGrades, error) {
	course, err := s.Courses.GetCourse(ctx, instructorID, courseID)
	if err != nil {
		return CourseGrades{}, err
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT r.participantNumber, COUNT(*) FROM Redemption r JOIN QrCode q ON q.code = r.qrcode_code LEFT JOIN QrRequest request ON request.id = q.qrrequest_id JOIN QrSet qs ON qs.id = COALESCE(q.qrset_id, request.qrset_id) WHERE qs.user_id = ? GROUP BY r.participantNumber ORDER BY COUNT(*) DESC, r.participantNumber", courseID)
	if err != nil {
		return CourseGrades{}, err
	}
	defer rows.Close()
	var out []GradeRow
	var counts []int
	for rows.Next() {
		var row GradeRow
		if err := rows.Scan(&row.StudentID, &row.Redemptions); err != nil {
			return CourseGrades{}, err
		}
		out = append(out, row)
		counts = append(counts, row.Redemptions)
	}
	sort.Ints(counts)
	if len(out) == 0 {
		return CourseGrades{Course: course}, nil
	}
	q1, q2, q3, q4 := quartiles(counts)
	ranks := ranks(out)
	for i := range out {
		out[i].Rank = ranks[out[i].StudentID]
		out[i].Grade = gradeFor(course.GradingMethod, out[i].Redemptions, q1, q2, q3, q4)
	}
	return CourseGrades{Course: course, Rows: out, Q1: q1, Q2: q2, Q3: q3, Q4: q4}, nil
}

func RenderCSV(rows []GradeRow) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"student_id", "redemptions", "grade", "rank"}); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := w.Write([]string{row.StudentID, itoa(row.Redemptions), itoa(row.Grade), itoa(row.Rank)}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func quartiles(values []int) (int, int, int, int) {
	if len(values) == 0 {
		return 0, 0, 0, 0
	}
	return nearestRank(values, 25), nearestRank(values, 50), nearestRank(values, 75), values[len(values)-1]
}

func nearestRank(values []int, pct int) int {
	idx := int(math.Ceil(float64(pct) / 100 * float64(len(values))))
	if idx < 1 {
		idx = 1
	}
	return values[idx-1]
}

func ranks(rows []GradeRow) map[string]int {
	copyRows := make([]GradeRow, len(rows))
	copy(copyRows, rows)
	sort.Slice(copyRows, func(i, j int) bool {
		if copyRows[i].Redemptions == copyRows[j].Redemptions {
			return copyRows[i].StudentID < copyRows[j].StudentID
		}
		return copyRows[i].Redemptions > copyRows[j].Redemptions
	})
	out := map[string]int{}
	rank := 0
	prev := -1
	for i, row := range copyRows {
		if row.Redemptions != prev {
			rank = i + 1
			prev = row.Redemptions
		}
		out[row.StudentID] = rank
	}
	return out
}

func gradeFor(method string, value int, q1 int, q2 int, q3 int, q4 int) int {
	var pts []point
	switch method {
	case "0=0,>=Q2=10":
		pts = []point{{0, 0}, {q2, 10}}
	case "0=0,>=Q3=10":
		pts = []point{{0, 0}, {q3, 10}}
	case "0=0,Q4=10":
		pts = []point{{0, 0}, {q4, 10}}
	case "Q2=5,>=Q3=10":
		pts = []point{{0, 0}, {q2, 5}, {q3, 10}}
	case "Q2=5,Q4=10":
		pts = []point{{0, 0}, {q2, 5}, {q4, 10}}
	case "Q1=5,>=Q2=10":
		pts = []point{{0, 0}, {q1, 5}, {q2, 10}}
	case "Q1=5,>=Q3=10":
		pts = []point{{0, 0}, {q1, 5}, {q3, 10}}
	case "Q1=5,Q4=10":
		pts = []point{{0, 0}, {q1, 5}, {q4, 10}}
	default:
		panic("unknown grading method")
	}
	score := interpolate(pts, value)
	if score > 10 {
		score = 10
	}
	if score < 0 {
		score = 0
	}
	return score
}

func interpolate(pts []point, v int) int {
	if v <= pts[0].x {
		return pts[0].y
	}
	for i := 1; i < len(pts); i++ {
		if v <= pts[i].x {
			x0, y0 := pts[i-1].x, pts[i-1].y
			x1, y1 := pts[i].x, pts[i].y
			if x1 == x0 {
				return y1
			}
			f := float64(v-x0) / float64(x1-x0)
			return int(math.Floor(float64(y0) + (float64(y1-y0) * f) + 0.5))
		}
	}
	return pts[len(pts)-1].y
}

func itoa(v int) string {
	return fmt.Sprintf("%d", v)
}
