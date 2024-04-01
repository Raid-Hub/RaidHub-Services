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