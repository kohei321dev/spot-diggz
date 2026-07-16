CREATE TABLE IF NOT EXISTS sdz_spots (
    spot_id text PRIMARY KEY,
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
    description text NOT NULL DEFAULT '',
    lat double precision NOT NULL CHECK (lat BETWEEN -90 AND 90),
    lng double precision NOT NULL CHECK (lng BETWEEN -180 AND 180),
    tags text[] NOT NULL DEFAULT '{}',
    tag_keys text[] NOT NULL DEFAULT '{}',
    visibility text NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'private', 'unlisted')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_sdz_spots_visibility_active
    ON sdz_spots (visibility)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sdz_spots_lng_lat_active
    ON sdz_spots (lng, lat)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sdz_spots_tag_keys
    ON sdz_spots
    USING gin (tag_keys);
