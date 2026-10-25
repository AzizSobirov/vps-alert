package restore

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/AzizSobirov/vps-alert/internal/backup"
	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

// fakeTelegram records the text of every message sent.
type fakeTelegram struct {
	mu   sync.Mutex
	msgs []string
}

func (f *fakeTelegram) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.msgs = append(f.msgs, string(body))
	f.mu.Unlock()
	w.Write([]byte(`{"ok":true,"result":{}}`))
}

func (f *fakeTelegram) all() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.msgs, "\n")
}

// A docker stub emulating a postgres container "shop-db" with database
// "shop". Imported SQL is written to the imported file.
const dockerStub = `#!/bin/sh
shift                       # exec
[ "$1" = -i ] && shift
shift                       # container
shift; shift                # sh -c
script="$1"; shift
case "$script" in
*pg_dump*) echo "CREATE TABLE orders (id int); -- dumped $2" ;;
*"psql -XAt"*) echo shop ;;
*) if [ "$1" = psql ]; then cat > %q; fi ;;
esac
`

func setup(t *testing.T) (*backup.Manager, *fakeTelegram, string) {
	if _, err := exec.LookPath("restic"); err != nil {
		t.Skip("restic not installed")
	}
	tmp := t.TempDir()
	www := filepath.Join(tmp, "www")
	os.MkdirAll(www, 0o755)
	os.WriteFile(filepath.Join(www, "index.html"), []byte("hello"), 0o644)

	bin := filepath.Join(tmp, "bin")
	os.MkdirAll(bin, 0o755)
	os.WriteFile(filepath.Join(bin, "docker"), []byte(fmt.Sprintf(dockerStub, filepath.Join(tmp, "imported.sql"))), 0o755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	tg := &fakeTelegram{}
	srv := httptest.NewServer(tg)
	t.Cleanup(srv.Close)

	cfg := &config.Config{
		ServerName:     "test",
		DataDir:        filepath.Join(tmp, "data"),
		ResticBin:      "restic",
		ResticPassword: "test-password",
		LocalRepo:      filepath.Join(tmp, "repo"),
	}
	store, err := settings.Open(cfg.SettingsPath())
	if err != nil {
		t.Fatal(err)
	}
	store.Update(func(s *settings.Settings) error {
		s.Backup.Paths = []string{www}
		s.Backup.Databases = []settings.Database{{Engine: "postgres", Container: "shop-db", Name: "shop"}}
		return nil
	})
	notify := &telegram.Notifier{TG: telegram.NewWithURL("token", srv.URL), ChatID: 1}
	bm := &backup.Manager{Cfg: cfg, Store: store, Notify: notify}

	if err := bm.Run(context.Background(), "test"); err != nil {
		t.Fatalf("backup: %v\nmessages: %s", err, tg.all())
	}
	if !store.Get().Backup.LastOK {
		t.Fatal("LastOK is false")
	}
	if _, err := os.Stat(bm.DumpsDir()); !os.IsNotExist(err) {
		t.Error("dumps were not removed after the backup")
	}
	return bm, tg, tmp
}

func newRestorer(bm *backup.Manager, answers string) (*restorer, *bytes.Buffer) {
	var out bytes.Buffer
	return &restorer{backup: bm, notify: bm.Notify, in: bufio.NewReader(strings.NewReader(answers)), out: &out}, &out
}

func TestBackupThenRestoreDatabase(t *testing.T) {
	bm, tg, tmp := setup(t)
	if msgs := tg.all(); !strings.Contains(msgs, "Backup finished") || !strings.Contains(msgs, "shop") {
		t.Errorf("backup notification: %s", msgs)
	}

	// snapshot 1, "A database", database 1, import, keep container, default name.
	r, out := newRestorer(bm, "\n2\n\n\n\n\n")
	if err := r.run(context.Background()); err != nil {
		t.Fatalf("restore: %v\n%s", err, out)
	}
	if !strings.Contains(out.String(), "shop_restored") {
		t.Errorf("existing database not protected by default name:\n%s", out)
	}
	sql, _ := os.ReadFile(filepath.Join(tmp, "imported.sql"))
	if !strings.Contains(string(sql), "CREATE TABLE orders") {
		t.Errorf("imported SQL = %q\n%s", sql, out)
	}
	if !strings.Contains(tg.all(), "Restore") {
		t.Error("restore was not reported to the chat")
	}
}

func TestBackupThenRestoreFiles(t *testing.T) {
	bm, _, tmp := setup(t)
	dest := filepath.Join(tmp, "restored")
	www := bm.Store.Get().Backup.Paths[0]

	// snapshot 1, "Files and folders", path 2 (www), destination.
	r, out := newRestorer(bm, "\n1\n2\n"+dest+"\n")
	if err := r.run(context.Background()); err != nil {
		t.Fatalf("restore: %v\n%s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(dest, www, "index.html"))
	if err != nil || string(data) != "hello" {
		t.Errorf("restored file: %q, %v\n%s", data, err, out)
	}
}

func TestDatabaseConfirmationRequired(t *testing.T) {
	bm, _, tmp := setup(t)
	// Import into the existing "shop" but type the wrong name to confirm.
	r, out := newRestorer(bm, "\n2\n\n\n\nshop\nwrong\n")
	if err := r.run(context.Background()); err != errAborted {
		t.Fatalf("err = %v, want errAborted\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(tmp, "imported.sql")); !os.IsNotExist(err) {
		t.Error("data was imported without confirmation")
	}
}
