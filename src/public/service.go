package public

import (
	"context"
	"database/sql"

	"pixelbonus/src/courses"
)

type Row struct {
	StudentID   string
	Redemptions int
}

type CourseView struct {
	Course courses.Course
	Rows   []Row
}

type Service struct {
	DB *sql.DB
}

func (s *Service) ListPublicRedemptions(ctx context.Context, publicNonce string) (CourseView, error) {
	var course courses.Course
	if err := s.DB.QueryRowContext(ctx, courses.SelectCourse+" WHERE c.hashedUrl = ?", publicNonce).Scan(&course.ID, &course.InstructorID, &course.Name, &course.GradingMethod, &course.PublicNonce, &course.CreatedAt); err != nil {
		return CourseView{}, err
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT r.participantNumber, COUNT(*) FROM Redemption r JOIN QrCode q ON q.code = r.qrcode_code LEFT JOIN QrRequest request ON request.id = q.qrrequest_id JOIN QrSet qs ON qs.id = COALESCE(q.qrset_id, request.qrset_id) WHERE qs.user_id = ? GROUP BY r.participantNumber ORDER BY r.participantNumber", course.ID)
	if err != nil {
		return CourseView{}, err
	}
	defer rows.Close()
	out := CourseView{Course: course}
	for rows.Next() {
		var row Row
		if err := rows.Scan(&row.StudentID, &row.Redemptions); err != nil {
			return CourseView{}, err
		}
		out.Rows = append(out.Rows, row)
	}
	return out, rows.Err()
}
