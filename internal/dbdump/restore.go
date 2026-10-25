package dbdump

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AzizSobirov/vps-alert/internal/settings"
)

// ManifestFile records which dump file belongs to which database, so a
// restore does not have to reverse the sanitized file names.
const ManifestFile = "manifest.json"

type ManifestEntry struct {
	DB   settings.Database `json:"db"`
	File string            `json:"file"` // relative to the dump directory
	Size int64             `json:"size"`
}

func WriteManifest(dir string, results []Result) error {
	entries := []ManifestEntry{}
	for _, r := range results {
		if r.Err == nil {
			entries = append(entries, ManifestEntry{DB: r.DB, File: FileName(r.DB), Size: r.Size})
		}
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ManifestFile), data, 0o600)
}

// Exists reports whether db exists on its server.
func Exists(ctx context.Context, db settings.Database) (bool, error) {
	var dbs []settings.Database
	var err error
	switch {
	case db.Container != "":
		dbs, err = containerDatabases(ctx, db.Container, db.Engine)
	case db.Engine == Postgres:
		dbs, err = hostPostgres(ctx)
	case db.Engine == MySQL:
		dbs, err = hostMySQL(ctx)
	}
	for _, d := range dbs {
		if d.Name == db.Name {
			return true, nil
		}
	}
	return false, err
}

func Create(ctx context.Context, db settings.Database) error {
	return runClient(ctx, db, "create")
}

func Drop(ctx context.Context, db settings.Database) error {
	return runClient(ctx, db, "drop")
}

// Import feeds the SQL in r into db; the client's error output goes to
// stderr. Like pg_restore, psql continues past errors such as a missing
// role, so the caller should show what was printed.
func Import(ctx context.Context, db settings.Database, r io.Reader, stderr io.Writer) error {
	cmd, err := clientCmd(ctx, db, "import")
	if err != nil {
		return err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = r, io.Discard, stderr
	return cmd.Run()
}

func runClient(ctx context.Context, db settings.Database, op string) error {
	cmd, err := clientCmd(ctx, db, op)
	if err != nil {
		return err
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return commandError(err, out)
	}
	return nil
}

const (
	pgClientScript = `exec "$0" -U "${POSTGRES_USER:-postgres}" "$@"`
	myClientScript = myAuth + `exec "$(command -v mariadb || command -v mysql)" -uroot "$@"`
)

func clientCmd(ctx context.Context, db settings.Database, op string) (*exec.Cmd, error) {
	if db.Name == "" || strings.HasPrefix(db.Name, "-") || strings.ContainsAny(db.Name, "`\x00\n") {
		return nil, fmt.Errorf("invalid database name %q", db.Name)
	}
	quoted := "`" + db.Name + "`" // safe: backticks are rejected above

	var args []string // client binary first, then its arguments
	switch {
	case db.Engine == Postgres && op == "create":
		args = []string{"createdb", db.Name}
	case db.Engine == Postgres && op == "drop":
		args = []string{"dropdb", "--if-exists", db.Name}
	case db.Engine == Postgres && op == "import":
		args = []string{"psql", "-X", "-q", "-d", db.Name}
	case db.Engine == MySQL && op == "create":
		args = []string{"-e", "CREATE DATABASE " + quoted}
	case db.Engine == MySQL && op == "drop":
		args = []string{"-e", "DROP DATABASE IF EXISTS " + quoted}
	case db.Engine == MySQL && op == "import":
		args = []string{db.Name}
	default:
		return nil, errors.New("unsupported database operation")
	}

	switch {
	case db.Container != "" && db.Engine == Postgres:
		// sh -c script $0 $1...: $0 is the client, "$@" its arguments.
		full := append([]string{"exec", "-i", db.Container, "sh", "-c", pgClientScript}, args...)
		return exec.CommandContext(ctx, "docker", full...), nil
	case db.Container != "":
		full := append([]string{"exec", "-i", db.Container, "sh", "-c", myClientScript, "sh"}, args...)
		return exec.CommandContext(ctx, "docker", full...), nil
	case db.Engine == Postgres:
		return exec.CommandContext(ctx, "runuser", append([]string{"-u", "postgres", "--"}, args...)...), nil
	default:
		return exec.CommandContext(ctx, firstBin("mariadb", "mysql"), args...), nil
	}
}
