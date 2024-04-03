CREATE TABLE "raid" (
    "id" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "is_sunset" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "raid_pkey" PRIMARY KEY ("id")
);

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
CREATE INDEX "idx_raid_definition_raid_id" ON "raid_definition"("raid_id");
CREATE INDEX "idx_raid_definition_version_id" ON "raid_definition"("version_id");

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

CREATE OR REPLACE FUNCTION bungie_name(p player) RETURNS VARCHAR AS $$
BEGIN
  RETURN CASE
           WHEN p.bungie_global_display_name IS NOT NULL AND p.bungie_global_display_name_code IS NOT NULL THEN
             CONCAT(p.bungie_global_display_name, '#', p.bungie_global_display_name_code)
           ELSE
             NULL
         END;
END;
$$ LANGUAGE plpgsql;

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

-- Insert Raid data
INSERT INTO "raid" ("id", "name", "is_sunset") VALUES
    (1, 'Leviathan', true),
    (2, 'Eater of Worlds', true),
    (3, 'Spire of Stars', true),
    (4, 'Last Wish', false),
    (5, 'Scourge of the Past', true),
    (6, 'Crown of Sorrow', true),
    (7, 'Garden of Salvation', false),
    (8, 'Deep Stone Crypt', false),
    (9, 'Vault of Glass', false),
    (10, 'Vow of the Disciple', false),
    (11, 'King''s Fall', false),
    (12, 'Root of Nightmares', false),
    (13, 'Crota''s End', false);

-- Insert Version data
INSERT INTO "raid_version" ("id", "name", "associated_raid_id") VALUES
    (1, 'Normal', NULL),
    (2, 'Guided Games', NULL),
    (3, 'Prestige', NULL),
    (4, 'Master', NULL),
    (64, 'Tempo''s Edge', 9),
    (65, 'Regicide', 11),
    (66, 'Superior Swordplay', 13);


-- Insert RaidHash data
INSERT INTO "raid_definition" ("raid_id", "version_id", "hash") VALUES
    -- LEVIATHAN
    (1, 1, 2693136600),
    (1, 1, 2693136601),
    (1, 1, 2693136602),
    (1, 1, 2693136603),
    (1, 1, 2693136604),
    (1, 1, 2693136605),
    -- LEVIATHAN GUIDEDGAMES
    (1, 2, 89727599),
    (1, 2, 287649202),
    (1, 2, 1699948563),
    (1, 2, 1875726950),
    (1, 2, 3916343513),
    (1, 2, 4039317196),
    -- LEVIATHAN PRESTIGE
    (1, 3, 417231112),
    (1, 3, 508802457),
    (1, 3, 757116822),
    (1, 3, 771164842),
    (1, 3, 1685065161),
    (1, 3, 1800508819),
    (1, 3, 2449714930),
    (1, 3, 3446541099),
    (1, 3, 4206123728),
    (1, 3, 3912437239),
    (1, 3, 3879860661),
    (1, 3, 3857338478),
    -- EATER_OF_WORLDS
    (2, 1, 3089205900),
    -- EATER_OF_WORLDS GUIDEDGAMES
    (2, 2, 2164432138),
    -- EATER_OF_WORLDS PRESTIGE
    (2, 3, 809170886),
    -- SPIRE_OF_STARS
    (3, 1, 119944200),
    -- SPIRE_OF_STARS GUIDEDGAMES
    (3, 2, 3004605630),
    -- SPIRE_OF_STARS PRESTIGE
    (3, 3, 3213556450),
    -- LAST_WISH
    (4, 1, 2122313384),
    (4, 1, 2214608157),
    -- LAST_WISH GUIDEDGAMES
    (4, 2, 1661734046),
    -- SCOURGE_OF_THE_PAST
    (5, 1, 548750096),
    -- SCOURGE_OF_THE_PAST GUIDEDGAMES
    (5, 2, 2812525063),
    -- CROWN_OF_SORROW
    (6, 1, 3333172150),
    -- CROWN_OF_SORROW GUIDEDGAMES
    (6, 2, 960175301),
    -- GARDEN_OF_SALVATION
    (7, 1, 2659723068),
    (7, 1, 3458480158),
    (7, 1, 1042180643),
    -- GARDEN_OF_SALVATION GUIDEDGAMES
    (7, 2, 2497200493),
    (7, 2, 3845997235),
    (7, 2, 3823237780),
    -- DEEP_STONE_CRYPT
    (8, 1, 910380154),
    -- DEEP_STONE_CRYPT GUIDEDGAMES
    (8, 2, 3976949817),
    -- VAULT_OF_GLASS
    (9, 1, 3881495763),
    -- VAULT_OF_GLASS GUIDEDGAMES
    (9, 2, 3711931140),
    -- VAULT_OF_GLASS CHALLENGE_VOG
    (9, 64, 1485585878),
    -- VAULT_OF_GLASS MASTER
    (9, 4, 1681562271),
    (9, 4, 3022541210),
    -- VOW_OF_THE_DISCIPLE
    (10, 1, 1441982566),
    (10, 1, 2906950631),
    -- VOW_OF_THE_DISCIPLE GUIDEDGAMES
    (10, 2, 4156879541),
    -- VOW_OF_THE_DISCIPLE MASTER
    (10, 4, 4217492330),
    (10, 4, 3889634515),
    -- KINGS_FALL
    (11, 1, 1374392663),
    -- KINGS_FALL GUIDEDGAMES
    (11, 2, 2897223272),
    -- KINGS_FALL CHALLENGE_KF
    (11, 65, 1063970578),
    -- KINGS_FALL MASTER
    (11, 4, 2964135793),
    (11, 4, 3257594522),
    -- ROOT_OF_NIGHTMARES
    (12, 1, 2381413764),
    -- ROOT_OF_NIGHTMARES GUIDEDGAMES
    (12, 2, 1191701339),
    -- ROOT_OF_NIGHTMARES MASTER
    (12, 4, 2918919505),
    -- CROTAS_END
    (13, 1, 4179289725),
    -- CROTAS_END GUIDEDGAMES
    (13, 2, 4103176774),
    -- CROTAS_END CHALLENGE_CROTA
    (13, 66, 156253568),
    -- CROTAS_END MASTER
    (13, 4, 1507509200);

