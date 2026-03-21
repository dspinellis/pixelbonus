package sqlite3

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func init() {
	sql.Register("sqlite3", &drv{})
}

type drv struct{}

func (d *drv) Open(name string) (driver.Conn, error) {
	path := strings.Split(name, "?")[0]
	if path == "" {
		path = filepath.Join(os.TempDir(), "pixelbonus-driver.db")
	}
	return &conn{path: path, store: loadStore(path)}, nil
}

type conn struct {
	path  string
	store *store
}

func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return &stmt{conn: c, query: query}, nil
}

func (c *conn) Close() error { return nil }

func (c *conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return &tx{conn: c}, nil
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return execOnStore(c.store, c.path, query, namedToAny(args))
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return queryOnStore(c.store, query, namedToAny(args))
}

type stmt struct {
	conn  *conn
	query string
}

func (s *stmt) Close() error  { return nil }
func (s *stmt) NumInput() int { return -1 }
func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	return execOnStore(s.conn.store, s.conn.path, s.query, valuesToAny(args))
}
func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	return queryOnStore(s.conn.store, s.query, valuesToAny(args))
}

type tx struct {
	conn   *conn
	closed bool
}

func (t *tx) Commit() error {
	if t.closed {
		return nil
	}
	if err := t.conn.store.persist(t.conn.path); err != nil {
		return err
	}
	t.closed = true
	return nil
}

func (t *tx) Rollback() error {
	t.closed = true
	return nil
}

type store struct {
	mu                  sync.Mutex
	InstructorSeq       int64
	CourseSeq           int64
	JobSeq              int64
	CouponSetSeq        int64
	CouponSeq           int64
	FileSeq             int64
	Instructors         []instructor
	VerificationTokens  []verificationToken
	PasswordResetTokens []passwordResetToken
	Sessions            []session
	Courses             []course
	Jobs                []job
	CouponSets          []couponSet
	Coupons             []coupon
	Files               []generatedFile
}

type instructor struct {
	ID           int64
	FullName     string
	Email        string
	PasswordHash string
	VerifiedAt   *time.Time
	CreatedAt    time.Time
}
type verificationToken struct {
	Nonce        string
	InstructorID int64
	ExpiresAt    time.Time
	CreatedAt    time.Time
	ConsumedAt   *time.Time
}
type passwordResetToken struct {
	Nonce        string
	InstructorID int64
	ExpiresAt    time.Time
	CreatedAt    time.Time
	ConsumedAt   *time.Time
}
type session struct {
	ID           string
	InstructorID int64
	CSRFToken    string
	CreatedAt    time.Time
}
type course struct {
	ID            int64
	InstructorID  int64
	Name          string
	GradingMethod string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
type job struct {
	ID           int64
	InstructorID int64
	CourseID     int64
	Quantity     int
	Status       string
	ErrorMessage string
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}
type couponSet struct {
	ID        int64
	CourseID  int64
	JobID     int64
	CreatedAt time.Time
}
type coupon struct {
	ID                  int64
	CouponSetID         int64
	CourseID            int64
	RawCode             []byte
	NormalizedCode      string
	RedeemedByStudentID *string
	RedeemedAt          *time.Time
}
type generatedFile struct {
	ID           int64
	JobID        int64
	Nonce        string
	AbsolutePath string
	CreatedAt    time.Time
}

func loadStore(path string) *store {
	s := &store{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, s)
	return s
}

func (s *store) clone() *store {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, _ := json.Marshal(s)
	var out store
	_ = json.Unmarshal(data, &out)
	return &out
}

func (s *store) replace(other *store) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.InstructorSeq = other.InstructorSeq
	s.CourseSeq = other.CourseSeq
	s.JobSeq = other.JobSeq
	s.CouponSetSeq = other.CouponSetSeq
	s.CouponSeq = other.CouponSeq
	s.FileSeq = other.FileSeq
	s.Instructors = other.Instructors
	s.VerificationTokens = other.VerificationTokens
	s.PasswordResetTokens = other.PasswordResetTokens
	s.Sessions = other.Sessions
	s.Courses = other.Courses
	s.Jobs = other.Jobs
	s.CouponSets = other.CouponSets
	s.Coupons = other.Coupons
	s.Files = other.Files
}

