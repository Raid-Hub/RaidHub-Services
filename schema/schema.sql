CREATE TABLE "raid" (
    "id" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "is_sunset" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "raid_pkey" PRIMARY KEY ("id")
);
CREATE INDEX "idx_raid_definition_raid_id" ON "raid_definition"("raid_id");
CREATE INDEX "idx_raid_definition_version_id" ON "raid_definition"("version_id");

CREATE TABLE "raid_version" (
    "id" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "associated_raid_id" INTEGER,

    CONSTRAINT "raid_version_pkey" PRIMARY KEY ("id")
);
ALTER TABLE "raid_version" ADD CONSTRAINT "raid_version_associated_raid_id_fkey" FOREIGN KEY ("associated_raid_id") REFERENCES "raid"("id") ON DELETE SET NULL ON UPDATE CASCADE;

CREATE TABLE "raid_definition" (
    "hash" BIGINT NOT NULL,
    "raid_id" INTEGER NOT NULL,
    "version_id" INTEGER NOT NULL,

    CONSTRAINT "raid_definition_pkey" PRIMARY KEY ("hash")
);
ALTER TABLE "raid_definition" ADD CONSTRAINT "raid_definition_raid_id_fkey" FOREIGN KEY ("raid_id") REFERENCES "raid"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE "raid_definition" ADD CONSTRAINT "raid_definition_version_id_fkey" FOREIGN KEY ("version_id") REFERENCES "raid_version"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

CREATE TABLE "activity" (
    "instance_id" BIGINT NOT NULL,
    "raid_hash" BIGINT NOT NULL,
    "flawless" BOOLEAN,
    "completed" BOOLEAN NOT NULL,
    "fresh" BOOLEAN,
    "player_count" INTEGER NOT NULL,
    "date_started" TIMESTAMP(3) NOT NULL,
    "date_completed" TIMESTAMP(3) NOT NULL,
    "duration" INTEGER NOT NULL,
    "platform_type" INTEGER NOT NULL,

    CONSTRAINT "activity_pkey" PRIMARY KEY ("instance_id")
);
ALTER TABLE "activity" ADD CONSTRAINT "activity_raid_hash_fkey" FOREIGN KEY ("raid_hash") REFERENCES "raid_definition"("hash") ON DELETE NO ACTION ON UPDATE NO ACTION;
-- Raid Completion Leaderboard
CREATE INDEX "idx_raidhash_date_completed" ON "activity"("raid_hash", "date_completed");
-- Recent Activity
CREATE INDEX "date_index" ON "activity"("date_completed" DESC);
-- Tag Search Index
CREATE INDEX "tag_index" ON "activity"("completed", "player_count", "fresh", "flawless");
-- Speedrun Index
CREATE INDEX "speedrun_index" ON "activity"("raid_hash", "completed", "fresh", "duration" ASC);

CREATE TABLE "player" (
    "membership_id" BIGINT NOT NULL,
    "membership_type" INTEGER,
    "icon_path" TEXT,
    "display_name" TEXT,
    "bungie_global_display_name" TEXT,
    "bungie_global_display_name_code" TEXT,
    "last_seen" TIMESTAMP(3) NOT NULL,
    "clears" INTEGER NOT NULL DEFAULT 0,
    "fresh_clears" INTEGER NOT NULL DEFAULT 0,
    "sherpas" INTEGER NOT NULL DEFAULT 0,
    "sum_of_best" INTEGER,

    CONSTRAINT "player_pkey" PRIMARY KEY ("membership_id")
);
CREATE EXTENSION pg_trgm;
-- Player Search Indices
CREATE INDEX "trgm_idx_both_display_names" ON "player" USING GIN ("display_name" gin_trgm_ops, "bungie_global_display_name" gin_trgm_ops);
CREATE INDEX "trgm_idx_bungie_global_display_name" ON "player" USING GIN ("bungie_global_display_name" gin_trgm_ops);
CREATE INDEX "trgm_idx_bungie_global_name_and_code" ON "player" USING GIN ("bungie_global_display_name" gin_trgm_ops, "bungie_global_display_name_code" gin_trgm_ops);
CREATE INDEX "trgm_idx_display_name" ON "player" USING GIN ("display_name" gin_trgm_ops);

