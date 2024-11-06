CREATE MATERIALIZED VIEW "individual_raid_leaderboard" AS
  SELECT
    membership_id,
    activity_id,

    player_stats.clears,
    ROW_NUMBER() OVER (PARTITION BY activity_id ORDER BY player_stats.clears DESC, membership_id ASC) AS clears_position,
    RANK() OVER (PARTITION BY activity_id ORDER BY player_stats.clears DESC) AS clears_rank,

    player_stats.fresh_clears,
    ROW_NUMBER() OVER (PARTITION BY activity_id ORDER BY player_stats.fresh_clears DESC, membership_id ASC) AS fresh_clears_position,
    RANK() OVER (PARTITION BY activity_id ORDER BY player_stats.fresh_clears DESC) AS fresh_clears_rank,
    
    player_stats.sherpas,
    ROW_NUMBER() OVER (PARTITION BY activity_id ORDER BY player_stats.sherpas DESC, membership_id ASC) AS sherpas_position,
    RANK() OVER (PARTITION BY activity_id ORDER BY player_stats.sherpas DESC) AS sherpas_rank,

    player_stats.total_time_played_seconds AS total_time_played,
    ROW_NUMBER() OVER (PARTITION BY activity_id ORDER BY player_stats.total_time_played_seconds DESC, membership_id ASC) AS total_time_played_position,
    RANK() OVER (PARTITION BY activity_id ORDER BY player_stats.total_time_played_seconds DESC) AS total_time_played_rank
  FROM player_stats
  JOIN player USING (membership_id) 
  WHERE player_stats.clears > 0 AND activity_id IN (
    SELECT id FROM activity_definition WHERE is_raid = true
  )
  AND NOT player.is_private AND player.cheat_level < 2;

CREATE UNIQUE INDEX idx_individual_raid_leaderboard_membership_id ON individual_raid_leaderboard (activity_id DESC, membership_id ASC);
CREATE UNIQUE INDEX idx_individual_raid_leaderboard_clears ON individual_raid_leaderboard (activity_id DESC, clears_position ASC);
CREATE UNIQUE INDEX idx_individual_raid_leaderboard_fresh_clears ON individual_raid_leaderboard (activity_id DESC, fresh_clears_position ASC);
CREATE UNIQUE INDEX idx_individual_raid_leaderboard_sherpas ON individual_raid_leaderboard (activity_id DESC, sherpas_position ASC);
CREATE UNIQUE INDEX idx_individual_raid_leaderboard_total_time_played ON individual_raid_leaderboard (activity_id DESC, total_time_played_position ASC);

-- ALTER MATERIALIZED VIEW "individual_raid_leaderboard" OWNER TO raidhub_user;

CREATE MATERIALIZED VIEW "individual_pantheon_version_leaderboard" AS
  SELECT
    membership_id,
    version_id,

    clears,
    ROW_NUMBER() OVER (PARTITION BY version_id ORDER BY clears DESC, membership_id ASC) AS clears_position,
    RANK() OVER (PARTITION BY version_id ORDER BY clears DESC) AS clears_rank,

    fresh_clears,
    ROW_NUMBER() OVER (PARTITION BY version_id ORDER BY fresh_clears DESC, membership_id ASC) AS fresh_clears_position,
    RANK() OVER (PARTITION BY version_id ORDER BY fresh_clears DESC) AS fresh_clears_rank,

    score,
    ROW_NUMBER() OVER (PARTITION BY version_id ORDER BY score DESC, membership_id ASC) AS score_position,
    RANK() OVER (PARTITION BY version_id ORDER BY score DESC) AS score_rank
  FROM (
    WITH hashes AS (
        SELECT hash FROM activity_version WHERE activity_id = 101
    )
    SELECT 
        "lateral".membership_id,
        version_id,
        COUNT(*) AS clears,
        SUM(CASE WHEN "lateral".fresh THEN 1 ELSE 0 END) AS fresh_clears,
        SUM("lateral".score) AS score
    FROM hashes
    JOIN activity_version USING (hash)
    LEFT JOIN LATERAL (
        SELECT 
            membership_id,
            fresh,
            score
        FROM instance_player 
        JOIN instance USING (instance_id)
        JOIN player USING (membership_id)
        WHERE instance_player.completed
            AND activity_version.hash = instance.hash
            AND NOT player.is_private AND player.cheat_level < 2
    ) AS "lateral" ON TRUE
     GROUP BY membership_id, version_id
  ) as foo
  WHERE clears > 0;

