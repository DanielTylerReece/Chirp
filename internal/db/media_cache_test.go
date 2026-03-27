package db

import (
	"fmt"
	"testing"
)

func TestMediaCacheAddAndGet(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	entry := &MediaCacheEntry{
		MediaID:   "media-1",
		LocalPath: "/tmp/cache/media-1.jpg",
		MimeType:  "image/jpeg",
		CachedAt:  1700000000,
		SizeBytes: 1024 * 100, // 100KB
	}

	if err := db.AddMediaCache(entry); err != nil {
		t.Fatalf("AddMediaCache: %v", err)
	}

	got, err := db.GetMediaCache("media-1")
	if err != nil {
		t.Fatalf("GetMediaCache: %v", err)
	}
	if got == nil {
		t.Fatal("expected entry, got nil")
	}
	if got.MediaID != entry.MediaID {
		t.Errorf("MediaID: got %q, want %q", got.MediaID, entry.MediaID)
	}
	if got.LocalPath != entry.LocalPath {
		t.Errorf("LocalPath: got %q, want %q", got.LocalPath, entry.LocalPath)
	}
	if got.MimeType != entry.MimeType {
		t.Errorf("MimeType: got %q, want %q", got.MimeType, entry.MimeType)
	}
	if got.CachedAt != entry.CachedAt {
		t.Errorf("CachedAt: got %d, want %d", got.CachedAt, entry.CachedAt)
	}
	if got.SizeBytes != entry.SizeBytes {
		t.Errorf("SizeBytes: got %d, want %d", got.SizeBytes, entry.SizeBytes)
	}
}

func TestMediaCacheGetUncached(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	got, err := db.GetMediaCache("nonexistent")
	if err != nil {
		t.Fatalf("GetMediaCache: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for uncached media, got %+v", got)
	}
}

func TestMediaCacheSize(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	// Empty cache should be 0
	size, err := db.MediaCacheSize()
	if err != nil {
		t.Fatalf("MediaCacheSize: %v", err)
	}
	if size != 0 {
		t.Errorf("expected 0 size for empty cache, got %d", size)
	}

	// Add entries
	for i := 0; i < 3; i++ {
		if err := db.AddMediaCache(&MediaCacheEntry{
			MediaID:   fmt.Sprintf("media-%d", i),
			LocalPath: fmt.Sprintf("/tmp/cache/media-%d.jpg", i),
			MimeType:  "image/jpeg",
			CachedAt:  int64(1700000000 + i),
			SizeBytes: 1000,
		}); err != nil {
			t.Fatalf("AddMediaCache %d: %v", i, err)
		}
	}

	size, err = db.MediaCacheSize()
	if err != nil {
		t.Fatalf("MediaCacheSize: %v", err)
	}
	if size != 3000 {
		t.Errorf("expected 3000 bytes, got %d", size)
	}
}

