CREATE MATERIALIZED VIEW firsts_clears_tmp AS  (
    SELECT DISTINCT ON (ap.membership_id, ah.activity_id) 
        ap.instance_id,
        ap.membership_id,                                                 
        ah.activity_id                                                    
    FROM activity_player ap                                          
    JOIN activity a USING (instance_id)                            
    JOIN activity_hash ah USING (hash)                              
    WHERE ap.completed                                                  
    ORDER BY ap.membership_id, ah.activity_id, a.date_completed
);

CREATE MATERIALIZED VIEW noob_counts AS (
    SELECT 
        instance_id,
        count(*) AS newb_count             
    FROM firsts_clears_tmp                
    GROUP BY instance_id
);
CREATE UNIQUE INDEX idx_noob_counts ON noob_counts(instance_id);

CREATE MATERIALIZED VIEW p_stats_cache AS (
    WITH ordered_instances AS (
        SELECT 
            membership_id,
            activity_id,
            instance_id,
            ROW_NUMBER() OVER (PARTITION BY ap.membership_id, ah.activity_id ORDER BY a.duration ASC, a.date_completed ASC) AS rank
        FROM activity a
        JOIN activity_player ap USING (instance_id)
        JOIN activity_hash ah USING (hash)
        WHERE ap.completed AND a.fresh AND a.completed
    ),
    agg_stats AS (
        SELECT 
            ap.membership_id, 
            ah.activity_id, 
            COUNT(*) as clears,
            SUM(CASE WHEN a.fresh THEN 1 ELSE 0 END) as fresh_clears,
            SUM(ap.sherpas) as sherpa_count,
            SUM(ap.time_played_seconds) as total_time_played
        FROM activity_player ap
        JOIN activity a USING (instance_id)
        JOIN activity_hash ah USING (hash)
        WHERE ap.completed
        GROUP BY ap.membership_id, ah.activity_id
    )
    SELECT 
        agg_stats.*,
        ordered_instances.instance_id AS fastest_instance_id 
    FROM agg_stats
    LEFT JOIN ordered_instances USING (membership_id, activity_id)
    WHERE ordered_instances.rank = 1
);
CREATE UNIQUE INDEX idx_p_stats_cache ON p_stats_cache(membership_id, activity_id);