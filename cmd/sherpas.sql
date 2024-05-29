BEGIN;

-- Set everyones first raid clears 
WITH firsts AS (
    SELECT 
        instance_id,
        ap.membership_id,
        ah.activity_id,
        ROW_NUMBER() OVER (PARTITION BY ap.membership_id, ah.activity_id ORDER BY a.date_completed ASC) AS clear_num
    FROM activity_player ap
    JOIN activity a USING (instance_id)
    JOIN activity_hash ah USING (hash)
    WHERE ap.completed
)
UPDATE activity_player ap
SET is_first_clear = f.clear_num = 1
FROM firsts f
WHERE ap.instance_id = f.instance_id AND ap.membership_id = f.membership_id;

-- Set the sherpa count for each activity_player
WITH sherpas AS (
    SELECT instance_id, COUNT(*) as newb_count
    FROM activity_player WHERE is_first_clear
    GROUP BY instance_id
)
UPDATE activity_player ap
SET sherpas = s.newb_count
FROM sherpas s
WHERE ap.instance_id = s.instance_id 
    AND NOT ap.is_first_clear;

-- Update activity level stats
WITH p_stats AS (
    SELECT 
        ap.membership_id, 
        ah.activity_id, 
        COUNT(*) as clears,
        SUM(CASE WHEN a.fresh THEN 1 ELSE 0 END) as fresh_clears,
        SUM(ap.sherpas) as sherpa_count, 
        SUM(CASE WHEN a.player_count = 3 THEN 1 ELSE 0 END) as trios,
        SUM(CASE WHEN a.player_count = 2 THEN 1 ELSE 0 END) as duos,
        SUM(CASE WHEN a.player_count = 1 THEN 1 ELSE 0 END) as solos,
    FROM activity_player ap
    JOIN activity a USING (instance_id)
    JOIN activity_hash ah USING (hash)
    WHERE ap.completed
    GROUP BY ap.membership_id, ah.activity_id
)
UPDATE player_stats ps
SET 
    clears = p.clears,
    fresh_clears = p.fresh_clears,
    sherpas = p.sherpa_count,
    trios = p.trios,
    duos = p.duos,
    solos = p.solos
FROM p_stats p
WHERE ps.membership_id = p.membership_id
    AND ps.activity_id = p.activity_id;

-- Update player level stats
WITH g_stats AS (
    SELECT 
        membership_id, 
        SUM(clears) as clears,
        SUM(fresh_clears) as fresh_clears,
        SUM(sherpas) as sherpas
    FROM player_stats
    GROUP BY membership_id
)
UPDATE player SET 
    clears = g_stats.clears,
    fresh_clears = g_stats.fresh_clears,
    sherpas = g_stats.sherpas
FROM g_stats
WHERE g_stats.membership_id = player.membership_id;

COMMIT;