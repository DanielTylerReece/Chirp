package db

import (
	"database/sql"
	"fmt"
)

// MediaCacheEntry represents a cached media file tracked in the database.
type MediaCacheEntry struct {
	MediaID   string
	LocalPath string
	MimeType  string
	CachedAt  int64
	SizeBytes int64
}

// AddMediaCache records a cached media file.
func (db *DB) AddMediaCache(entry *MediaCacheEntry) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO media_cache (media_id, local_path, mime_type, cached_at, size_bytes)
		VALUES (?, ?, ?, ?, ?)
	`, entry.MediaID, entry.LocalPath, entry.MimeType, entry.CachedAt, entry.SizeBytes)
	if err != nil {
		return fmt.Errorf("add media cache: %w", err)
	}
	return nil
}

// GetMediaCache gets the local path for a cached media file. Returns nil if not cached.
func (db *DB) GetMediaCache(mediaID string) (*MediaCacheEntry, error) {
	var e MediaCacheEntry
	err := db.QueryRow(`
		SELECT media_id, local_path, mime_type, cached_at, size_bytes
		FROM media_cache WHERE media_id = ?
	`, mediaID).Scan(&e.MediaID, &e.LocalPath, &e.MimeType, &e.CachedAt, &e.SizeBytes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get media cache: %w", err)
	}
	return &e, nil
}

// ListMediaCache returns all cache entries, oldest first.
func (db *DB) ListMediaCache() ([]MediaCacheEntry, error) {
	rows, err := db.Query(`
		SELECT media_id, local_path, mime_type, cached_at, size_bytes
		FROM media_cache ORDER BY cached_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list media cache: %w", err)
	}
	defer rows.Close()

	var entries []MediaCacheEntry
	for rows.Next() {
		var e MediaCacheEntry
		if err := rows.Scan(&e.MediaID, &e.LocalPath, &e.MimeType, &e.CachedAt, &e.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan media cache: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// DeleteMediaCache removes a cache entry (call after deleting the file).
func (db *DB) DeleteMediaCache(mediaID string) error {
	_, err := db.Exec(`DELETE FROM media_cache WHERE media_id = ?`, mediaID)
	if err != nil {
		return fmt.Errorf("delete media cache: %w", err)
	}
	return nil
}

// MediaCacheSize returns total size of cached media in bytes.
func (db *DB) MediaCacheSize() (int64, error) {
	var total int64
	err := db.QueryRow(`SELECT COALESCE(SUM(size_bytes), 0) FROM media_cache`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("media cache size: %w", err)
	}
	return total, nil
}

// EvictMediaCache deletes oldest entries until total size is under maxBytes.
// Returns the list of local paths that should be deleted from disk.
func (db *DB) EvictMediaCache(maxBytes int64) ([]string, error) {
	total, err := db.MediaCacheSize()
	if err != nil {
		return nil, err
	}

	if total <= maxBytes {
		return nil, nil
	}

	entries, err := db.ListMediaCache() // oldest first
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, e := range entries {
		if total <= maxBytes {
			break
		}
		if err := db.DeleteMediaCache(e.MediaID); err != nil {
			return paths, fmt.Errorf("evict %s: %w", e.MediaID, err)
		}
		paths = append(paths, e.LocalPath)
		total -= e.SizeBytes
	}

	return paths, nil
}
