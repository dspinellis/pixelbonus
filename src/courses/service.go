package courses

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"pixelbonus/src/database"
)

var ValidGradingMethods = []string{
	"0=0,>=Q2=10",
	"0=0,>=Q3=10",
	"0=0,Q4=10",
	"Q2=5,>=Q3=10",
	"Q2=5,Q4=10",
	"Q1=5,>=Q2=10",
	"Q1=5,>=Q3=10",
	"Q1=5,Q4=10",
}

type Course struct {
	ID            int64
	InstructorID  int64
	Name          string
	GradingMethod string
	PublicNonce   string
	CreatedAt     string
}

type CreateCourseInput struct {
	Name          string
	GradingMethod string
}

type UpdateCourseInput struct {
	ID            int64
	Name          string
	GradingMethod string
}

type Service struct {
	DB     *sql.DB
	Now    func() time.Time
	Random func([]byte) (int, error)
}

func (s *Service) init() {
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.Random == nil {
		s.Random = rand.Read
	}
}

func (s *Service) CreateCourse(ctx context.Context, instructorID int64, input CreateCourseInput) (Course, error) {
	s.init()
	if err := validate(input.Name, input.GradingMethod); err != nil {
		return Course{}, err
	}
	buf := make([]byte, 16)
	if _, err := s.Random(buf); err != nil {
		return Course{}, err
	}
	publicNonce := base64.RawURLEncoding.EncodeToString(buf)
	now := database.DateTime(s.Now())
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Course{}, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "INSERT INTO Course(user_id,name,hashedUrl,created_at,updated_at) VALUES(?,?,?,?,?)", instructorID, strings.TrimSpace(input.Name), publicNonce, now, now)
	if err != nil {
		return Course{}, err
	}
	id, _ := res.LastInsertId()
	if _, err := tx.ExecContext(ctx, "INSERT INTO pixelbonus_course_settings(course_id,grading_method) VALUES(?,?)", id, input.GradingMethod); err != nil {
		return Course{}, err
	}
	if err := tx.Commit(); err != nil {
		return Course{}, err
	}
	return Course{ID: id, InstructorID: instructorID, Name: strings.TrimSpace(input.Name), GradingMethod: input.GradingMethod, PublicNonce: publicNonce, CreatedAt: now}, nil
}

func (s *Service) ListCourses(ctx context.Context, instructorID int64) ([]Course, error) {
	rows, err := s.DB.QueryContext(ctx, SelectCourse+" WHERE c.user_id = ? ORDER BY c.created_at DESC, c.id DESC", instructorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Course
	for rows.Next() {
		var c Course
		if err := rows.Scan(&c.ID, &c.InstructorID, &c.Name, &c.GradingMethod, &c.PublicNonce, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) GetCourse(ctx context.Context, instructorID int64, courseID int64) (Course, error) {
	var c Course
	err := s.DB.QueryRowContext(ctx, SelectCourse+" WHERE c.id = ? AND c.user_id = ?", courseID, instructorID).Scan(&c.ID, &c.InstructorID, &c.Name, &c.GradingMethod, &c.PublicNonce, &c.CreatedAt)
	return c, err
}

func (s *Service) UpdateCourse(ctx context.Context, instructorID int64, input UpdateCourseInput) (Course, error) {
	if err := validate(input.Name, input.GradingMethod); err != nil {
		return Course{}, err
	}
	s.init()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Course{}, err
	}
	defer tx.Rollback()
	var id int64
	if err := tx.QueryRowContext(ctx, "SELECT id FROM Course WHERE id = ? AND user_id = ?", input.ID, instructorID).Scan(&id); err != nil {
		return Course{}, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE Course SET name = ?, updated_at = ? WHERE id = ? AND user_id = ?", strings.TrimSpace(input.Name), database.DateTime(s.Now()), input.ID, instructorID); err != nil {
		return Course{}, err
	}
	// Delete/insert is portable and keeps legacy tables unchanged.
	if _, err := tx.ExecContext(ctx, "DELETE FROM pixelbonus_course_settings WHERE course_id = ?", id); err != nil {
		return Course{}, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO pixelbonus_course_settings(course_id,grading_method) VALUES(?,?)", id, input.GradingMethod); err != nil {
		return Course{}, err
	}
	if err := tx.Commit(); err != nil {
		return Course{}, err
	}
	return s.GetCourse(ctx, instructorID, input.ID)
}

func validate(name string, method string) error {
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > 120 {
		return errors.New("invalid course name")
	}
	for _, item := range ValidGradingMethods {
		if item == method {
			return nil
		}
	}
	return errors.New("invalid grading method")
}

// Legacy courses have no per-course grading model. Use the owner's recognized
// preference, otherwise the application's default; never overwrite legacy values.
const SelectCourse = "SELECT c.id,COALESCE(c.user_id,0),c.name,COALESCE(s.grading_method, CASE WHEN u.preferred_grading_model IN ('0=0,>=Q2=10','0=0,>=Q3=10','0=0,Q4=10','Q2=5,>=Q3=10','Q2=5,Q4=10','Q1=5,>=Q2=10','Q1=5,>=Q3=10','Q1=5,Q4=10') THEN u.preferred_grading_model ELSE '0=0,>=Q2=10' END),c.hashedUrl,c.created_at FROM Course c LEFT JOIN pixelbonus_course_settings s ON s.course_id = c.id LEFT JOIN Users u ON u.id = c.user_id"
