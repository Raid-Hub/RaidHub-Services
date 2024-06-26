ALTER TABLE "activity_definition"
ADD COLUMN "path" TEXT NOT NULL,
ADD COLUMN "release_date" TIMESTAMP(0) NOT NULL,
ADD COLUMN "day_one_end" TIMESTAMP(0) GENERATED ALWAYS AS ("release_date" + INTERVAL '1 day') STORED,
ADD COLUMN "contest_end" TIMESTAMP(0),
ADD COLUMN "week_one_end" TIMESTAMP(0),
ADD COLUMN "milestone_hash" BIGINT;

ALTER TABLE "activity_hash"
ADD COLUMN "is_world_first" BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN "release_date_override" TIMESTAMP(0);

ALTER TABLE "version_definition" 
ADD COLUMN "path" TEXT NOT NULL,
ADD COLUMN "is_challenge_mode"BOOLEAN NOT NULL DEFAULT false;