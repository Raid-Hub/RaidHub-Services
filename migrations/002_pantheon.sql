ALTER TABLE "activity" ADD COLUMN "score" INT NOT NULL DEFAULT 0;

ALTER TABLE "activity" RENAME COLUMN "raid_hash" TO "hash";
ALTER TABLE "activity" DROP CONSTRAINT "activity_raid_hash_fkey";

ALTER TABLE "raid_definition" RENAME TO "activity_hash";
ALTER TABLE "activity_hash" RENAME COLUMN "raid_id" TO "activity_id";
ALTER TABLE "player_stats" RENAME COLUMN "raid_id" TO "activity_id";

ALTER TABLE "raid" RENAME TO "activity_definition";
ALTER TABLE "activity_definition" ADD COLUMN "is_raid" BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE "raid_version" RENAME TO "version_definition";
ALTER TABLE "version_definition" RENAME COLUMN "associated_raid_id" TO "associated_activity_id";

INSERT INTO "activity_definition" ("id", "name", "is_raid") VALUES
    (101, 'Atraks Sovereign', false),
    (102, 'Oryx Exalted', false),
    (103, 'Rhulk Indomitable', false),
    (104, 'Nezarec Sublime', false);

-- Insert Version data
INSERT INTO "version_definition" ("id", "name", "associated_activity_id") VALUES
    (128, 'The Pantheon', NULL);

INSERT INTO "activity_hash" ("hash", "activity_id", "version_id") VALUES 
    -- Atraks Sovereign
    (4169648179, 101, 128),
    -- Oryx Exalted
    (4169648176, 102, 128),
    -- Rhulk Indomitable
    (4169648177, 103, 128),
    -- Nezarec Sublime
    (4169648182, 104, 128);

ALTER TABLE "activity" ADD CONSTRAINT "activity_hash_fk" FOREIGN KEY ("hash") REFERENCES "activity_hash"("hash");