func TestMediaCacheEvict(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	// Add 5 entries totaling 500MB
	for i := 0; i < 5; i++ {
		if err := db.AddMediaCache(&MediaCacheEntry{
			MediaID:   fmt.Sprintf("media-%d", i),
			LocalPath: fmt.Sprintf("/tmp/cache/media-%d.dat", i),
			MimeType:  "application/octet-stream",
			CachedAt:  int64(1700000000 + i), // media-0 is oldest
			SizeBytes: 100 * 1024 * 1024,     // 100MB each
		}); err != nil {
			t.Fatalf("AddMediaCache %d: %v", i, err)
		}
	}

	// Verify total is 500MB
	size, err := db.MediaCacheSize()
	if err != nil {
		t.Fatalf("MediaCacheSize: %v", err)
	}
	if size != 500*1024*1024 {
		t.Fatalf("expected 500MB, got %d", size)
	}

	// Evict to 200MB — should remove 3 oldest (media-0, media-1, media-2)
	paths, err := db.EvictMediaCache(200 * 1024 * 1024)
	if err != nil {
		t.Fatalf("EvictMediaCache: %v", err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 evicted paths, got %d: %v", len(paths), paths)
	}

	// Verify the evicted paths are the oldest entries
	expectedPaths := []string{
		"/tmp/cache/media-0.dat",
		"/tmp/cache/media-1.dat",
		"/tmp/cache/media-2.dat",
	}
	for i, p := range paths {
		if p != expectedPaths[i] {
			t.Errorf("evicted path %d: got %q, want %q", i, p, expectedPaths[i])
		}
	}

	// Verify remaining size is 200MB
	size, err = db.MediaCacheSize()
	if err != nil {
		t.Fatalf("MediaCacheSize after evict: %v", err)
	}
	if size != 200*1024*1024 {
		t.Errorf("expected 200MB remaining, got %d", size)
	}

	// Verify media-3 and media-4 still exist
	for _, id := range []string{"media-3", "media-4"} {
		got, err := db.GetMediaCache(id)
		if err != nil {
			t.Fatalf("GetMediaCache %s: %v", id, err)
		}
		if got == nil {
			t.Errorf("%s should still be cached", id)
		}
	}

	// Verify media-0, media-1, media-2 are gone
	for _, id := range []string{"media-0", "media-1", "media-2"} {
		got, err := db.GetMediaCache(id)
		if err != nil {
			t.Fatalf("GetMediaCache %s: %v", id, err)
		}
		if got != nil {
			t.Errorf("%s should have been evicted", id)
		}
	}
}

func TestMediaCacheEvictUnderLimit(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	// Add a small entry
	if err := db.AddMediaCache(&MediaCacheEntry{
		MediaID:   "small",
		LocalPath: "/tmp/cache/small.jpg",
		SizeBytes: 1000,
		CachedAt:  1700000000,
	}); err != nil {
		t.Fatalf("AddMediaCache: %v", err)
	}

	// Evict with a limit larger than total — should evict nothing
	paths, err := db.EvictMediaCache(1024 * 1024)
	if err != nil {
		t.Fatalf("EvictMediaCache: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected 0 evicted paths, got %d", len(paths))
	}
}

func TestMediaCacheListOrdering(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	// Insert out of order
	timestamps := []int64{1700000003, 1700000001, 1700000005, 1700000002, 1700000004}
	for i, ts := range timestamps {
		if err := db.AddMediaCache(&MediaCacheEntry{
			MediaID:   fmt.Sprintf("media-%d", i),
			LocalPath: fmt.Sprintf("/tmp/cache/media-%d.jpg", i),
			MimeType:  "image/jpeg",
			CachedAt:  ts,
			SizeBytes: 1000,
		}); err != nil {
			t.Fatalf("AddMediaCache %d: %v", i, err)
		}
	}

	entries, err := db.ListMediaCache()
	if err != nil {
		t.Fatalf("ListMediaCache: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}

	// Verify oldest first ordering
	for i := 1; i < len(entries); i++ {
		if entries[i].CachedAt < entries[i-1].CachedAt {
			t.Errorf("entries not in ASC order: index %d (%d) < index %d (%d)",
				i, entries[i].CachedAt, i-1, entries[i-1].CachedAt)
		}
	}

	// Verify the actual order
	expectedOrder := []int64{1700000001, 1700000002, 1700000003, 1700000004, 1700000005}
	for i, e := range entries {
		if e.CachedAt != expectedOrder[i] {
			t.Errorf("entry %d: got CachedAt %d, want %d", i, e.CachedAt, expectedOrder[i])
		}
	}
}

func TestMediaCacheDelete(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	if err := db.AddMediaCache(&MediaCacheEntry{
		MediaID:   "to-delete",
		LocalPath: "/tmp/cache/delete-me.jpg",
		SizeBytes: 5000,
		CachedAt:  1700000000,
	}); err != nil {
		t.Fatalf("AddMediaCache: %v", err)
	}

	if err := db.DeleteMediaCache("to-delete"); err != nil {
		t.Fatalf("DeleteMediaCache: %v", err)
	}

	got, err := db.GetMediaCache("to-delete")
	if err != nil {
		t.Fatalf("GetMediaCache after delete: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}