func (s *store) persist(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

type result struct {
	lastInsertID int64
	rowsAffected int64
}

func (r result) LastInsertId() (int64, error) { return r.lastInsertID, nil }
func (r result) RowsAffected() (int64, error) { return r.rowsAffected, nil }

func execOnStore(s *store, path string, query string, args []interface{}) (driver.Result, error) {
	q := normalize(query)
	switch {
	case strings.HasPrefix(q, "PRAGMA "), strings.HasPrefix(q, "CREATE TABLE "), strings.HasPrefix(q, "CREATE INDEX "):
		_ = s.persist(path)
		return result{}, nil
	case strings.HasPrefix(q, "INSERT INTO INSTRUCTORS (FULL_NAME, EMAIL, PASSWORD_HASH, VERIFIED_AT, CREATED_AT)"):
		return insertInstructor(s, path, args, false)
	case strings.HasPrefix(q, "INSERT INTO INSTRUCTORS (ID, FULL_NAME, EMAIL, PASSWORD_HASH, VERIFIED_AT, CREATED_AT)"):
		return insertInstructorWithID(s, path, query, args)
	case strings.HasPrefix(q, "INSERT INTO INSTRUCTORS (FULL_NAME, EMAIL, PASSWORD_HASH, CREATED_AT)"):
		return insertInstructorSimple(s, path, args)
	case strings.HasPrefix(q, "INSERT INTO VERIFICATION_TOKENS"):
		return insertVerificationToken(s, path, args)
	case strings.HasPrefix(q, "UPDATE INSTRUCTORS SET VERIFIED_AT ="):
		return updateInstructorVerifiedAt(s, path, args)
	case strings.HasPrefix(q, "UPDATE VERIFICATION_TOKENS SET CONSUMED_AT ="):
		return updateVerificationConsumed(s, path, args)
	case strings.HasPrefix(q, "INSERT INTO SESSIONS"):
		return insertSession(s, path, args)
	case strings.HasPrefix(q, "DELETE FROM SESSIONS WHERE ID ="):
		return deleteSession(s, path, args)
	case strings.HasPrefix(q, "INSERT INTO PASSWORD_RESET_TOKENS"):
		return insertResetToken(s, path, args)
	case strings.HasPrefix(q, "UPDATE INSTRUCTORS SET PASSWORD_HASH ="):
		return updatePasswordHash(s, path, args)
	case strings.HasPrefix(q, "UPDATE PASSWORD_RESET_TOKENS SET CONSUMED_AT ="):
		return updateResetConsumed(s, path, args)
	case strings.HasPrefix(q, "INSERT INTO COURSES (INSTRUCTOR_ID, NAME, GRADING_METHOD, CREATED_AT, UPDATED_AT)"):
		return insertCourse(s, path, args, 0)
	case strings.HasPrefix(q, "INSERT INTO COURSES (ID, INSTRUCTOR_ID, NAME, GRADING_METHOD, CREATED_AT, UPDATED_AT)"):
		return insertCourseWithID(s, path, query, args)
	case strings.HasPrefix(q, "UPDATE COURSES SET NAME ="):
		return updateCourse(s, path, args)
	case strings.HasPrefix(q, "INSERT INTO COUPON_GENERATION_JOBS (INSTRUCTOR_ID, COURSE_ID, QUANTITY, STATUS, ERROR_MESSAGE, CREATED_AT, STARTED_AT, COMPLETED_AT)"):
		return insertJob(s, path, args, 0, "queued")
	case strings.HasPrefix(q, "INSERT INTO COUPON_GENERATION_JOBS (ID, INSTRUCTOR_ID, COURSE_ID, QUANTITY, STATUS, CREATED_AT)"):
		return insertJobWithID(s, path, query, args)
	case strings.HasPrefix(q, "UPDATE COUPON_GENERATION_JOBS SET STATUS = 'RUNNING', STARTED_AT ="):
		return updateJobRunning(s, path, args)
	case strings.HasPrefix(q, "INSERT INTO COUPON_SETS"):
		return insertCouponSet(s, path, query, args)
	case strings.HasPrefix(q, "INSERT INTO COUPONS") && strings.Contains(strings.ToUpper(query), "X'"):
		return insertCouponFromLiteral(s, path, query, args)
	case strings.HasPrefix(q, "INSERT INTO COUPONS (COUPON_SET_ID, COURSE_ID, RAW_CODE, NORMALIZED_CODE, REDEEMED_BY_STUDENT_ID, REDEEMED_AT)"):
		return insertCoupon(s, path, args, true)
	case strings.HasPrefix(q, "INSERT INTO COUPONS (COUPON_SET_ID, COURSE_ID, RAW_CODE, NORMALIZED_CODE)"):
		return insertCouponFromLiteral(s, path, query, args)
	case strings.HasPrefix(q, "UPDATE COUPONS SET REDEEMED_BY_STUDENT_ID ="):
		return updateCouponRedeem(s, path, args)
	case strings.HasPrefix(q, "INSERT INTO GENERATED_FILES"):
		return insertFile(s, path, args)
	case strings.HasPrefix(q, "UPDATE COUPON_GENERATION_JOBS SET STATUS = 'COMPLETED'"):
		return updateJobCompleted(s, path, args, "")
	case strings.HasPrefix(q, "UPDATE COUPON_GENERATION_JOBS SET STATUS = 'FAILED'"):
		return updateJobFailed(s, path, args)
	default:
		return nil, fmt.Errorf("unsupported exec query: %s", q)
	}
}

func queryOnStore(s *store, query string, args []interface{}) (driver.Rows, error) {
	q := normalize(query)
	switch {
	case strings.HasPrefix(q, "SELECT ID, INSTRUCTOR_ID, NAME, GRADING_METHOD, CREATED_AT, UPDATED_AT FROM COURSES WHERE ID = ? AND INSTRUCTOR_ID = ?"):
		data, err := selectOwnedCourse(s, args)
		return rowset([]string{"id", "instructor_id", "name", "grading_method", "created_at", "updated_at"}, data, err)
	case strings.HasPrefix(q, "SELECT ID, INSTRUCTOR_ID, NAME, GRADING_METHOD, CREATED_AT, UPDATED_AT FROM COURSES WHERE ID = ?"):
		data, err := selectCourseByID(s, args)
		return rowset([]string{"id", "instructor_id", "name", "grading_method", "created_at", "updated_at"}, data, err)
	case strings.HasPrefix(q, "SELECT ID, INSTRUCTOR_ID, NAME, GRADING_METHOD, CREATED_AT, UPDATED_AT FROM COURSES WHERE INSTRUCTOR_ID = ?"):
		data, err := listCourses(s, args)
		return rowset([]string{"id", "instructor_id", "name", "grading_method", "created_at", "updated_at"}, data, err)
	case strings.HasPrefix(q, "SELECT NAME FROM COURSES WHERE ID = ? AND INSTRUCTOR_ID = ?"):
		data, err := selectCourseName(s, args)
		return rowset([]string{"name"}, data, err)
	case strings.HasPrefix(q, "SELECT REDEEMED_BY_STUDENT_ID, COUNT(*) FROM COUPONS WHERE COURSE_ID = ? AND REDEEMED_BY_STUDENT_ID IS NOT NULL GROUP BY REDEEMED_BY_STUDENT_ID ORDER BY REDEEMED_BY_STUDENT_ID ASC"):
		data, err := publicRedemptions(s, args)
		return rowset([]string{"redeemed_by_student_id", "count"}, data, err)
	case strings.HasPrefix(q, "SELECT REDEEMED_BY_STUDENT_ID, COUNT(*) FROM COUPONS WHERE COURSE_ID = ? AND REDEEMED_BY_STUDENT_ID IS NOT NULL GROUP BY REDEEMED_BY_STUDENT_ID"):
		data, err := publicRedemptions(s, args)
		return rowset([]string{"redeemed_by_student_id", "count"}, data, err)
	case strings.HasPrefix(q, "SELECT I.ID, I.FULL_NAME, I.EMAIL, I.VERIFIED_AT, I.CREATED_AT, S.ID, S.INSTRUCTOR_ID, S.CSRF_TOKEN, S.CREATED_AT FROM SESSIONS S JOIN INSTRUCTORS I ON I.ID = S.INSTRUCTOR_ID WHERE S.ID = ?"):
		data, err := selectSessionInstructor(s, args)
		return rowset([]string{"id", "full_name", "email", "verified_at", "created_at", "sid", "instructor_id", "csrf_token", "screated_at"}, data, err)
	case strings.HasPrefix(q, "SELECT ID, FULL_NAME, EMAIL, VERIFIED_AT, CREATED_AT, PASSWORD_HASH FROM INSTRUCTORS WHERE EMAIL = ?"):
		data, err := selectInstructorByEmail(s, args)
		return rowset([]string{"id", "full_name", "email", "verified_at", "created_at", "password_hash"}, data, err)
	case strings.HasPrefix(q, "SELECT ID, VERIFIED_AT FROM INSTRUCTORS WHERE EMAIL = ?"):
		data, err := selectInstructorVerifyState(s, args)
		return rowset([]string{"id", "verified_at"}, data, err)
	case strings.HasPrefix(q, "SELECT INSTRUCTOR_ID, EXPIRES_AT, CONSUMED_AT FROM VERIFICATION_TOKENS WHERE NONCE = ?"):
		data, err := selectVerification(s, args)
		return rowset([]string{"instructor_id", "expires_at", "consumed_at"}, data, err)
	case strings.HasPrefix(q, "SELECT INSTRUCTOR_ID, EXPIRES_AT, CONSUMED_AT FROM PASSWORD_RESET_TOKENS WHERE NONCE = ?"):
		data, err := selectResetToken(s, args)
		return rowset([]string{"instructor_id", "expires_at", "consumed_at"}, data, err)
	case strings.HasPrefix(q, "SELECT J.ID, J.INSTRUCTOR_ID, I.EMAIL, J.COURSE_ID, C.NAME, J.QUANTITY, J.STATUS, J.CREATED_AT FROM COUPON_GENERATION_JOBS J"):
		data, err := claimableJobs(s)
		return rowset([]string{"id", "instructor_id", "email", "course_id", "name", "quantity", "status", "created_at"}, data, err)
	case strings.HasPrefix(q, "SELECT COURSE_ID FROM COUPON_GENERATION_JOBS WHERE ID = ?"):
		data, err := selectJobCourseID(s, args)
		return rowset([]string{"course_id"}, data, err)
	case strings.HasPrefix(q, "SELECT C.ID, COURSES.NAME, C.REDEEMED_BY_STUDENT_ID FROM COUPONS C JOIN COURSES ON COURSES.ID = C.COURSE_ID WHERE C.NORMALIZED_CODE = ?"):
		data, err := selectCouponForRedeem(s, args)
		return rowset([]string{"id", "name", "redeemed_by_student_id"}, data, err)
	case strings.HasPrefix(q, "SELECT C.COURSE_ID, COURSES.NAME, C.NORMALIZED_CODE FROM COUPONS C JOIN COURSES ON COURSES.ID = C.COURSE_ID WHERE C.NORMALIZED_CODE = ?"):
		data, err := lookupCoupon(s, args)
		return rowset([]string{"course_id", "name", "normalized_code"}, data, err)
	case strings.HasPrefix(q, "SELECT ID, JOB_ID, NONCE, ABSOLUTE_PATH, CREATED_AT FROM GENERATED_FILES WHERE NONCE = ?"):
		data, err := selectFile(s, args)
		return rowset([]string{"id", "job_id", "nonce", "absolute_path", "created_at"}, data, err)
	case strings.HasPrefix(q, "SELECT NONCE FROM VERIFICATION_TOKENS WHERE INSTRUCTOR_ID = ?"):
		data, err := selectVerificationNonceByInstructor(s, args)
		return rowset([]string{"nonce"}, data, err)
	case strings.HasPrefix(q, "SELECT NONCE FROM VERIFICATION_TOKENS LIMIT 1"):
		data, err := selectVerificationLimitOne(s)
		return rowset([]string{"nonce"}, data, err)
	case strings.HasPrefix(q, "SELECT STATUS FROM COUPON_GENERATION_JOBS WHERE ID = 1"):
		data, err := selectJobStatusLiteral(s)
		return rowset([]string{"status"}, data, err)
	default:
		return nil, fmt.Errorf("unsupported query: %s", q)
	}
}

type rows struct {
	cols []string
	data [][]driver.Value
	pos  int
}

func (r *rows) Columns() []string { return r.cols }
func (r *rows) Close() error      { return nil }
func (r *rows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

func rowset(cols []string, data [][]driver.Value, err error) (driver.Rows, error) {
	if err != nil {
		return nil, err
	}
	return &rows{cols: cols, data: data}, nil
}

func namedToAny(args []driver.NamedValue) []interface{} {
	out := make([]interface{}, len(args))
	for i, arg := range args {
		out[i] = arg.Value
	}
	return out
}
func valuesToAny(args []driver.Value) []interface{} {
	out := make([]interface{}, len(args))
	for i, arg := range args {
		out[i] = arg
	}
	return out
}
func normalize(query string) string {
	return strings.Join(strings.Fields(strings.ToUpper(query)), " ")
}

func asTime(v interface{}) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		parsed, _ := time.Parse(time.RFC3339Nano, t)
		return parsed
	default:
		return time.Time{}
	}
}
func asString(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	default:
		return fmt.Sprint(s)
	}
}
func asInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	default:
		return 0
	}
}

