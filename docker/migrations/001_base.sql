CREATE TABLE "raid" (
    "id" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "is_sunset" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "raid_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "raid_version" (
    "id" INTEGER NOT NULL PRIMARY KEY,
    "name" TEXT NOT NULL,
    "associated_raid_id" INTEGER,

    CONSTRAINT "raid_version_associated_raid_id_fkey" FOREIGN KEY ("associated_raid_id") REFERENCES "raid"("id") ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE TABLE "raid_definition" (
    "hash" BIGINT NOT NULL PRIMARY KEY,
    "raid_id" INTEGER NOT NULL,
    "version_id" INTEGER NOT NULL,

    CONSTRAINT "raid_definition_raid_id_fkey" FOREIGN KEY ("raid_id") REFERENCES "raid"("id") ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT "raid_definition_version_id_fkey" FOREIGN KEY ("version_id") REFERENCES "raid_version"("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE INDEX "idx_raid_definition_raid_id" ON "raid_definition"("raid_id");
CREATE INDEX "idx_raid_definition_version_id" ON "raid_definition"("version_id");

CREATE TABLE "activity" (
    "instance_id" BIGINT NOT NULL PRIMARY KEY,
    "raid_hash" BIGINT NOT NULL,
    "flawless" BOOLEAN,
    "completed" BOOLEAN NOT NULL,
    "fresh" BOOLEAN,
    "player_count" INTEGER NOT NULL,
    "date_started" TIMESTAMP(3) NOT NULL,
    "date_completed" TIMESTAMP(3) NOT NULL,
    "duration" INTEGER NOT NULL,
    "platform_type" INTEGER NOT NULL,

    CONSTRAINT "activity_raid_hash_fkey" FOREIGN KEY ("raid_hash") REFERENCES "raid_definition"("hash") ON DELETE NO ACTION ON UPDATE NO ACTION
);


-- Raid Completion Leaderboard
CREATE INDEX "hash_date_completed_index_partial" ON "activity"("hash", "date_completed") WHERE "completed";
-- Tag Search Index
CREATE INDEX "tag_index_partial" ON "activity"("hash", "player_count", "fresh", "flawless") WHERE "completed" AND ("player_count" <= 3 OR "flawless");
-- Speedrun Index
CREATE INDEX "speedrun_index_partial" ON "activity"("hash", "duration") WHERE "completed" AND "fresh";

CREATE TABLE "player" (
    "membership_id" BIGINT NOT NULL PRIMARY KEY,
    "membership_type" INTEGER,
    "icon_path" TEXT,
    "display_name" TEXT,
    "bungie_global_display_name" TEXT,
    "bungie_global_display_name_code" TEXT,
    "bungie_name" TEXT GENERATED ALWAYS AS (
        CASE
            WHEN "bungie_global_display_name" IS NOT NULL AND "bungie_global_display_name_code" IS NOT NULL THEN
                "bungie_global_display_name" || '#' || "bungie_global_display_name_code"
            ELSE
                "display_name"
        END
    ) STORED, 
    "last_seen" TIMESTAMP(3) NOT NULL,
    "clears" INTEGER NOT NULL DEFAULT 0,
    "fresh_clears" INTEGER NOT NULL DEFAULT 0,
    "sherpas" INTEGER NOT NULL DEFAULT 0,
    "sum_of_best" INTEGER,
    "last_crawled" TIMESTAMP(3),
    "_search_score" NUMERIC GENERATED ALWAYS AS (
        sqrt("clears" + 1) * power(2, GREATEST(0, EXTRACT(EPOCH FROM ("last_seen" - TIMESTAMP '2017-08-27'))) / 20000000)
    ) STORED
);

CREATE INDEX "primary_search_idx" ON "player"(lower("bungie_name") text_pattern_ops, "_search_score" DESC);
CREATE INDEX "secondary_search_idx" ON "player"(lower("display_name") text_pattern_ops, "_search_score" DESC);

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

    CONSTRAINT "player_activity_instance_id_membership_id_pkey" PRIMARY KEY ("instance_id","membership_id"),

    CONSTRAINT "player_activity_instance_id_fkey" FOREIGN KEY ("instance_id") REFERENCES "activity"("instance_id") ON DELETE RESTRICT ON UPDATE NO ACTION,
    CONSTRAINT "player_activity_membership_id_fkey" FOREIGN KEY ("membership_id") REFERENCES "player"("membership_id") ON DELETE RESTRICT ON UPDATE NO ACTION
);
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

    CONSTRAINT "player_stats_pkey" PRIMARY KEY ("membership_id","raid_id"),

    CONSTRAINT "raid_id_fkey" FOREIGN KEY ("raid_id") REFERENCES "raid"("id") ON DELETE RESTRICT ON UPDATE NO ACTION,
    CONSTRAINT "player_membership_id_fkey" FOREIGN KEY ("membership_id") REFERENCES "player"("membership_id") ON DELETE RESTRICT ON UPDATE NO ACTION,
    CONSTRAINT "player_stats_fastest_instance_id_fkey" FOREIGN KEY ("fastest_instance_id") REFERENCES "activity"("instance_id") ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE TYPE "WorldFirstLeaderboardType" AS ENUM ('Normal', 'Challenge', 'Prestige', 'Master');
CREATE TABLE "leaderboard" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "raid_id" INTEGER NOT NULL,
    "date" TIMESTAMP(3) NOT NULL,
    "is_world_first" BOOLEAN NOT NULL DEFAULT false,
    "type" "WorldFirstLeaderboardType" NOT NULL,

    -- CONSTRAINT "leaderboard_pkey" PRIMARY KEY ("id"),

    CONSTRAINT "leaderboard_raid_id_fkey" FOREIGN KEY ("raid_id") REFERENCES "raid"("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE UNIQUE INDEX "leaderboard_raid_hash_type_key" ON "leaderboard"("raid_id", "type");

CREATE TABLE "activity_leaderboard_entry" (
    "rank" INTEGER NOT NULL,
    "position" INTEGER NOT NULL,
    "leaderboard_id" TEXT NOT NULL,
    "instance_id" BIGINT NOT NULL,

    CONSTRAINT "activity_leaderboard_entry_pkey" PRIMARY KEY ("leaderboard_id", "instance_id"),

    CONSTRAINT "activity_leaderboard_entry_instance_id_fkey" FOREIGN KEY ("instance_id") REFERENCES "activity"("instance_id") ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT "activity_leaderboard_entry_leaderboard_id_fkey" FOREIGN KEY ("leaderboard_id") REFERENCES "leaderboard"("id") ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX "activity_leaderboard_entry_instance_id_index" ON "activity_leaderboard_entry" ("instance_id");
CREATE INDEX "activity_leaderboard_position" ON "activity_leaderboard_entry"("leaderboard_id", "position" ASC);

-- Raw PGCR Data
CREATE TABLE "pgcr" (
    "instance_id" BIGINT NOT NULL PRIMARY KEY,
    "data" BYTEA NOT NULL,
    "date_crawled" TIMESTAMP DEFAULT NOW()
);

-- Materialized Views
CREATE MATERIALIZED VIEW global_leaderboard AS
  SELECT
    membership_id,

    clears,
    ROW_NUMBER() OVER (ORDER BY clears DESC, membership_id ASC) AS clears_position,
    RANK() OVER (ORDER BY clears DESC) AS clears_rank,

    fresh_clears,
    ROW_NUMBER() OVER (ORDER BY fresh_clears DESC, membership_id ASC) AS fresh_clears_position,
    RANK() OVER (ORDER BY fresh_clears DESC) AS fresh_clears_rank,
    
    sherpas,
    ROW_NUMBER() OVER (ORDER BY sherpas DESC, membership_id ASC) AS sherpas_position,
    RANK() OVER (ORDER BY sherpas DESC) AS sherpas_rank,

    sum_of_best as speed,
    ROW_NUMBER() OVER (ORDER BY sum_of_best ASC, membership_id ASC) AS speed_position,
    RANK() OVER (ORDER BY sum_of_best ASC) AS speed_rank
    
  FROM player
  WHERE clears > 0;

CREATE UNIQUE INDEX idx_global_leaderboard_membership_id ON global_leaderboard (membership_id ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_clears ON global_leaderboard (clears_position ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_fresh_clears ON global_leaderboard (fresh_clears_position ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_sherpas ON global_leaderboard (sherpas_position ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_speed ON global_leaderboard (speed_position ASC);

CREATE MATERIALIZED VIEW world_first_player_rankings AS 
WITH tmp AS (
    SELECT
        p.membership_id,
        ROW_NUMBER() OVER (PARTITION BY p.membership_id, al.raid_id ORDER BY ale.rank ASC) AS placement_num,
        ((1 / SQRT(ale.rank)) * POWER(1.25, raid_id - 1)) as score
    FROM
        player p
    JOIN
        player_activity pa ON p.membership_id = pa.membership_id
    JOIN
        activity_leaderboard_entry ale ON pa.instance_id = ale.instance_id
    JOIN
        leaderboard al ON ale.leaderboard_id = al.id
    WHERE
        ale.rank <= 500 AND al.is_world_first
)
SELECT
    membership_id,
    SUM(score) AS score,
    RANK() OVER (ORDER BY SUM(score) DESC) AS rank
FROM tmp
WHERE placement_num = 1
GROUP BY membership_id
ORDER BY rank ASC;

CREATE UNIQUE INDEX idx_world_first_player_ranking_membership_id ON world_first_player_rankings (membership_id);
CREATE INDEX idx_world_first_player_ranking_rank ON world_first_player_rankings (rank ASC);
