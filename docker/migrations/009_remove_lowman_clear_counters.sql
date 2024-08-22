ALTER TABLE "player_stats"
DROP COLUMN "solos",
DROP COLUMN "duos",
DROP COLUMN "trios";

CREATE INDEX "idx_activity_player_is_first_clear" ON "activity_player"("is_first_clear") INCLUDE (instance_id) WHERE is_first_clear;