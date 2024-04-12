CREATE TABLE "_migrations" (
    "id" INT PRIMARY KEY,
    "name" TEXT NOT NULL,
    "applied_at" TIMESTAMP DEFAULT NOW()
);