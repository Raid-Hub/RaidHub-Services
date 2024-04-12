ALTER TABLE "activity" ADD COLUMN "score" INT NOT NULL DEFAULT 0;

ALTER TABLE "activity" RENAME COLUMN "raid_hash" TO "hash";
ALTER TABLE "activity" DROP CONSTRAINT "activity_raid_hash_fkey";

CREATE TABLE "activity_definition" (
    "hash" BIGINT PRIMARY KEY
);

ALTER TABLE "raid_definition" RENAME TO "raid_definition_old";
CREATE TABLE "raid_definition" (
    "raid_id" INTEGER NOT NULL,
    "version_id" INTEGER NOT NULL,

    CONSTRAINT "raid_definition_raid_id_fkey" FOREIGN KEY ("raid_id") REFERENCES "raid"("id") ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT "raid_definition_version_id_fkey" FOREIGN KEY ("version_id") REFERENCES "raid_version"("id") ON DELETE RESTRICT ON UPDATE CASCADE
) INHERITS ("activity_definition");

INSERT INTO "raid_definition" ("hash", "raid_id", "version_id")
SELECT "hash", "raid_id", "version_id" FROM "raid_definition_old";

CREATE TABLE "pantheon_version" (
    "id" INTEGER NOT NULL PRIMARY KEY,
    "name" TEXT NOT NULL
);

CREATE TABLE "pantheon_definition" (
    "version_id" INTEGER NOT NULL,

    CONSTRAINT "pantheon_definition_version_id_fkey" FOREIGN KEY ("version_id") REFERENCES "pantheon_version"("id") ON DELETE RESTRICT ON UPDATE CASCADE
) INHERITS ("activity_definition");

INSERT INTO "pantheon_version" ("id", "name") VALUES 
    (0, 'Atraks Sovereign'),
    (1, 'Oryx Exalted'),
    (2, 'Rhulk Indomitable'),
    (3, 'Nezarec Sublime');

INSERT INTO "pantheon_definition" ("hash", "version_id") VALUES 
    -- Atraks Sovereign
    (4169648179, 0),
    -- Oryx Exalted
    (4169648176, 1),
    -- Rhulk Indomitable
    (4169648177, 2),
    -- Nezarec Sublime
    (4169648182, 3);

ALTER TABLE "activity" ADD CONSTRAINT "activity_hash_fk" FOREIGN KEY ("hash") REFERENCES "activity_definition"("hash");
DROP TABLE "raid_definition_old";

ALTER TABLE "player_activity" RENAME TO "activity_player";
