
CREATE TABLE "season" (
    "id" INTEGER NOT NULL PRIMARY KEY,
    "short_name" TEXT NOT NULL,
    "long_name" TEXT NOT NULL,
    "dlc" TEXT NOT NULL,
    "start_date" TIMESTAMP(0) WITH TIME ZONE NOT NULL
);
CREATE INDEX "season_idx_date" ON season(start_date DESC);

CREATE OR REPLACE FUNCTION get_season(sd TIMESTAMP)
RETURNS INTEGER AS $$
DECLARE
    season_id INTEGER;
BEGIN
    SELECT ("id") INTO season_id FROM "season"
    WHERE "season"."start_date" AT TIME ZONE 'UTC' < sd 
    ORDER BY "season"."start_date" AT TIME ZONE 'UTC' DESC 
    LIMIT 1;

    RETURN season_id;
END;
$$ LANGUAGE plpgsql
IMMUTABLE;

ALTER TABLE "activity"
ADD COLUMN "season_id" INTEGER GENERATED ALWAYS AS (get_season(date_started AT TIME ZONE 'UTC')) STORED;