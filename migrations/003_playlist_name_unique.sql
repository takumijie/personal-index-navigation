UPDATE playlists SET name = TRIM(name);

DELETE FROM playlist_tracks
WHERE playlist_id IN (
    SELECT id FROM playlists
    WHERE id NOT IN (SELECT MIN(id) FROM playlists GROUP BY LOWER(name))
);

DELETE FROM playlists
WHERE id NOT IN (SELECT MIN(id) FROM playlists GROUP BY LOWER(name));

CREATE UNIQUE INDEX IF NOT EXISTS idx_playlists_name_unique ON playlists(LOWER(TRIM(name)));
