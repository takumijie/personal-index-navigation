ALTER TABLE music_tracks ADD COLUMN external_id TEXT NOT NULL DEFAULT '';
ALTER TABLE music_tracks ADD COLUMN object_key TEXT NOT NULL DEFAULT '';
ALTER TABLE music_tracks ADD COLUMN lyric_path TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_music_external ON music_tracks(source, external_id);
CREATE INDEX IF NOT EXISTS idx_music_object ON music_tracks(source, object_key);

CREATE TABLE IF NOT EXISTS music_provider_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL UNIQUE,
    data TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
