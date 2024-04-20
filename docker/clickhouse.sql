
CREATE TABLE player (
    membership_id                   Int64,
    membership_type                 UInt16,
    icon_path                       String,
    display_name                    String,
    bungie_global_display_name      String,
    bungie_global_display_name_code String,
    bungie_name                     String,
    last_seen                       DateTime,
    sherpas                         UInt32,
    sum_of_best                     Int32,

    PRIMARY KEY (membership_id)
) ENGINE = ReplacingMergeTree()
ORDER BY (membership_id, last_seen);

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

    PRIMARY KEY (instance_id)
) ENGINE = ReplacingMergeTree()
ORDER BY (instance_id);

CREATE TABLE instance_player (
    instance_id          Int64,
    membership_id        Int64,
    completed            UInt8,
    time_played_seconds  UInt32,
    sherpas              UInt32,
    is_first_clear       UInt8,

    PRIMARY KEY (instance_id, membership_id)
) ENGINE = ReplacingMergeTree()
ORDER BY (instance_id, membership_id);

CREATE TABLE instance_character (
    instance_id          Int64,
    membership_id        Int64,
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

    PRIMARY KEY (instance_id, membership_id, character_id)
) ENGINE = ReplacingMergeTree()
ORDER BY (instance_id, membership_id, character_id);

CREATE TABLE instance_character_weapon (
    instance_id     Int64,
    membership_id   Int64,
    character_id    Int64,
    weapon_hash     UInt32,
    kills           UInt32,
    precision_kills UInt32,

    PRIMARY KEY (instance_id, membership_id, character_id, weapon_hash)
) ENGINE = ReplacingMergeTree()
ORDER BY (instance_id, membership_id, character_id, weapon_hash);

CREATE TABLE weapon_definition (
    hash         UInt32,
    name         String,
    icon_path    String,

    PRIMARY KEY (hash)
) ENGINE = ReplacingMergeTree();

CREATE TABLE activity_definition (
    id          Int32,
    name        String,
    is_sunset   UInt8,
    is_raid     UInt8,

    PRIMARY KEY (id)
) ENGINE = ReplacingMergeTree()
ORDER BY id;

CREATE TABLE version_definition (
    id    UInt8,
    name  String,

    PRIMARY KEY (id)
) ENGINE = ReplacingMergeTree()
ORDER BY id;

CREATE TABLE activity (
    hash         UInt32,
    activity_id  UInt8,
    version_id   UInt8,

    PRIMARY KEY (hash)
) ENGINE = ReplacingMergeTree()