CREATE TABLE "player_activity" (
    "instance_id" BIGINT NOT NULL,
    "membership_id" BIGINT NOT NULL,
    "finished_raid" BOOLEAN NOT NULL,
    "kills" INTEGER NOT NULL DEFAULT 0,
    "assists" INTEGER NOT NULL DEFAULT 0,
    "deaths" INTEGER NOT NULL DEFAULT 0,
    "time_played_seconds" INTEGER NOT NULL DEFAULT 0,
    "class_hash" BIGINT,
    "sherpas" INTEGER NOT NULL DEFAULT 0,
    "is_first_clear" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "player_activity_instance_id_membership_id_pkey" PRIMARY KEY ("instance_id","membership_id")
);
ALTER TABLE "player_activity" ADD CONSTRAINT "player_activity_instance_id_fkey" FOREIGN KEY ("instance_id") REFERENCES "activity"("instance_id") ON DELETE CASCADE ON UPDATE NO ACTION;
ALTER TABLE "player_activity" ADD CONSTRAINT "player_activity_membership_id_fkey" FOREIGN KEY ("membership_id") REFERENCES "player"("membership_id") ON DELETE RESTRICT ON UPDATE NO ACTION;
CREATE INDEX "idx_instance_id" ON "player_activity"("instance_id");
CREATE INDEX "idx_membership_id" ON "player_activity"("membership_id");

CREATE TABLE "player_stats" (
    "membership_id" BIGINT NOT NULL,
    "raid_id" INTEGER NOT NULL,
    "clears" INTEGER NOT NULL DEFAULT 0,
    "fresh_clears" INTEGER NOT NULL DEFAULT 0,
    "sherpas" INTEGER NOT NULL DEFAULT 0,
    "trios" INTEGER NOT NULL DEFAULT 0,
    "duos" INTEGER NOT NULL DEFAULT 0,
    "solos" INTEGER NOT NULL DEFAULT 0,
    "fastest_instance_id" BIGINT,

    CONSTRAINT "player_stats_pkey" PRIMARY KEY ("membership_id","raid_id")
);
ALTER TABLE "player_stats" ADD CONSTRAINT "raid_id_fkey" FOREIGN KEY ("raid_id") REFERENCES "raid"("id") ON DELETE RESTRICT ON UPDATE NO ACTION;
ALTER TABLE "player_stats" ADD CONSTRAINT "player_membership_id_fkey" FOREIGN KEY ("membership_id") REFERENCES "player"("membership_id") ON DELETE RESTRICT ON UPDATE NO ACTION;
ALTER TABLE "player_stats" ADD CONSTRAINT "player_stats_fastest_instance_id_fkey" FOREIGN KEY ("fastest_instance_id") REFERENCES "activity"("instance_id") ON DELETE SET NULL ON UPDATE CASCADE;

CREATE TYPE "WorldFirstLeaderboardType" AS ENUM ('Normal', 'Challenge', 'Prestige', 'Master');
CREATE TABLE "leaderboard" (
    "id" TEXT NOT NULL,
    "raid_id" INTEGER NOT NULL,
    "date" TIMESTAMP(3) NOT NULL,
    "is_world_first" BOOLEAN NOT NULL DEFAULT false,
    "type" "WorldFirstLeaderboardType" NOT NULL,

    CONSTRAINT "leaderboard_pkey" PRIMARY KEY ("id")
);
ALTER TABLE "leaderboard" ADD CONSTRAINT "leaderboard_raid_id_fkey" FOREIGN KEY ("raid_id") REFERENCES "raid"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
CREATE UNIQUE INDEX "leaderboard_raid_hash_type_key" ON "leaderboard"("raid_id", "type");

CREATE TABLE "activity_leaderboard_entry" (
    "rank" INTEGER NOT NULL,
    "position" INTEGER NOT NULL,
    "leaderboard_id" TEXT NOT NULL,
    "instance_id" BIGINT NOT NULL,

    CONSTRAINT "activity_leaderboard_entry_pkey" PRIMARY KEY ("leaderboard_id", "instance_id")
);

ALTER TABLE "activity_leaderboard_entry" ADD CONSTRAINT "activity_leaderboard_entry_instance_id_fkey" FOREIGN KEY ("instance_id") REFERENCES "activity"("instance_id") ON DELETE NO ACTION ON UPDATE NO ACTION;
ALTER TABLE "activity_leaderboard_entry" ADD CONSTRAINT "activity_leaderboard_entry_leaderboard_id_fkey" FOREIGN KEY ("leaderboard_id") REFERENCES "leaderboard"("id") ON DELETE CASCADE ON UPDATE CASCADE;
CREATE INDEX "activity_leaderboard_entry_instance_id_index" ON "activity_leaderboard_entry" USING HASH ("instance_id");
CREATE INDEX "activity_leaderboard_position" ON "activity_leaderboard_entry"("leaderboard_id", "position" ASC);

-- Raw PGCR Data
CREATE TABLE "pgcr" (
    "instance_id" BIGINT NOT NULL,
    "data" BYTEA NOT NULL,

    CONSTRAINT "pgcr_pkey" PRIMARY KEY ("instance_id")
);