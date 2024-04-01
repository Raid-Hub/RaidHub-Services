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
