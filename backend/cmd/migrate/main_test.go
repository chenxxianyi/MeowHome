package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadMigrationStopsBeforeEmbeddedDownSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "001_initial.up.sql")
	content := "-- +migrate Up\nCREATE TABLE cats (id INT);\n-- +migrate Down\nDROP TABLE cats;\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	body, err := loadMigration(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if body != "CREATE TABLE cats (id INT);" {
		t.Fatalf("up migration body = %q", body)
	}
}

func TestListMigrationsSortsAndParsesVersions(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"015_vaccine.up.sql", "003_records.up.sql", "015_vaccine.down.sql", "003_records.down.sql"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("SELECT 1;\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	files, err := listMigrations(dir)
	if err != nil {
		t.Fatal(err)
	}
	versions := []int{files[0].Version, files[1].Version}
	names := []string{files[0].Name, files[1].Name}
	if !reflect.DeepEqual(versions, []int{3, 15}) || !reflect.DeepEqual(names, []string{"003_records.up.sql", "015_vaccine.up.sql"}) {
		t.Fatalf("migrations = %+v", files)
	}
	if filepath.Base(files[1].DownPath) != "015_vaccine.down.sql" {
		t.Fatalf("down path = %q", files[1].DownPath)
	}
}

func TestMigrationNumberRejectsInvalidNames(t *testing.T) {
	for _, name := range []string{"initial.up.sql", "000_initial.up.sql", "abc_initial.up.sql"} {
		if _, err := migrationNumber(name); err == nil {
			t.Errorf("migrationNumber(%q) accepted invalid version", name)
		}
	}
}
