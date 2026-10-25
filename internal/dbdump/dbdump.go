// Package dbdump finds PostgreSQL and MySQL/MariaDB databases on the host and
// in Docker containers, and dumps them to SQL files for restic to back up.
//
// It relies on local passwordless access: peer authentication as the
// postgres user, the MySQL root unix socket, and the credentials that
// official Docker images keep in their environment.
package dbdump

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/settings"
)

const (
	Postgres = "postgres"
	MySQL    = "mysql"
)

const pgListSQL = "SELECT datname FROM pg_database WHERE NOT datistemplate AND datname <> 'postgres' ORDER BY 1"

var mysqlSystemDBs = map[string]bool{
	"information_schema": true, "performance_schema": true, "mysql": true, "sys": true,
}

// Scripts run inside containers with sh -c. The database name is always
// passed as "$1", never interpolated, so odd names cannot inject commands.
const (
	pgListScript = `exec psql -XAt -U "${POSTGRES_USER:-postgres}" -d postgres -c "$1"`
	pgDumpScript = `exec pg_dump -U "${POSTGRES_USER:-postgres}" -d "$1"`

	myAuth = `P="${MARIADB_ROOT_PASSWORD:-${MYSQL_ROOT_PASSWORD:-}}"
F="${MARIADB_ROOT_PASSWORD_FILE:-${MYSQL_ROOT_PASSWORD_FILE:-}}"
if [ -z "$P" ] && [ -n "$F" ]; then P=$(cat "$F"); fi
export MYSQL_PWD="$P"
`
	myListScript = myAuth + `exec "$(command -v mariadb || command -v mysql)" -uroot -N -B -e "SHOW DATABASES"`
	myDumpScript = myAuth + `exec "$(command -v mariadb-dump || command -v mysqldump)" -uroot --single-transaction --quick --routines --triggers --events "$1"`
)

// Discover lists the databases it can reach. Warnings describe engines that
// are present but could not be queried.
func Discover(ctx context.Context) (found []settings.Database, warnings []string) {
	add := func(dbs []settings.Database, err error, where string) {
		found = append(found, dbs...)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", where, err))
		}
	}
	dbs, err := hostPostgres(ctx)
	add(dbs, err, "postgres")
	dbs, err = hostMySQL(ctx)
	add(dbs, err, "mysql")

	containers, err := dockerContainers(ctx)
	if err != nil {
		warnings = append(warnings, "docker: "+err.Error())
	}
	for _, c := range containers {
		dbs, err := containerDatabases(ctx, c.name, c.engine)
		add(dbs, err, "docker:"+c.name)
	}
	return found, warnings
}

func hostPostgres(ctx context.Context) ([]settings.Database, error) {
	if !have("psql") {
		return nil, nil
	}
	if _, err := user.Lookup("postgres"); err != nil {
		return nil, nil // client only, no local server
	}
	out, err := run(ctx, "runuser", "-u", "postgres", "--", "psql", "-XAt", "-c", pgListSQL)
	if err != nil {
		return nil, err
	}
	return toDatabases(Postgres, "", lines(out), nil), nil
}

