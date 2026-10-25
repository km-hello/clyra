-- +goose Up
CREATE TABLE `notification_runs` (
	`id` text PRIMARY KEY NOT NULL,
	`run_date` text NOT NULL,
	`scheduled_time` text CHECK(scheduled_time IS NULL OR (length(scheduled_time) = 5 AND scheduled_time GLOB '[0-2][0-9]:[0-5][0-9]' AND substr(scheduled_time, 1, 2) <= '23')),
	`run_type` text NOT NULL,
	`channel` text DEFAULT 'telegram' NOT NULL,
	`status` text NOT NULL,
	`matched_count` integer DEFAULT 0 NOT NULL,
	`message_count` integer DEFAULT 0 NOT NULL,
	`started_at` integer NOT NULL,
	`finished_at` integer,
	`error_message` text,
	CONSTRAINT "notification_runs_type" CHECK("notification_runs"."run_type" IN ('scheduled', 'manual_test')),
	CONSTRAINT "notification_runs_schedule" CHECK((run_type = 'scheduled' AND scheduled_time IS NOT NULL) OR (run_type = 'manual_test' AND scheduled_time IS NULL)),
	CONSTRAINT "notification_runs_channel" CHECK("notification_runs"."channel" = 'telegram'),
	CONSTRAINT "notification_runs_status" CHECK("notification_runs"."status" IN ('running', 'succeeded', 'failed')),
	CONSTRAINT "notification_runs_counts" CHECK(typeof("notification_runs"."matched_count") = 'integer' AND "notification_runs"."matched_count" BETWEEN 0 AND 9007199254740991 AND typeof("notification_runs"."message_count") = 'integer' AND "notification_runs"."message_count" BETWEEN 0 AND 9007199254740991),
	CONSTRAINT "notification_runs_started" CHECK(typeof("notification_runs"."started_at") = 'integer' AND "notification_runs"."started_at" BETWEEN 0 AND 9007199254740991),
	CONSTRAINT "notification_runs_finished" CHECK(("notification_runs"."status" = 'running' AND "notification_runs"."finished_at" IS NULL) OR ("notification_runs"."status" <> 'running' AND "notification_runs"."finished_at" IS NOT NULL AND typeof("notification_runs"."finished_at") = 'integer' AND "notification_runs"."finished_at" BETWEEN "notification_runs"."started_at" AND 9007199254740991)),
	CONSTRAINT "notification_runs_error" CHECK("notification_runs"."error_message" IS NULL OR length("notification_runs"."error_message") <= 500)
);

CREATE UNIQUE INDEX `notification_runs_scheduled_unique` ON `notification_runs` (`run_date`,`scheduled_time`,`channel`) WHERE "notification_runs"."run_type" = 'scheduled';
CREATE INDEX `notification_runs_started_idx` ON `notification_runs` (`started_at`);
CREATE TABLE `settings` (
	`id` integer PRIMARY KEY DEFAULT 1 NOT NULL,
	`scheduler_enabled` integer DEFAULT 0 NOT NULL,
	`default_reminder_days` integer DEFAULT 7 NOT NULL,
	`timezone` text DEFAULT 'Asia/Singapore' NOT NULL,
	`notification_time` text DEFAULT '09:00' NOT NULL,
	`telegram_chat_id` text,
	`theme` text DEFAULT 'system' NOT NULL,
	`created_at` integer NOT NULL,
	`updated_at` integer NOT NULL,
	CONSTRAINT "settings_singleton" CHECK("settings"."id" = 1),
	CONSTRAINT "settings_scheduler_enabled" CHECK("settings"."scheduler_enabled" IN (0, 1)),
	CONSTRAINT "settings_reminder_days" CHECK(typeof("settings"."default_reminder_days") = 'integer' AND "settings"."default_reminder_days" BETWEEN 0 AND 365),
	CONSTRAINT "settings_notification_time" CHECK("settings"."notification_time" GLOB '[0-2][0-9]:[0-5][0-9]' AND substr("settings"."notification_time", 1, 2) <= '23'),
	CONSTRAINT "settings_timezone" CHECK(length(trim("settings"."timezone")) > 0),
	CONSTRAINT "settings_chat_id" CHECK("settings"."telegram_chat_id" IS NULL OR (length("settings"."telegram_chat_id") > 0 AND "settings"."telegram_chat_id" = trim("settings"."telegram_chat_id"))),
	CONSTRAINT "settings_theme" CHECK("settings"."theme" IN ('system', 'light', 'dark')),
	CONSTRAINT "settings_timestamps" CHECK(typeof("settings"."created_at") = 'integer' AND "settings"."created_at" BETWEEN 0 AND 9007199254740991 AND typeof("settings"."updated_at") = 'integer' AND "settings"."updated_at" BETWEEN 0 AND 9007199254740991)
);

