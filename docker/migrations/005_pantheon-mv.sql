CREATE MATERIALIZED VIEW individual_pantheon_leaderboard AS
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
    
    trios,
    ROW_NUMBER() OVER (ORDER BY trios DESC, membership_id ASC) AS trios_position,
    RANK() OVER (ORDER BY trios DESC) AS trios_rank,
    
    duos,
    ROW_NUMBER() OVER (ORDER BY duos DESC, membership_id ASC) AS duos_position,
    RANK() OVER (ORDER BY duos DESC) AS duos_rank
  FROM player_stats
  WHERE clears > 0 AND player_stats.activity_id = 101;

CREATE UNIQUE INDEX idx_individual_pantheon_leaderboard_membership_id ON individual_pantheon_leaderboard (membership_id);
CREATE UNIQUE INDEX idx_individual_pantheon_leaderboard_clears ON individual_pantheon_leaderboard (clears_position ASC);
CREATE UNIQUE INDEX idx_individual_pantheon_leaderboard_fresh_clears ON individual_pantheon_leaderboard (fresh_clears_position ASC);
CREATE UNIQUE INDEX idx_individual_pantheon_leaderboard_sherpas ON individual_pantheon_leaderboard (sherpas_position ASC);
CREATE UNIQUE INDEX idx_individual_pantheon_leaderboard_trios ON individual_pantheon_leaderboard (trios_position ASC);