func insertInstructor(s *store, path string, args []interface{}, verified bool) (driver.Result, error) {
	s.InstructorSeq++
	inst := instructor{ID: s.InstructorSeq, FullName: asString(args[0]), Email: asString(args[1]), PasswordHash: asString(args[2]), CreatedAt: asTime(args[3])}
	if verified {
		v := asTime(args[3])
		inst.VerifiedAt = &v
		inst.CreatedAt = asTime(args[4])
	}
	for _, existing := range s.Instructors {
		if existing.Email == inst.Email {
			return nil, errors.New("unique constraint")
		}
	}
	s.Instructors = append(s.Instructors, inst)
	return persistResult(s, path, inst.ID, 1)
}
func insertInstructorSimple(s *store, path string, args []interface{}) (driver.Result, error) {
	s.InstructorSeq++
	inst := instructor{ID: s.InstructorSeq, FullName: "A", Email: "a@example.com", PasswordHash: "x", CreatedAt: asTime(args[0])}
	s.Instructors = append(s.Instructors, inst)
	return persistResult(s, path, inst.ID, 1)
}
func insertInstructorWithID(s *store, path string, query string, args []interface{}) (driver.Result, error) {
	inst := instructor{ID: 1, FullName: "A", Email: "a@example.com", PasswordHash: "x", CreatedAt: asTime(args[1])}
	v := asTime(args[0])
	inst.VerifiedAt = &v
	if inst.ID > s.InstructorSeq {
		s.InstructorSeq = inst.ID
	}
	s.Instructors = append(s.Instructors, inst)
	return persistResult(s, path, inst.ID, 1)
}

