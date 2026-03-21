-- SQLite schema for PixelBonus.
-- The schema is bootstrap-only and safe to execute multiple times.

PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS instructors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  full_name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  verified_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS verification_tokens (
  nonce TEXT PRIMARY KEY,
  instructor_id INTEGER NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL,
  consumed_at TIMESTAMP NULL,
  FOREIGN KEY (instructor_id) REFERENCES instructors(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
  nonce TEXT PRIMARY KEY,
  instructor_id INTEGER NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL,
  consumed_at TIMESTAMP NULL,
  FOREIGN KEY (instructor_id) REFERENCES instructors(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  instructor_id INTEGER NOT NULL,
  csrf_token TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL,
  FOREIGN KEY (instructor_id) REFERENCES instructors(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS courses (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  instructor_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  grading_method TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  FOREIGN KEY (instructor_id) REFERENCES instructors(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS coupon_generation_jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  instructor_id INTEGER NOT NULL,
  course_id INTEGER NOT NULL,
  quantity INTEGER NOT NULL,
  status TEXT NOT NULL,
  error_message TEXT NULL,
  created_at TIMESTAMP NOT NULL,
  started_at TIMESTAMP NULL,
  completed_at TIMESTAMP NULL,
  FOREIGN KEY (instructor_id) REFERENCES instructors(id) ON DELETE CASCADE,
  FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS coupon_sets (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  course_id INTEGER NOT NULL,
  job_id INTEGER NOT NULL UNIQUE,
  created_at TIMESTAMP NOT NULL,
  FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
  FOREIGN KEY (job_id) REFERENCES coupon_generation_jobs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS coupons (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  coupon_set_id INTEGER NOT NULL,
  course_id INTEGER NOT NULL,
  raw_code BLOB NOT NULL,
  normalized_code TEXT NOT NULL UNIQUE,
  redeemed_by_student_id TEXT NULL,
  redeemed_at TIMESTAMP NULL,
  FOREIGN KEY (coupon_set_id) REFERENCES coupon_sets(id) ON DELETE CASCADE,
  FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS coupons_course_id_idx ON coupons(course_id);
CREATE INDEX IF NOT EXISTS coupons_redeemed_by_idx ON coupons(course_id, redeemed_by_student_id);

CREATE TABLE IF NOT EXISTS generated_files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id INTEGER NOT NULL UNIQUE,
  nonce TEXT NOT NULL UNIQUE,
  absolute_path TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL,
  FOREIGN KEY (job_id) REFERENCES coupon_generation_jobs(id) ON DELETE CASCADE
);
