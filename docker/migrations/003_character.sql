ALTER TABLE "player_activity" RENAME TO "activity_player";
ALTER TABLE "activity_player" RENAME COLUMN "finished_raid" TO "completed";

CREATE TABLE "activity_character" (
    "instance_id" BIGINT NOT NULL,
    "membership_id" BIGINT NOT NULL,
    "character_id" BIGINT NOT NULL,
    "class_hash" BIGINT,
    "emblem_hash" BIGINT,
    "completed" BOOLEAN NOT NULL,
    "score" INTEGER NOT NULL DEFAULT 0,
    "kills" INTEGER NOT NULL DEFAULT 0,
    "assists" INTEGER NOT NULL DEFAULT 0,
    "deaths" INTEGER NOT NULL DEFAULT 0,
    "precision_kills" INTEGER NOT NULL DEFAULT 0,
    "super_kills" INTEGER NOT NULL DEFAULT 0,
    "grenade_kills" INTEGER NOT NULL DEFAULT 0,
    "melee_kills" INTEGER NOT NULL DEFAULT 0,
    "time_played_seconds" INTEGER NOT NULL,
    "start_seconds" INTEGER NOT NULL,

    CONSTRAINT "activity_character_pkey" PRIMARY KEY ("instance_id", "membership_id", "character_id"),
    
    CONSTRAINT "activity_character_instance_id_membership_id_fkey" FOREIGN KEY ("instance_id", "membership_id") REFERENCES "activity_player"("instance_id", "membership_id") ON DELETE RESTRICT ON UPDATE NO ACTION
);
CREATE INDEX "activity_character_idx_membership_id" ON "activity_character"("membership_id");

CREATE TABLE "activity_character_weapon" (
    "instance_id" BIGINT NOT NULL,
    "membership_id" BIGINT NOT NULL,
    "character_id" BIGINT NOT NULL,
    "weapon_hash" BIGINT NOT NULL,
    "kills" INTEGER NOT NULL DEFAULT 0,
    "precision_kills" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "activity_character_weapon_pkey" PRIMARY KEY ("instance_id","membership_id","character_id","weapon_hash"),

    CONSTRAINT "activity_character_weapon_fkey" FOREIGN KEY ("instance_id","membership_id","character_id") REFERENCES "activity_character"("instance_id","membership_id","character_id") ON DELETE RESTRICT ON UPDATE NO ACTION
);

ALTER TABLE "activity_player" DROP COLUMN "class_hash", DROP COLUMN "kills", DROP COLUMN "assists", DROP COLUMN "deaths";
DROP INDEX IF EXISTS "idx_instance_id";