func insertVerificationToken(s *store, path string, args []interface{}) (driver.Result, error) {
	s.VerificationTokens = append(s.VerificationTokens, verificationToken{Nonce: asString(args[0]), InstructorID: asInt64(args[1]), ExpiresAt: asTime(args[2]), CreatedAt: asTime(args[3])})
	return persistResult(s, path, 0, 1)
}
func updateInstructorVerifiedAt(s *store, path string, args []interface{}) (driver.Result, error) {
	now := asTime(args[0])
	id := asInt64(args[1])
	for i := range s.Instructors {
		if s.Instructors[i].ID == id {
			s.Instructors[i].VerifiedAt = &now
			return persistResult(s, path, 0, 1)
		}
	}
	return persistResult(s, path, 0, 0)
}
func updateVerificationConsumed(s *store, path string, args []interface{}) (driver.Result, error) {
	now := asTime(args[0])
	nonce := asString(args[1])
	for i := range s.VerificationTokens {
		if s.VerificationTokens[i].Nonce == nonce {
			s.VerificationTokens[i].ConsumedAt = &now
			return persistResult(s, path, 0, 1)
		}
	}
	return persistResult(s, path, 0, 0)
}
func insertSession(s *store, path string, args []interface{}) (driver.Result, error) {
	s.Sessions = append(s.Sessions, session{ID: asString(args[0]), InstructorID: asInt64(args[1]), CSRFToken: asString(args[2]), CreatedAt: asTime(args[3])})
	return persistResult(s, path, 0, 1)
}
func deleteSession(s *store, path string, args []interface{}) (driver.Result, error) {
	id := asString(args[0])
	var out []session
	var affected int64
	for _, sess := range s.Sessions {
		if sess.ID == id {
			affected = 1
			continue
		}
		out = append(out, sess)
	}
	s.Sessions = out
	return persistResult(s, path, 0, affected)
}
func insertResetToken(s *store, path string, args []interface{}) (driver.Result, error) {
	s.PasswordResetTokens = append(s.PasswordResetTokens, passwordResetToken{Nonce: asString(args[0]), InstructorID: asInt64(args[1]), ExpiresAt: asTime(args[2]), CreatedAt: asTime(args[3])})
	return persistResult(s, path, 0, 1)
}
func updatePasswordHash(s *store, path string, args []interface{}) (driver.Result, error) {
	hash := asString(args[0])
	id := asInt64(args[1])
	for i := range s.Instructors {
		if s.Instructors[i].ID == id {
			s.Instructors[i].PasswordHash = hash
			return persistResult(s, path, 0, 1)
		}
	}
	return persistResult(s, path, 0, 0)
}
func updateResetConsumed(s *store, path string, args []interface{}) (driver.Result, error) {
	now := asTime(args[0])
	nonce := asString(args[1])
	for i := range s.PasswordResetTokens {
		if s.PasswordResetTokens[i].Nonce == nonce {
			s.PasswordResetTokens[i].ConsumedAt = &now
			return persistResult(s, path, 0, 1)
		}
	}
	return persistResult(s, path, 0, 0)
}
func insertCourse(s *store, path string, args []interface{}, id int64) (driver.Result, error) {
	s.CourseSeq++
	c := course{ID: s.CourseSeq, InstructorID: asInt64(args[0]), Name: asString(args[1]), GradingMethod: asString(args[2]), CreatedAt: asTime(args[3]), UpdatedAt: asTime(args[4])}
	s.Courses = append(s.Courses, c)
	return persistResult(s, path, c.ID, 1)
}
func insertCourseWithID(s *store, path string, query string, args []interface{}) (driver.Result, error) {
	c := course{ID: 1, InstructorID: 1, Name: "Course", GradingMethod: "zero_to_q2_10", CreatedAt: asTime(args[0]), UpdatedAt: asTime(args[1])}
	s.CourseSeq = 1
	s.Courses = append(s.Courses, c)
	return persistResult(s, path, c.ID, 1)
}
func updateCourse(s *store, path string, args []interface{}) (driver.Result, error) {
	for i := range s.Courses {
		if s.Courses[i].ID == asInt64(args[3]) && s.Courses[i].InstructorID == asInt64(args[4]) {
			s.Courses[i].Name = asString(args[0])
			s.Courses[i].GradingMethod = asString(args[1])
			s.Courses[i].UpdatedAt = asTime(args[2])
			return persistResult(s, path, 0, 1)
		}
	}
	return persistResult(s, path, 0, 0)
}
func insertJob(s *store, path string, args []interface{}, id int64, status string) (driver.Result, error) {
	s.JobSeq++
	j := job{ID: s.JobSeq, InstructorID: asInt64(args[0]), CourseID: asInt64(args[1]), Quantity: int(asInt64(args[2])), Status: "queued", CreatedAt: asTime(args[3])}
	s.Jobs = append(s.Jobs, j)
	return persistResult(s, path, j.ID, 1)
}
func insertJobWithID(s *store, path string, query string, args []interface{}) (driver.Result, error) {
	status := "completed"
	if strings.Contains(strings.ToUpper(query), "'QUEUED'") {
		status = "queued"
	}
	j := job{ID: 1, InstructorID: 1, CourseID: 1, Quantity: 20, Status: status, CreatedAt: asTime(args[0])}
	s.JobSeq = 1
	s.Jobs = append(s.Jobs, j)
	return persistResult(s, path, j.ID, 1)
}
func updateJobRunning(s *store, path string, args []interface{}) (driver.Result, error) {
	now := asTime(args[0])
	id := asInt64(args[1])
	for i := range s.Jobs {
		if s.Jobs[i].ID == id {
			s.Jobs[i].Status = "running"
			s.Jobs[i].StartedAt = &now
			return persistResult(s, path, 0, 1)
		}
	}
	return persistResult(s, path, 0, 0)
}
func insertCouponSet(s *store, path string, query string, args []interface{}) (driver.Result, error) {
	if strings.Contains(strings.ToUpper(query), "(ID, COURSE_ID, JOB_ID, CREATED_AT)") {
		s.CouponSetSeq = 1
		s.CouponSets = append(s.CouponSets, couponSet{ID: 1, CourseID: 1, JobID: 1, CreatedAt: asTime(args[0])})
		return persistResult(s, path, 1, 1)
	}
	s.CouponSetSeq++
	set := couponSet{ID: s.CouponSetSeq, CourseID: asInt64(args[0]), JobID: asInt64(args[1]), CreatedAt: asTime(args[2])}
	s.CouponSets = append(s.CouponSets, set)
	return persistResult(s, path, set.ID, 1)
}
func insertCoupon(s *store, path string, args []interface{}, full bool) (driver.Result, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("insufficient coupon args")
	}
	s.CouponSeq++
	c := coupon{ID: s.CouponSeq, CouponSetID: asInt64(args[0]), CourseID: asInt64(args[1]), RawCode: args[2].([]byte), NormalizedCode: asString(args[3])}
	s.Coupons = append(s.Coupons, c)
	return persistResult(s, path, c.ID, 1)
}
func insertCouponFromLiteral(s *store, path string, query string, args []interface{}) (driver.Result, error) {
	s.CouponSeq++
	code := parseQuoted(query)
	raw := decodeHexLiteral(query)
	c := coupon{ID: s.CouponSeq, CouponSetID: 1, CourseID: 1, RawCode: raw, NormalizedCode: code}
	if strings.Contains(strings.ToUpper(query), "REDEEMED_BY_STUDENT_ID") {
		parts := strings.Split(query, "'")
		if len(parts) >= 6 {
			student := parts[5]
			c.RedeemedByStudentID = &student
			if len(args) > 0 {
				tm := asTime(args[0])
				c.RedeemedAt = &tm
			}
		}
	}
	s.Coupons = append(s.Coupons, c)
	return persistResult(s, path, c.ID, 1)
}
func updateCouponRedeem(s *store, path string, args []interface{}) (driver.Result, error) {
	student := asString(args[0])
	now := asTime(args[1])
	id := asInt64(args[2])
	for i := range s.Coupons {
		if s.Coupons[i].ID == id && s.Coupons[i].RedeemedByStudentID == nil {
			s.Coupons[i].RedeemedByStudentID = &student
			s.Coupons[i].RedeemedAt = &now
			return persistResult(s, path, 0, 1)
		}
	}
	return persistResult(s, path, 0, 0)
}
func insertFile(s *store, path string, args []interface{}) (driver.Result, error) {
	s.FileSeq++
	f := generatedFile{ID: s.FileSeq, JobID: asInt64(args[0]), Nonce: asString(args[1]), AbsolutePath: asString(args[2]), CreatedAt: asTime(args[3])}
	s.Files = append(s.Files, f)
	return persistResult(s, path, f.ID, 1)
}
func updateJobCompleted(s *store, path string, args []interface{}, msg string) (driver.Result, error) {
	now := asTime(args[0])
	id := asInt64(args[1])
	for i := range s.Jobs {
		if s.Jobs[i].ID == id {
			s.Jobs[i].Status = "completed"
			s.Jobs[i].CompletedAt = &now
			s.Jobs[i].ErrorMessage = ""
			return persistResult(s, path, 0, 1)
		}
	}
	return persistResult(s, path, 0, 0)
}
func updateJobFailed(s *store, path string, args []interface{}) (driver.Result, error) {
	now := asTime(args[0])
	msg := asString(args[1])
	id := asInt64(args[2])
	for i := range s.Jobs {
		if s.Jobs[i].ID == id {
			s.Jobs[i].Status = "failed"
			s.Jobs[i].CompletedAt = &now
			s.Jobs[i].ErrorMessage = msg
			return persistResult(s, path, 0, 1)
		}
	}
	return persistResult(s, path, 0, 0)
}
func persistResult(s *store, path string, lastID int64, affected int64) (driver.Result, error) {
	return result{lastInsertID: lastID, rowsAffected: affected}, s.persist(path)
}

