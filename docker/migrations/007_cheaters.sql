ALTER TABLE "player"
ADD COLUMN "cheat_level" SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE "activity"
ADD COLUMN "cheat_override" BOOLEAN NOT NULL DEFAULT False;

ALTER TABLE "player"
ADD COLUMN "is_private" BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN "history_last_crawled" TIMESTAMP(3);