CREATE TABLE `subscriptions` (
	`id` text PRIMARY KEY NOT NULL,
	`name` text NOT NULL,
	`amount_minor` integer NOT NULL,
	`currency` text NOT NULL,
	`icon` text,
	`website_url` text CHECK(website_url IS NULL OR (website_url = trim(website_url) AND length(website_url) BETWEEN 1 AND 2048)),
	`subscription_type` text NOT NULL,
	`start_date` text,
	`next_payment` text NOT NULL,
	`interval_count` integer,
	`interval_unit` text,
	`anchor_month` integer,
	`anchor_day` integer,
	`notify_enabled` integer DEFAULT true NOT NULL,
	`reminder_days` integer,
	`status` text DEFAULT 'active' NOT NULL,
	`notes` text,
	`version` integer DEFAULT 1 NOT NULL,
	`created_at` integer NOT NULL,
	`updated_at` integer NOT NULL,
	CONSTRAINT "subscriptions_name" CHECK("subscriptions"."name" = trim("subscriptions"."name") AND length("subscriptions"."name") BETWEEN 1 AND 100),
	CONSTRAINT "subscriptions_amount_minor" CHECK(typeof("subscriptions"."amount_minor") = 'integer' AND "subscriptions"."amount_minor" BETWEEN 0 AND 9007199254740991),
	CONSTRAINT "subscriptions_currency" CHECK("subscriptions"."currency" IN ('CNY', 'USD', 'EUR', 'GBP', 'JPY', 'HKD', 'TWD', 'SGD', 'AUD', 'CAD')),
	CONSTRAINT "subscriptions_notes" CHECK("subscriptions"."notes" IS NULL OR length("subscriptions"."notes") <= 2000),
	CONSTRAINT "subscriptions_type" CHECK("subscriptions"."subscription_type" IN ('one_time', 'recurring')),
	CONSTRAINT "subscriptions_status" CHECK("subscriptions"."status" IN ('active', 'inactive')),
	CONSTRAINT "subscriptions_notify" CHECK("subscriptions"."notify_enabled" IN (0, 1)),
	CONSTRAINT "subscriptions_reminder_days" CHECK("subscriptions"."reminder_days" IS NULL OR (typeof("subscriptions"."reminder_days") = 'integer' AND "subscriptions"."reminder_days" BETWEEN 0 AND 365)),
	CONSTRAINT "subscriptions_version" CHECK(typeof("subscriptions"."version") = 'integer' AND "subscriptions"."version" BETWEEN 1 AND 9007199254740991),
	CONSTRAINT "subscriptions_timestamps" CHECK(typeof("subscriptions"."created_at") = 'integer' AND "subscriptions"."created_at" BETWEEN 0 AND 9007199254740991 AND typeof("subscriptions"."updated_at") = 'integer' AND "subscriptions"."updated_at" BETWEEN 0 AND 9007199254740991),
	CONSTRAINT "subscriptions_cycle" CHECK(
    ("subscriptions"."subscription_type" = 'one_time' AND "subscriptions"."start_date" IS NULL AND "subscriptions"."interval_count" IS NULL AND "subscriptions"."interval_unit" IS NULL AND "subscriptions"."anchor_month" IS NULL AND "subscriptions"."anchor_day" IS NULL)
    OR ("subscriptions"."subscription_type" = 'recurring' AND "subscriptions"."start_date" IS NOT NULL AND "subscriptions"."start_date" < "subscriptions"."next_payment"
      AND "subscriptions"."interval_count" IS NOT NULL AND typeof("subscriptions"."interval_count") = 'integer' AND "subscriptions"."interval_count" BETWEEN 1 AND 9007199254740991
      AND "subscriptions"."interval_unit" IS NOT NULL AND "subscriptions"."interval_unit" IN ('day', 'week', 'month', 'year')
      AND (("subscriptions"."interval_unit" IN ('day', 'week') AND "subscriptions"."anchor_month" IS NULL AND "subscriptions"."anchor_day" IS NULL)
        OR ("subscriptions"."interval_unit" = 'month' AND "subscriptions"."anchor_month" IS NULL AND "subscriptions"."anchor_day" IS NOT NULL AND typeof("subscriptions"."anchor_day") = 'integer' AND "subscriptions"."anchor_day" BETWEEN 1 AND 31)
        OR ("subscriptions"."interval_unit" = 'year' AND "subscriptions"."anchor_month" IS NOT NULL AND typeof("subscriptions"."anchor_month") = 'integer' AND "subscriptions"."anchor_month" BETWEEN 1 AND 12 AND "subscriptions"."anchor_day" IS NOT NULL AND typeof("subscriptions"."anchor_day") = 'integer' AND "subscriptions"."anchor_day" BETWEEN 1 AND 31)))
  )
);

CREATE INDEX `subscriptions_reminders_idx` ON `subscriptions` (`status`,`notify_enabled`,`next_payment`);
CREATE INDEX `subscriptions_due_idx` ON `subscriptions` (`next_payment`,`id`);
CREATE TABLE `users` (
	`id` text PRIMARY KEY NOT NULL,
	`singleton_key` integer DEFAULT 1 NOT NULL,
	`username` text NOT NULL,
	`nickname` text,
	`avatar_url` text,
	`password_hash` text NOT NULL,
	`session_version` integer DEFAULT 1 NOT NULL,
	`created_at` integer NOT NULL,
	`updated_at` integer NOT NULL,
	CONSTRAINT "users_singleton" CHECK("users"."singleton_key" = 1),
	CONSTRAINT "users_username" CHECK("users"."username" NOT GLOB '*[^a-z]*' AND instr("users"."username", char(0)) = 0 AND length("users"."username") BETWEEN 4 AND 32),
	CONSTRAINT "users_password_hash" CHECK(length("users"."password_hash") > 0),
	CONSTRAINT "users_session_version" CHECK(typeof("users"."session_version") = 'integer' AND "users"."session_version" BETWEEN 1 AND 9007199254740991),
	CONSTRAINT "users_timestamps" CHECK(typeof("users"."created_at") = 'integer' AND "users"."created_at" BETWEEN 0 AND 9007199254740991 AND typeof("users"."updated_at") = 'integer' AND "users"."updated_at" BETWEEN 0 AND 9007199254740991)
);

CREATE UNIQUE INDEX `users_singleton_key_unique` ON `users` (`singleton_key`);
CREATE UNIQUE INDEX `users_username_unique` ON `users` (`username`);

INSERT INTO settings (id, created_at, updated_at)
VALUES (1, CAST(unixepoch('subsec') * 1000 AS INTEGER), CAST(unixepoch('subsec') * 1000 AS INTEGER));
