-- MySQL 5.7+ application schema, compatible with old-schema.sql.
-- Additive only: existing tables and data are never replaced.

CREATE TABLE IF NOT EXISTS `Users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `username` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `username_canonical` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `email` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `email_canonical` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `enabled` tinyint(1) NOT NULL,
  `salt` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `password` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `last_login` datetime DEFAULT NULL,
  `locked` tinyint(1) NOT NULL,
  `expired` tinyint(1) NOT NULL,
  `expires_at` datetime DEFAULT NULL,
  `confirmation_token` varchar(255) COLLATE utf8_unicode_ci DEFAULT NULL,
  `password_requested_at` datetime DEFAULT NULL,
  `roles` longtext COLLATE utf8_unicode_ci NOT NULL COMMENT '(DC2Type:array)',
  `credentials_expired` tinyint(1) NOT NULL,
  `credentials_expire_at` datetime DEFAULT NULL,
  `name` varchar(255) COLLATE utf8_unicode_ci DEFAULT NULL,
  `surname` varchar(255) COLLATE utf8_unicode_ci DEFAULT NULL,
  `organization` varchar(255) COLLATE utf8_unicode_ci DEFAULT NULL,
  `department` varchar(255) COLLATE utf8_unicode_ci DEFAULT NULL,
  `preferred_grading_model` varchar(255) COLLATE utf8_unicode_ci DEFAULT NULL,
  `grade_multiplier` double NOT NULL,
  `max_grade` int(11) NOT NULL,
  `min_grade` int(11) NOT NULL,
  `locale` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `UNIQ_D5428AED92FC23A8` (`username_canonical`),
  UNIQUE KEY `UNIQ_D5428AEDA0D96FBF` (`email_canonical`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS `Course` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) DEFAULT NULL,
  `name` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `hashedUrl` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `UNIQ_11326A8FF2389283` (`hashedUrl`),
  KEY `IDX_11326A8FA76ED395` (`user_id`),
  CONSTRAINT `FK_11326A8FA76ED395` FOREIGN KEY (`user_id`) REFERENCES `Users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS `QrSet` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) DEFAULT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `IDX_237FC35FA76ED395` (`user_id`),
  CONSTRAINT `FK_237FC35FA76ED395` FOREIGN KEY (`user_id`) REFERENCES `Course` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS `QrRequest` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `qrset_id` int(11) DEFAULT NULL,
  `status` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `quantity` int(11) NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `IDX_413BE0BDF1623DE` (`qrset_id`),
  CONSTRAINT `FK_413BE0BDF1623DE` FOREIGN KEY (`qrset_id`) REFERENCES `QrSet` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS `QrCode` (
  `code` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `qrset_id` int(11) DEFAULT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `qrrequest_id` int(11) DEFAULT NULL,
  PRIMARY KEY (`code`),
  KEY `IDX_36189E4DF1623DE` (`qrset_id`),
  KEY `IDX_36189E4C3803015` (`qrrequest_id`),
  CONSTRAINT `FK_36189E4C3803015` FOREIGN KEY (`qrrequest_id`) REFERENCES `QrRequest` (`id`) ON DELETE CASCADE,
  CONSTRAINT `FK_36189E4DF1623DE` FOREIGN KEY (`qrset_id`) REFERENCES `QrSet` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS `Redemption` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `participantNumber` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `qrcode_code` varchar(255) COLLATE utf8_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `IDX_633C37F3160278D` (`qrcode_code`),
  CONSTRAINT `FK_633C37F3160278D` FOREIGN KEY (`qrcode_code`) REFERENCES `QrCode` (`code`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS `Tag` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(50) COLLATE utf8_unicode_ci NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS `qrset_tags` (
  `qrset_id` int(11) NOT NULL,
  `tag_id` int(11) NOT NULL,
  PRIMARY KEY (`qrset_id`,`tag_id`),
  KEY `IDX_BB0DC445DF1623DE` (`qrset_id`),
  KEY `IDX_BB0DC445BAD26311` (`tag_id`),
  CONSTRAINT `FK_BB0DC445BAD26311` FOREIGN KEY (`tag_id`) REFERENCES `Tag` (`id`),
  CONSTRAINT `FK_BB0DC445DF1623DE` FOREIGN KEY (`qrset_id`) REFERENCES `QrSet` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS pixelbonus_course_settings (
    course_id int(11) NOT NULL PRIMARY KEY,
    grading_method varchar(64) NOT NULL,
    FOREIGN KEY (course_id) REFERENCES Course(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS pixelbonus_pdf_jobs (
    request_id int(11) NOT NULL PRIMARY KEY,
    file_nonce varchar(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
    file_path varchar(1024) NOT NULL,
    failure_reason text,
    FOREIGN KEY (request_id) REFERENCES QrRequest(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS email_verification_tokens (
    token varchar(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL PRIMARY KEY,
    instructor_id int(11) NOT NULL,
    expires_at varchar(32) NOT NULL,
    consumed_at varchar(32),
    created_at varchar(32) NOT NULL,
    FOREIGN KEY (instructor_id) REFERENCES Users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    token varchar(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL PRIMARY KEY,
    instructor_id int(11) NOT NULL,
    expires_at varchar(32) NOT NULL,
    consumed_at varchar(32),
    created_at varchar(32) NOT NULL,
    FOREIGN KEY (instructor_id) REFERENCES Users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

CREATE TABLE IF NOT EXISTS sessions (
    id varchar(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL PRIMARY KEY,
    instructor_id int(11) NOT NULL,
    issued_at varchar(32) NOT NULL,
    last_seen_at varchar(32) NOT NULL,
    expires_at varchar(32) NOT NULL,
    absolute_expires_at varchar(32) NOT NULL,
    FOREIGN KEY (instructor_id) REFERENCES Users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
