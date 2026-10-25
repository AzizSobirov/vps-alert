package dbdump

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AzizSobirov/vps-alert/internal/settings"
)

// fakeDocker puts a docker stub first (and alone) on PATH.
func fakeDocker(t *testing.T, script string) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestEngineForImage(t *testing.T) {
	for image, want := range map[string]string{
		"postgres:16-alpine":           Postgres,
		"postgis/postgis:16-3.4":       Postgres,
		"timescale/timescaledb:latest": Postgres,
		"mariadb:11":                   MySQL,
		"ghcr.io/org/mysql:8@sha256:x": MySQL,
		"nginx:latest":                 "",
		"myorg/postgres-backup:1":      Postgres,
	} {
		if got := EngineForImage(image); got != want {
			t.Errorf("EngineForImage(%q) = %q, want %q", image, got, want)
		}
	}
}

func TestDiscoverDocker(t *testing.T) {
	fakeDocker(t, `case "$1" in
ps) printf 'shop-db\tpostgres:16\nweb\tnginx:latest\nblog-db\tmariadb:11\n' ;;
exec) if [ "$2" = shop-db ]; then printf 'shop\nanalytics\n'; else printf 'information_schema\nwordpress\nmysql\n'; fi ;;
esac`)
	found, warnings := Discover(context.Background())
	want := []settings.Database{
		{Engine: Postgres, Container: "shop-db", Name: "shop"},
		{Engine: Postgres, Container: "shop-db", Name: "analytics"},
		{Engine: MySQL, Container: "blog-db", Name: "wordpress"},
	}
	if len(warnings) != 0 {
		t.Errorf("warnings: %v", warnings)
	}
	if len(found) != len(want) {
		t.Fatalf("found %v, want %v", found, want)
	}
	for i := range want {
		if found[i] != want[i] {
			t.Errorf("found[%d] = %v, want %v", i, found[i], want[i])
		}
	}
}

func TestDumpPassesNameAsArgument(t *testing.T) {
	// The stub prints its arguments, one per line, as the "dump".
	fakeDocker(t, `for a in "$@"; do printf '%s\n' "$a"; done`)
	dir := t.TempDir()
	db := settings.Database{Engine: Postgres, Container: "shop-db", Name: "shop; rm -rf /"}

	size, err := Dump(context.Background(), dir, db)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "docker-shop-db", "shop__rm_-rf__.sql"))
	if int64(len(data)) != size || size == 0 {
		t.Fatalf("size %d, file %d bytes", size, len(data))
	}
	args := strings.Split(strings.TrimSpace(string(data)), "\n")
	if last := args[len(args)-1]; last != db.Name {
		t.Errorf("name not passed as a separate argument, last arg %q", last)
	}
}

func TestDumpFailureRemovesFile(t *testing.T) {
	fakeDocker(t, `echo partial; echo 'pg_dump: error: database "x" does not exist' >&2; exit 1`)
	dir := t.TempDir()
	db := settings.Database{Engine: Postgres, Container: "c", Name: "x"}

	_, err := Dump(context.Background(), dir, db)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, FileName(db))); !os.IsNotExist(err) {
		t.Error("partial dump left behind")
	}
}

func TestDumpRejectsOptionLikeName(t *testing.T) {
	if _, err := Dump(context.Background(), t.TempDir(), settings.Database{Engine: MySQL, Name: "--all"}); err == nil {
		t.Error("expected error")
	}
}