func hostMySQL(ctx context.Context) ([]settings.Database, error) {
	if !have("mysqld") && !have("mariadbd") {
		return nil, nil
	}
	out, err := run(ctx, firstBin("mariadb", "mysql"), "-N", "-B", "-e", "SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	return toDatabases(MySQL, "", lines(out), mysqlSystemDBs), nil
}

type container struct{ name, engine string }

func dockerContainers(ctx context.Context) ([]container, error) {
	if !have("docker") {
		return nil, nil
	}
	out, err := run(ctx, "docker", "ps", "--format", "{{.Names}}\t{{.Image}}")
	if err != nil {
		return nil, err
	}
	var cs []container
	for _, l := range lines(out) {
		name, image, _ := strings.Cut(l, "\t")
		if engine := EngineForImage(image); engine != "" {
			cs = append(cs, container{name, engine})
		}
	}
	return cs, nil
}

// EngineForImage guesses the database engine from a Docker image name.
func EngineForImage(image string) string {
	// Drop the registry and tag: "ghcr.io/org/postgres:16" -> "postgres".
	image = strings.ToLower(image)
	image = image[strings.LastIndex(image, "/")+1:]
	image, _, _ = strings.Cut(image, ":")
	image, _, _ = strings.Cut(image, "@")
	switch {
	case strings.Contains(image, "postgres"), strings.Contains(image, "postgis"),
		strings.Contains(image, "timescale"), strings.Contains(image, "pgvector"):
		return Postgres
	case strings.Contains(image, "mysql"), strings.Contains(image, "mariadb"), strings.Contains(image, "percona"):
		return MySQL
	}
	return ""
}

func containerDatabases(ctx context.Context, name, engine string) ([]settings.Database, error) {
	if engine == Postgres {
		out, err := run(ctx, "docker", "exec", name, "sh", "-c", pgListScript, "sh", pgListSQL)
		if err != nil {
			return nil, err
		}
		return toDatabases(Postgres, name, lines(out), nil), nil
	}
	out, err := run(ctx, "docker", "exec", name, "sh", "-c", myListScript)
	if err != nil {
		return nil, err
	}
	return toDatabases(MySQL, name, lines(out), mysqlSystemDBs), nil
}

type Result struct {
	DB   settings.Database
	Size int64
	Err  error
}

// DumpAll dumps every database into dir. A failed dump does not stop the
// others.
func DumpAll(ctx context.Context, dir string, dbs []settings.Database) []Result {
	results := make([]Result, 0, len(dbs))
	for _, db := range dbs {
		size, err := Dump(ctx, dir, db)
		results = append(results, Result{DB: db, Size: size, Err: err})
	}
	return results
}

// Dump writes one database to dir/<source>/<name>.sql and returns its size.
func Dump(ctx context.Context, dir string, db settings.Database) (int64, error) {
	if db.Name == "" || strings.HasPrefix(db.Name, "-") {
		return 0, fmt.Errorf("invalid database name %q", db.Name)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()

	var cmd *exec.Cmd
	switch {
	case db.Container != "" && db.Engine == Postgres:
		cmd = exec.CommandContext(ctx, "docker", "exec", db.Container, "sh", "-c", pgDumpScript, "sh", db.Name)
	case db.Container != "" && db.Engine == MySQL:
		cmd = exec.CommandContext(ctx, "docker", "exec", db.Container, "sh", "-c", myDumpScript, "sh", db.Name)
	case db.Engine == Postgres:
		cmd = exec.CommandContext(ctx, "runuser", "-u", "postgres", "--", "pg_dump", "-d", db.Name)
	case db.Engine == MySQL:
		cmd = exec.CommandContext(ctx, firstBin("mariadb-dump", "mysqldump"),
			"--single-transaction", "--quick", "--routines", "--triggers", "--events", db.Name)
	default:
		return 0, fmt.Errorf("unknown engine %q", db.Engine)
	}

	path := filepath.Join(dir, FileName(db))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return 0, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, err
	}
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = f, &stderr
	err = cmd.Run()
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
		return 0, commandError(err, stderr.Bytes())
	}
	st, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return st.Size(), nil
}

var unsafeChars = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// FileName is the dump's path relative to the dump directory.
func FileName(db settings.Database) string {
	source := db.Engine
	if db.Container != "" {
		source = "docker-" + db.Container
	}
	return filepath.Join(unsafeChars.ReplaceAllString(source, "_"), unsafeChars.ReplaceAllString(db.Name, "_")+".sql")
}

func toDatabases(engine, container string, names []string, skip map[string]bool) []settings.Database {
	var dbs []settings.Database
	for _, n := range names {
		if !skip[n] {
			dbs = append(dbs, settings.Database{Engine: engine, Container: container, Name: n})
		}
	}
	return dbs
}

func run(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", commandError(err, stderr.Bytes())
	}
	return stdout.String(), nil
}

func commandError(err error, stderr []byte) error {
	msg := strings.TrimSpace(string(stderr))
	if i := strings.LastIndexByte(msg, '\n'); i >= 0 {
		msg = msg[i+1:] // the last line is usually the actual error
	}
	if len(msg) > 300 {
		msg = msg[:300] + "…"
	}
	if msg == "" {
		return err
	}
	return errors.New(msg)
}

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func have(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func firstBin(names ...string) string {
	for _, n := range names {
		if have(n) {
			return n
		}
	}
	return names[len(names)-1]
}
