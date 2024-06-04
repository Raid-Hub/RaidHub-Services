
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
    RANK() OVER (PARTITION BY activity_id ORDER BY player_stats.sherpas DESC) AS sherpas_rank
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
        SELECT hash FROM activity_hash WHERE activity_id = 101
    )
    SELECT 
        "lateral".membership_id,
        version_id,
        COUNT(*) AS clears,
        SUM(CASE WHEN "lateral".fresh THEN 1 ELSE 0 END) AS fresh_clears,
        SUM("lateral".score) AS score
    FROM hashes
    JOIN activity_hash USING (hash)
    LEFT JOIN LATERAL (
        SELECT 
            membership_id,
            fresh,
            score
        FROM activity_player 
        JOIN activity USING (instance_id)
        JOIN player USING (membership_id)
        WHERE activity_player.completed
            AND activity_hash.hash = activity.hash
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
    RANK() OVER (ORDER BY sum_of_best ASC) AS speed_rank
    
  FROM player
  WHERE clears > 0 AND NOT is_private AND cheat_level < 2;

CREATE UNIQUE INDEX idx_global_leaderboard_membership_id ON global_leaderboard (membership_id ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_clears ON global_leaderboard (clears_position ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_fresh_clears ON global_leaderboard (fresh_clears_position ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_sherpas ON global_leaderboard (sherpas_position ASC);
CREATE UNIQUE INDEX idx_global_leaderboard_speed ON global_leaderboard (speed_position ASC);

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
      FROM activity_hash
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
    FROM activity_player
    WHERE activity_player.instance_id = raw.instance_id
      AND activity_player.completed
    LIMIT 12
  ) as "players" ON true
  WHERE position <= 1000;

CREATE UNIQUE INDEX idx_team_activity_version_leaderboard_position ON team_activity_version_leaderboard (activity_id ASC, version_id ASC, position ASC);
CREATE INDEX idx_team_activity_version_leaderboard_membership_id ON team_activity_version_leaderboard USING GIN (membership_ids);

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
    FROM "activity_hash"
    INNER JOIN "activity_definition" ON "activity_definition"."id" = "activity_hash"."activity_id"
    INNER JOIN "version_definition" ON "version_definition"."id" = "activity_hash"."version_id"
    LEFT JOIN LATERAL (
      SELECT 
        "instance_id", 
        "date_completed"
      FROM "activity"
      WHERE "hash" = "activity_hash"."hash" 
        AND "completed" AND "cheat_override" = false
        AND "date_completed" < COALESCE("contest_end", "week_one_end")
      LIMIT 80000
    ) as "__inner__" ON true
    WHERE "is_world_first" = true
  )
  SELECT "entries".*, "players"."membership_ids" FROM "entries"
  LEFT JOIN LATERAL (
    SELECT JSONB_AGG("membership_id") AS "membership_ids"
    FROM "activity_player"
    WHERE "activity_player"."instance_id" = "entries"."instance_id"
      AND "activity_player"."completed"
    LIMIT 12
  ) AS "players" ON true;

CREATE INDEX idx_world_first_contest_leaderboard_rank ON world_first_contest_leaderboard (activity_id, position ASC);
CREATE UNIQUE INDEX idx_world_first_contest_leaderboard_instance ON world_first_contest_leaderboard (instance_id);
CREATE INDEX idx_world_first_contest_leaderboard_membership_ids ON world_first_contest_leaderboard USING GIN (membership_ids);