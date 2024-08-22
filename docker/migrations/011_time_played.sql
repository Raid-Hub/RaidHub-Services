ALTER TABLE "player_stats" ADD COLUMN "total_time_played_seconds" INTEGER NOT NULL DEFAULT 0;

ALTER TABLE "player" ADD COLUMN "total_time_played_seconds" INTEGER NOT NULL DEFAULT 0;