CREATE OR REPLACE FUNCTION SEASON(p_input_date timestamp with time zone)
RETURNS integer AS $$
DECLARE
    v_season_number integer;
    v_season_dates timestamp with time zone[] := ARRAY[
        '2017-12-05T17:00:00Z',
        '2018-05-08T18:00:00Z',
        '2018-09-04T17:00:00Z',
        '2018-11-27T17:00:00Z',
        '2019-03-05T17:00:00Z',
        '2019-06-04T17:00:00Z',
        '2019-10-01T17:00:00Z',
        '2019-12-10T17:00:00Z',
        '2020-03-10T17:00:00Z',
        '2020-06-09T17:00:00Z',
        '2020-11-10T17:00:00Z',
        '2021-02-09T17:00:00Z',
        '2021-05-11T17:00:00Z',
        '2021-08-24T17:00:00Z',
        '2022-02-22T17:00:00Z',
        '2022-05-24T17:00:00Z',
        '2022-08-23T17:00:00Z',
        '2022-12-06T17:00:00Z',
        '2023-02-28T17:00:00Z',
        '2023-05-23T17:00:00Z',
        '2023-08-22T17:00:00Z',
        '2023-11-28T17:00:00Z',
        '2024-06-04T17:00:00Z',
        -- add new seasons here
        '2099-12-31T17:00:00Z'
    ];
    v_low integer := 1;
    v_high integer := array_length(v_season_dates, 1);
BEGIN
    -- binary search
    WHILE v_low <= v_high LOOP
        DECLARE
            v_mid integer := floor((v_low + v_high) / 2);
        BEGIN
            IF p_input_date >= v_season_dates[v_mid] THEN
                v_low := v_mid + 1;
            ELSE
                v_high := v_mid - 1;
            END IF;
        END;
    END LOOP;

    v_season_number := v_high + 1;

    RETURN v_season_number;
END;
$$ LANGUAGE plpgsql;

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

CREATE MATERIALIZED VIEW individual_leaderboard AS
  SELECT
    membership_id,
    raid_id,

    clears,
    ROW_NUMBER() OVER (PARTITION BY raid_id ORDER BY clears DESC, membership_id ASC) AS clears_position,
    RANK() OVER (PARTITION BY raid_id ORDER BY clears DESC) AS clears_rank,

    fresh_clears,
    ROW_NUMBER() OVER (PARTITION BY raid_id ORDER BY fresh_clears DESC, membership_id ASC) AS fresh_clears_position,
    RANK() OVER (PARTITION BY raid_id ORDER BY fresh_clears DESC) AS fresh_clears_rank,
    
    sherpas,
    ROW_NUMBER() OVER (PARTITION BY raid_id ORDER BY sherpas DESC, membership_id ASC) AS sherpas_position,
    RANK() OVER (PARTITION BY raid_id ORDER BY sherpas DESC) AS sherpas_rank,
    
    trios,
    ROW_NUMBER() OVER (PARTITION BY raid_id ORDER BY trios DESC, membership_id ASC) AS trios_position,
    RANK() OVER (PARTITION BY raid_id ORDER BY trios DESC) AS trios_rank,
    
    duos,
    ROW_NUMBER() OVER (PARTITION BY raid_id ORDER BY duos DESC, membership_id ASC) AS duos_position,
    RANK() OVER (PARTITION BY raid_id ORDER BY duos DESC) AS duos_rank,
    
    solos,
    ROW_NUMBER() OVER (PARTITION BY raid_id ORDER BY solos DESC, membership_id ASC) AS solos_position,
    RANK() OVER (PARTITION BY raid_id ORDER BY solos DESC) AS solos_rank
  FROM player_stats
  WHERE clears > 0;

CREATE UNIQUE INDEX idx_individual_leaderboard_membership_id ON individual_leaderboard (raid_id DESC, membership_id ASC);
CREATE UNIQUE INDEX idx_individual_leaderboard_clears ON individual_leaderboard (raid_id DESC, clears_position ASC);
CREATE UNIQUE INDEX idx_individual_leaderboard_fresh_clears ON individual_leaderboard (raid_id DESC, fresh_clears_position ASC);
CREATE UNIQUE INDEX idx_individual_leaderboard_sherpas ON individual_leaderboard (raid_id DESC, sherpas_position ASC);
CREATE UNIQUE INDEX idx_individual_leaderboard_trios ON individual_leaderboard (raid_id DESC, trios_position ASC);
CREATE UNIQUE INDEX idx_individual_leaderboard_duos ON individual_leaderboard (raid_id DESC, duos_position ASC);
CREATE UNIQUE INDEX idx_individual_leaderboard_solos ON individual_leaderboard (raid_id DESC, solos_position ASC);

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
