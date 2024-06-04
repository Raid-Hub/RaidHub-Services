SET flatten_nested = 0;
CREATE TABLE instance (
    instance_id     Int64,
    hash            UInt32,
    completed       UInt8,
    player_count    UInt32,
    fresh           UInt8,
    flawless        UInt8,
    date_started    DateTime,
    date_completed  DateTime,
    platform_type   UInt16,
    duration        UInt32,
    score           Int32,
    players         Nested(
            membership_id        Int64,
            completed            UInt8,
            time_played_seconds  UInt32,
            sherpas              UInt32,
            is_first_clear       UInt8,
            characters           Nested(
                    character_id         Int64,
                    class_hash           UInt32,
                    emblem_hash          UInt32,
                    completed            UInt8,
                    score                Int32,
                    kills                UInt32,
                    assists              UInt32,
                    deaths               UInt32,
                    precision_kills      UInt32,
                    super_kills          UInt32,
                    grenade_kills        UInt32,
                    melee_kills          UInt32,
                    time_played_seconds  UInt32,
                    start_seconds        UInt32,
                    weapons              Nested(
                            weapon_hash     UInt32,
                            kills           UInt32,
                            precision_kills UInt32
            )
        )
    ),
    PRIMARY KEY (date_completed, instance_id)
) ENGINE = ReplacingMergeTree()
ORDER BY (date_completed, instance_id);