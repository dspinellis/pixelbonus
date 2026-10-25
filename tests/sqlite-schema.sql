-- MySQL 5.7+ application schema, compatible with old-schema.sql.
-- Additive only: existing tables and data are never replaced.

CREATE TABLE IF NOT EXISTS `Users` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `username` varchar(255) NOT NULL,
  `username_canonical` varchar(255) NOT NULL,
  `email` varchar(255) NOT NULL,
  `email_canonical` varchar(255) NOT NULL,
  `enabled` tinyint(1) NOT NULL,
  `salt` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `last_login` datetime DEFAULT NULL,
  `locked` tinyint(1) NOT NULL,
  `expired` tinyint(1) NOT NULL,
  `expires_at` datetime DEFAULT NULL,
  `confirmation_token` varchar(255) DEFAULT NULL,
  `password_requested_at` datetime DEFAULT NULL,
  `roles` longtext NOT NULL,
  `credentials_expired` tinyint(1) NOT NULL,
  `credentials_expire_at` datetime DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `surname` varchar(255) DEFAULT NULL,
  `organization` varchar(255) DEFAULT NULL,
  `department` varchar(255) DEFAULT NULL,
  `preferred_grading_model` varchar(255) DEFAULT NULL,
  `grade_multiplier` double NOT NULL,
  `max_grade` int(11) NOT NULL,
  `min_grade` int(11) NOT NULL,
  `locale` varchar(255) NOT NULL,
  UNIQUE (`username_canonical`),
  UNIQUE (`email_canonical`)
);

CREATE TABLE IF NOT EXISTS `Course` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` int(11) DEFAULT NULL,
  `name` varchar(255) NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `hashedUrl` varchar(255) NOT NULL,
  UNIQUE (`hashedUrl`),
  CONSTRAINT `FK_11326A8FA76ED395` FOREIGN KEY (`user_id`) REFERENCES `Users` (`id`) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS `QrSet` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` int(11) DEFAULT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  CONSTRAINT `FK_237FC35FA76ED395` FOREIGN KEY (`user_id`) REFERENCES `Course` (`id`) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS `QrRequest` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `qrset_id` int(11) DEFAULT NULL,
  `status` varchar(255) NOT NULL,
  `quantity` int(11) NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  CONSTRAINT `FK_413BE0BDF1623DE` FOREIGN KEY (`qrset_id`) REFERENCES `QrSet` (`id`) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS `QrCode` (
  `code` varchar(255) NOT NULL,
  `qrset_id` int(11) DEFAULT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `qrrequest_id` int(11) DEFAULT NULL,
  PRIMARY KEY (`code`),
  CONSTRAINT `FK_36189E4C3803015` FOREIGN KEY (`qrrequest_id`) REFERENCES `QrRequest` (`id`) ON DELETE CASCADE,
  CONSTRAINT `FK_36189E4DF1623DE` FOREIGN KEY (`qrset_id`) REFERENCES `QrSet` (`id`) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS `Redemption` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `participantNumber` varchar(255) NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `qrcode_code` varchar(255) DEFAULT NULL,
  CONSTRAINT `FK_633C37F3160278D` FOREIGN KEY (`qrcode_code`) REFERENCES `QrCode` (`code`) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS `Tag` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` varchar(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS `qrset_tags` (
  `qrset_id` int(11) NOT NULL,
  `tag_id` int(11) NOT NULL,
  PRIMARY KEY (`qrset_id`,`tag_id`),
  CONSTRAINT `FK_BB0DC445BAD26311` FOREIGN KEY (`tag_id`) REFERENCES `Tag` (`id`),
  CONSTRAINT `FK_BB0DC445DF1623DE` FOREIGN KEY (`qrset_id`) REFERENCES `QrSet` (`id`) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS pixelbonus_course_settings (
    course_id int(11) NOT NULL PRIMARY KEY,
    grading_method varchar(64) NOT NULL,
    FOREIGN KEY (course_id) REFERENCES Course(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS pixelbonus_pdf_jobs (
    request_id int(11) NOT NULL PRIMARY KEY,
    file_nonce varchar(64) NOT NULL UNIQUE,
    file_path varchar(1024) NOT NULL,
    failure_reason text,
    FOREIGN KEY (request_id) REFERENCES QrRequest(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS email_verification_tokens (
    token varchar(64) NOT NULL PRIMARY KEY,
    instructor_id int(11) NOT NULL,
    expires_at varchar(32) NOT NULL,
    consumed_at varchar(32),
    created_at varchar(32) NOT NULL,
    FOREIGN KEY (instructor_id) REFERENCES Users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    token varchar(64) NOT NULL PRIMARY KEY,
    instructor_id int(11) NOT NULL,
    expires_at varchar(32) NOT NULL,
    consumed_at varchar(32),
    created_at varchar(32) NOT NULL,
    FOREIGN KEY (instructor_id) REFERENCES Users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sessions (
    id varchar(64) NOT NULL PRIMARY KEY,
    instructor_id int(11) NOT NULL,
    issued_at varchar(32) NOT NULL,
    last_seen_at varchar(32) NOT NULL,
    expires_at varchar(32) NOT NULL,
    absolute_expires_at varchar(32) NOT NULL,
    FOREIGN KEY (instructor_id) REFERENCES Users(id) ON DELETE CASCADE
);
