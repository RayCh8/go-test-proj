-- +goose Up
CREATE TABLE IF NOT EXISTS `records` (
	`id`   	     varchar(255),
	`the_num`    INTEGER,
	`the_str`    varchar(255),
	`created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	`updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose Down
DROP TABLE records;