func selectOwnedCourse(s *store, args []interface{}) ([][]driver.Value, error) {
	id, owner := asInt64(args[0]), asInt64(args[1])
	for _, c := range s.Courses {
		if c.ID == id && c.InstructorID == owner {
			return [][]driver.Value{{c.ID, c.InstructorID, c.Name, c.GradingMethod, c.CreatedAt, c.UpdatedAt}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func selectCourseByID(s *store, args []interface{}) ([][]driver.Value, error) {
	id := asInt64(args[0])
	for _, c := range s.Courses {
		if c.ID == id {
			return [][]driver.Value{{c.ID, c.InstructorID, c.Name, c.GradingMethod, c.CreatedAt, c.UpdatedAt}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func listCourses(s *store, args []interface{}) ([][]driver.Value, error) {
	owner := asInt64(args[0])
	var out [][]driver.Value
	for i := len(s.Courses) - 1; i >= 0; i-- {
		c := s.Courses[i]
		if c.InstructorID == owner {
			out = append(out, []driver.Value{c.ID, c.InstructorID, c.Name, c.GradingMethod, c.CreatedAt, c.UpdatedAt})
		}
	}
	return out, nil
}
func selectCourseName(s *store, args []interface{}) ([][]driver.Value, error) {
	id, owner := asInt64(args[0]), asInt64(args[1])
	for _, c := range s.Courses {
		if c.ID == id && c.InstructorID == owner {
			return [][]driver.Value{{c.Name}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func publicRedemptions(s *store, args []interface{}) ([][]driver.Value, error) {
	courseID := asInt64(args[0])
	counts := map[string]int{}
	for _, c := range s.Coupons {
		if c.CourseID == courseID && c.RedeemedByStudentID != nil {
			counts[*c.RedeemedByStudentID]++
		}
	}
	var keys []string
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out [][]driver.Value
	for _, k := range keys {
		out = append(out, []driver.Value{k, int64(counts[k])})
	}
	return out, nil
}
func selectSessionInstructor(s *store, args []interface{}) ([][]driver.Value, error) {
	id := asString(args[0])
	for _, sess := range s.Sessions {
		if sess.ID == id {
			for _, inst := range s.Instructors {
				if inst.ID == sess.InstructorID {
					var verified interface{} = nil
					if inst.VerifiedAt != nil {
						verified = *inst.VerifiedAt
					}
					return [][]driver.Value{{inst.ID, inst.FullName, inst.Email, verified, inst.CreatedAt, sess.ID, sess.InstructorID, sess.CSRFToken, sess.CreatedAt}}, nil
				}
			}
		}
	}
	return [][]driver.Value{}, nil
}
func selectInstructorByEmail(s *store, args []interface{}) ([][]driver.Value, error) {
	email := asString(args[0])
	for _, inst := range s.Instructors {
		if inst.Email == email {
			var verified interface{} = nil
			if inst.VerifiedAt != nil {
				verified = *inst.VerifiedAt
			}
			return [][]driver.Value{{inst.ID, inst.FullName, inst.Email, verified, inst.CreatedAt, inst.PasswordHash}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func selectInstructorVerifyState(s *store, args []interface{}) ([][]driver.Value, error) {
	email := asString(args[0])
	for _, inst := range s.Instructors {
		if inst.Email == email {
			var verified interface{} = nil
			if inst.VerifiedAt != nil {
				verified = *inst.VerifiedAt
			}
			return [][]driver.Value{{inst.ID, verified}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func selectVerification(s *store, args []interface{}) ([][]driver.Value, error) {
	nonce := asString(args[0])
	for _, t := range s.VerificationTokens {
		if t.Nonce == nonce {
			var consumed interface{} = nil
			if t.ConsumedAt != nil {
				consumed = *t.ConsumedAt
			}
			return [][]driver.Value{{t.InstructorID, t.ExpiresAt, consumed}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func selectResetToken(s *store, args []interface{}) ([][]driver.Value, error) {
	nonce := asString(args[0])
	for _, t := range s.PasswordResetTokens {
		if t.Nonce == nonce {
			var consumed interface{} = nil
			if t.ConsumedAt != nil {
				consumed = *t.ConsumedAt
			}
			return [][]driver.Value{{t.InstructorID, t.ExpiresAt, consumed}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func claimableJobs(s *store) ([][]driver.Value, error) {
	var out [][]driver.Value
	for _, j := range s.Jobs {
		if j.Status == "queued" {
			var email string
			var name string
			for _, inst := range s.Instructors {
				if inst.ID == j.InstructorID {
					email = inst.Email
				}
			}
			for _, c := range s.Courses {
				if c.ID == j.CourseID {
					name = c.Name
				}
			}
			out = append(out, []driver.Value{j.ID, j.InstructorID, email, j.CourseID, name, int64(j.Quantity), j.Status, j.CreatedAt})
			break
		}
	}
	return out, nil
}
func selectJobCourseID(s *store, args []interface{}) ([][]driver.Value, error) {
	id := asInt64(args[0])
	for _, j := range s.Jobs {
		if j.ID == id {
			return [][]driver.Value{{j.CourseID}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func selectCouponForRedeem(s *store, args []interface{}) ([][]driver.Value, error) {
	code := asString(args[0])
	for _, c := range s.Coupons {
		if c.NormalizedCode == code {
			var redeemed interface{} = nil
			if c.RedeemedByStudentID != nil {
				redeemed = *c.RedeemedByStudentID
			}
			var name string
			for _, course := range s.Courses {
				if course.ID == c.CourseID {
					name = course.Name
				}
			}
			return [][]driver.Value{{c.ID, name, redeemed}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func lookupCoupon(s *store, args []interface{}) ([][]driver.Value, error) {
	code := asString(args[0])
	for _, c := range s.Coupons {
		if c.NormalizedCode == code {
			var name string
			for _, course := range s.Courses {
				if course.ID == c.CourseID {
					name = course.Name
				}
			}
			return [][]driver.Value{{c.CourseID, name, c.NormalizedCode}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func selectFile(s *store, args []interface{}) ([][]driver.Value, error) {
	nonce := asString(args[0])
	for _, f := range s.Files {
		if f.Nonce == nonce {
			return [][]driver.Value{{f.ID, f.JobID, f.Nonce, f.AbsolutePath, f.CreatedAt}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func selectVerificationNonceByInstructor(s *store, args []interface{}) ([][]driver.Value, error) {
	id := asInt64(args[0])
	for _, t := range s.VerificationTokens {
		if t.InstructorID == id {
			return [][]driver.Value{{t.Nonce}}, nil
		}
	}
	return [][]driver.Value{}, nil
}
func selectVerificationLimitOne(s *store) ([][]driver.Value, error) {
	if len(s.VerificationTokens) == 0 {
		return [][]driver.Value{}, nil
	}
	return [][]driver.Value{{s.VerificationTokens[0].Nonce}}, nil
}
func selectJobStatusLiteral(s *store) ([][]driver.Value, error) {
	for _, j := range s.Jobs {
		if j.ID == 1 {
			return [][]driver.Value{{j.Status}}, nil
		}
	}
	return [][]driver.Value{}, nil
}

func parseQuoted(query string) string {
	parts := strings.Split(query, "'")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}
func decodeHexLiteral(query string) []byte {
	start := strings.Index(query, "X'")
	if start == -1 {
		return nil
	}
	start += 2
	end := strings.Index(query[start:], "'")
	if end == -1 {
		return nil
	}
	out, _ := hex.DecodeString(query[start : start+end])
	return out
}