CREATE UNIQUE INDEX idx_individual_pantheon_version_leaderboard_membership_id ON individual_pantheon_version_leaderboard (version_id ASC, membership_id ASC);
CREATE UNIQUE INDEX idx_individual_pantheon_version_leaderboard_clears ON individual_pantheon_version_leaderboard (version_id ASC, clears_position ASC);
CREATE UNIQUE INDEX idx_individual_pantheon_version_leaderboard_fresh_clears ON individual_pantheon_version_leaderboard (version_id ASC, fresh_clears_position ASC);
CREATE UNIQUE INDEX idx_individual_pantheon_version_leaderboard_score ON individual_pantheon_version_leaderboard (version_id ASC, score_position ASC);

-- Materialized Views
CREATE MATERIALIZED VIEW "individual_global_leaderboard" AS
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
    RANK() OVER (ORDER BY sum_of_best ASC) AS speed_rank,

    total_time_played_seconds AS total_time_played,
    ROW_NUMBER() OVER (ORDER BY total_time_played_seconds DESC, membership_id ASC) AS total_time_played_position,
    RANK() OVER (ORDER BY total_time_played_seconds DESC) AS total_time_played_rank
    
  FROM player
  WHERE clears > 0 AND NOT is_private AND cheat_level < 2;

CREATE UNIQUE INDEX idx_global_leaderboard_membership_id ON individual_global_leaderboard (membership_id ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_clears ON individual_global_leaderboard (clears_position ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_fresh_clears ON individual_global_leaderboard (fresh_clears_position ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_sherpas ON individual_global_leaderboard (sherpas_position ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_speed ON individual_global_leaderboard (speed_position ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_total_time_played ON individual_global_leaderboard (total_time_played_position ASC);

-- ALTER MATERIALIZED VIEW "individual_global_leaderboard" OWNER TO raidhub_user;

CREATE MATERIALIZED VIEW "team_activity_version_leaderboard" AS
  WITH raw AS (
    SELECT
      activity_id,
      version_id,
      instance_id,
      time_after_launch AS value,
      ROW_NUMBER() OVER (PARTITION BY activity_id, version_id ORDER BY date_completed ASC) AS position,
      RANK() OVER (PARTITION BY activity_id, version_id ORDER BY date_completed ASC) AS rank
    FROM (
      SELECT hash, activity_id, version_id, release_date_override
      FROM activity_version
      WHERE version_id <> 2 -- Ignore Guided Games
      ORDER BY activity_id ASC, version_id ASC
      LIMIT 100
    ) AS activity_version
    JOIN activity_definition ON activity_version.activity_id = activity_definition.id
    LEFT JOIN LATERAL (
      SELECT 
        instance_id, 
        date_completed,
        EXTRACT(EPOCH FROM (date_completed - COALESCE(release_date_override, release_date))) AS time_after_launch 
      FROM activity
      WHERE activity.hash = activity_version.hash
        AND activity.completed 
        AND NOT activity.cheat_override
      ORDER BY activity.date_completed ASC
      LIMIT 1000
    ) AS first_thousand ON true
  )
  SELECT raw.*, "players".membership_ids FROM raw
  LEFT JOIN LATERAL (
    SELECT JSONB_AGG(membership_id) AS membership_ids
    FROM instance_player
    WHERE instance_player.instance_id = raw.instance_id
      AND instance_player.completed
    LIMIT 12
  ) as "players" ON true
  WHERE position <= 1000;

CREATE UNIQUE INDEX idx_team_activity_version_leaderboard_position ON team_activity_version_leaderboard (activity_id ASC, version_id ASC, position ASC);
CREATE INDEX idx_team_activity_version_leaderboard_membership_id ON team_activity_version_leaderboard USING GIN (membership_ids);

-- ALTER MATERIALIZED VIEW "team_activity_version_leaderboard" OWNER TO raidhub_user;

CREATE MATERIALIZED VIEW "world_first_contest_leaderboard" AS
   WITH "entries" AS (
    SELECT
      "activity_id",
      ROW_NUMBER() OVER (PARTITION BY "activity_id" ORDER BY "date_completed" ASC) AS "position",
      RANK() OVER (PARTITION BY "activity_id" ORDER BY "date_completed" ASC) AS "rank",
      "instance_id",
      "date_completed",
      EXTRACT(EPOCH FROM ("date_completed" - "release_date")) AS "time_after_launch",
      "is_challenge_mode"
    FROM "activity_version"
    INNER JOIN "activity_definition" ON "activity_definition"."id" = "activity_version"."activity_id"
    INNER JOIN "version_definition" ON "version_definition"."id" = "activity_version"."version_id"
    LEFT JOIN LATERAL (
      SELECT 
        "instance_id", 
        "date_completed"
      FROM "instance"
      WHERE "hash" = "activity_version"."hash" 
        AND "completed" AND "cheat_override" = false
        AND "date_completed" < COALESCE("contest_end", "week_one_end")
      LIMIT 80000
    ) as "__inner__" ON true
    WHERE "is_world_first" = true
  )
  SELECT "entries".*, "players"."membership_ids" FROM "entries"
  LEFT JOIN LATERAL (
    SELECT JSONB_AGG("membership_id") AS "membership_ids"
    FROM "instance_player"
    WHERE "instance_player"."instance_id" = "entries"."instance_id"
      AND "instance_player"."completed"
    LIMIT 12
  ) AS "players" ON true;

CREATE INDEX idx_world_first_contest_leaderboard_rank ON world_first_contest_leaderboard (activity_id, position ASC);
CREATE UNIQUE INDEX idx_world_first_contest_leaderboard_instance ON world_first_contest_leaderboard (instance_id);
CREATE INDEX idx_world_first_contest_leaderboard_membership_ids ON world_first_contest_leaderboard USING GIN (membership_ids);

-- ALTER MATERIALIZED VIEW "world_first_contest_leaderboard" OWNER TO raidhub_user;

CREATE MATERIALIZED VIEW "world_first_player_rankings" AS 
WITH unnested_entries AS (
    SELECT
        world_first_contest_leaderboard.*,
        jsonb_array_elements(membership_ids)::bigint AS membership_id
    FROM
        world_first_contest_leaderboard
), tmp AS (
    SELECT DISTINCT ON (membership_id, activity_id)
        membership_id,
        ((1 / SQRT(rank)) * POWER(1.25, activity_id - 1)) as score
    FROM unnested_entries
    ORDER BY membership_id, activity_id, rank ASC
)
SELECT
    membership_id,
    SUM(score) AS score,
    RANK() OVER (ORDER BY SUM(score) DESC) AS rank,
    ROW_NUMBER() OVER (ORDER BY SUM(score) DESC) AS position
FROM tmp
JOIN player USING (membership_id)
WHERE cheat_level < 2
GROUP BY membership_id
ORDER BY rank ASC;

CREATE UNIQUE INDEX idx_world_first_player_ranking_membership_id ON world_first_player_rankings (membership_id);
CREATE INDEX idx_world_first_player_ranking_position ON world_first_player_rankings (position ASC);

-- ALTER MATERIALIZED VIEW "world_first_player_rankings" OWNER TO raidhub_user;


CREATE MATERIALIZED VIEW "clan_leaderboard" AS (
    WITH
    "ranked_scores" AS (
        SELECT 
            cm."membership_id",
            cm."group_id",
            wpr."score",
            ROW_NUMBER() OVER (PARTITION BY cm."group_id" ORDER BY wpr."score" DESC) AS "intra_clan_ranking"
        FROM "clan_members" cm
        LEFT JOIN "world_first_player_rankings" wpr ON cm."membership_id" = wpr."membership_id"
    )
    SELECT 
        "group_id",
        COUNT("membership_id") AS "known_member_count",
        SUM("p"."clears") AS "clears",
        ROUND(AVG("p"."clears")) AS "average_clears",
        SUM("p"."fresh_clears") AS "fresh_clears",
        ROUND(AVG("p"."fresh_clears")) AS "average_fresh_clears",
        SUM("p"."sherpas") AS "sherpas",
        ROUND(AVG("p"."sherpas")) AS "average_sherpas",
        SUM("p"."total_time_played_seconds") AS "time_played_seconds",
        ROUND(AVG("p"."total_time_played_seconds")) AS "average_time_played_seconds",
        COALESCE(SUM(rs."score"), 0) AS "total_contest_score",
        COALESCE(SUM(rs."score" * POWER(0.9, rs."intra_clan_ranking" - 6))::DOUBLE PRECISION / (POWER(1 + COUNT("membership_id"), (1 / 3))), 0) AS "weighted_contest_score"
    FROM "clan_members" cm
    JOIN "player" p USING ("membership_id")
    JOIN "ranked_scores" rs USING ("group_id", "membership_id")
    JOIN "clan" USING ("group_id")
    GROUP BY "group_id", "clan"."name"
);
CREATE UNIQUE INDEX idx_clan_leaderboard_group_id ON clan_leaderboard (group_id);

-- ALTER MATERIALIZED VIEW "clan_leaderboard" OWNER TO raidhub_user;