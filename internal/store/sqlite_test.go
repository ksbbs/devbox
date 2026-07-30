package store

import (
	"path/filepath"
	"testing"
)

func TestReleaseSourceCRUDAndPersistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "devbox.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	created, err := st.CreateReleaseSource("M7A", "moesnow", "March7thAssistant", "update.7z")
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.Name != "M7A" || created.AssetName != "update.7z" {
		t.Fatalf("unexpected source: %+v", created)
	}
	if _, err := st.CreateReleaseSource("duplicate", "moesnow", "March7thAssistant", "update.7z"); err == nil {
		t.Fatal("expected duplicate source error")
	}
	if _, err := st.CreateReleaseSource("case duplicate", "MOESNOW", "march7thassistant", "update.7z"); err == nil {
		t.Fatal("expected case-insensitive duplicate source error")
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	sources, err := st.ListReleaseSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].ID != created.ID {
		t.Fatalf("source was not persisted: %+v", sources)
	}
	deleted, err := st.DeleteReleaseSource(created.ID)
	if err != nil || !deleted {
		t.Fatalf("delete source: deleted=%v err=%v", deleted, err)
	}
	deleted, err = st.DeleteReleaseSource(created.ID)
	if err != nil || deleted {
		t.Fatalf("delete missing source: deleted=%v err=%v", deleted, err)
	}
}
