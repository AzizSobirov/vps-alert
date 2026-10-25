// Package restore is the interactive "vps-alert restore" command. It runs
// only from a shell on the server; the bot has no way to trigger it.
package restore

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/backup"
	"github.com/AzizSobirov/vps-alert/internal/dbdump"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

var errAborted = errors.New("aborted")

type restorer struct {
	backup *backup.Manager
	notify *telegram.Notifier
	in     *bufio.Reader
	out    io.Writer
}

func Run(ctx context.Context, bm *backup.Manager, notify *telegram.Notifier) error {
	r := &restorer{backup: bm, notify: notify, in: bufio.NewReader(os.Stdin), out: os.Stdout}
	err := r.run(ctx)
	if errors.Is(err, errAborted) {
		fmt.Fprintln(r.out, "Nothing was changed.")
		return nil
	}
	return err
}

func (r *restorer) run(ctx context.Context) error {
	if err := r.backup.Available(); err != nil {
		return err
	}
	cfg := r.backup.Cfg

	target := "local"
	if cfg.R2Enabled() {
		i, err := r.choose("Restore from", []string{
			"local  " + cfg.LocalRepo,
			"r2     " + cfg.R2Bucket + "/" + cfg.R2Prefix,
		})
		if err != nil {
			return err
		}
		if i == 1 {
			target = "r2"
		}
	}

	fmt.Fprintln(r.out, "Loading snapshots…")
	snaps, err := r.backup.Snapshots(ctx, target, 20)
	if err != nil {
		return err
	}
	if len(snaps) == 0 {
		return fmt.Errorf("no snapshots in %s", target)
	}
	var opts []string
	for _, s := range snaps {
		size := ""
		if s.Summary != nil {
			size = metrics.Bytes(s.Summary.TotalBytesProcessed)
		}
		opts = append(opts, fmt.Sprintf("%s  %s  %9s", s.ShortID, s.Time.Local().Format("2006-01-02 15:04"), size))
	}
	i, err := r.choose("Snapshot (newest first)", opts)
	if err != nil {
		return err
	}
	snap := snaps[i]

	dumps := dumpsPath(snap.Paths, r.backup.DumpsDir())
	if dumps == "" {
		return r.files(ctx, target, snap, "")
	}
	i, err = r.choose("What to restore", []string{"Files and folders", "A database"})
	if err != nil {
		return err
	}
	if i == 0 {
		return r.files(ctx, target, snap, dumps)
	}
	return r.database(ctx, target, snap, dumps)
}

func (r *restorer) files(ctx context.Context, target string, snap backup.Snapshot, dumps string) error {
	var paths []string
	for _, p := range snap.Paths {
		if p != dumps {
			paths = append(paths, p)
		}
	}
	i, err := r.choose("Which path", append([]string{"Everything in the snapshot"}, paths...))
	if err != nil {
		return err
	}
	dest := r.ask("Restore into directory", "/var/tmp/vps-alert-restore-"+time.Now().Format("20060102-150405"))

	args := []string{"restore", snap.ShortID, "--target", dest}
	what := "everything"
	if i > 0 {
		what = paths[i-1]
		args = append(args, "--include", what)
	}
	cmd, err := r.backup.Command(ctx, target, args...)
	if err != nil {
		return err
	}
	cmd.Stdout, cmd.Stderr = r.out, r.out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restic restore: %w", err)
	}

	fmt.Fprintf(r.out, "\n✅ Restored into %s\n", dest)
	fmt.Fprintf(r.out, "Files keep their full path, e.g. %s. Nothing on the live system was overwritten; copy what you need into place.\n",
		filepath.Join(dest, firstOr(paths, "/var/www")))
	r.report(ctx, fmt.Sprintf("files (%s) from snapshot %s into %s", what, snap.ShortID, dest))
	return nil
}

func (r *restorer) database(ctx context.Context, target string, snap backup.Snapshot, dumps string) error {
	data, err := r.resticOutput(ctx, target, "dump", snap.ShortID, path.Join(dumps, dbdump.ManifestFile))
	if err != nil {
		return fmt.Errorf("this snapshot has no database list (made by an older version?): %w", err)
	}
	var entries []dbdump.ManifestEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parse %s: %w", dbdump.ManifestFile, err)
	}
	if len(entries) == 0 {
		return errors.New("no databases in this snapshot")
	}
	var opts []string
	for _, e := range entries {
		opts = append(opts, fmt.Sprintf("%-40s %9s", e.DB.Label(), metrics.Bytes(uint64(e.Size))))
	}
	i, err := r.choose("Database", opts)
	if err != nil {
		return err
	}
	entry := entries[i]
	file := path.Join(dumps, entry.File)

	mode, err := r.choose("How", []string{"Import into a database on this server", "Only save the .sql file"})
	if err != nil {
		return err
	}
	if mode == 1 {
		return r.saveDump(ctx, target, snap, file)
	}

	dst := entry.DB
	if dst.Container != "" {
		dst.Container = r.ask("Docker container", dst.Container)
	}
	fmt.Fprintln(r.out, "Checking existing databases…")
	exists, err := dbdump.Exists(ctx, dst)
	if err != nil {
		return err
	}
	def := dst.Name
	if exists {
		// Restoring next to the live database is the safe default.
		def = dst.Name + "_restored"
	}
	if dst.Name = r.ask("Target database name", def); dst.Name != entry.DB.Name {
		if exists, err = dbdump.Exists(ctx, dst); err != nil {
			return err
		}
	}

	if exists {
		fmt.Fprintf(r.out, "\n⚠️  Database %s already exists. ALL of its current data will be deleted.\n", dst.Label())
		if r.ask("Type the database name to confirm", "") != dst.Name {
			return errAborted
		}
		if err := dbdump.Drop(ctx, dst); err != nil {
			return fmt.Errorf("drop %s: %w", dst.Name, err)
		}
	}
	if err := dbdump.Create(ctx, dst); err != nil {
		return fmt.Errorf("create %s: %w", dst.Name, err)
	}

	fmt.Fprintf(r.out, "Importing %s into %s…\n", entry.File, dst.Label())
	dump, err := r.backup.Command(ctx, target, "dump", snap.ShortID, file)
	if err != nil {
		return err
	}
	var dumpErr, importErr bytes.Buffer
	dump.Stderr = &dumpErr
	stdout, err := dump.StdoutPipe()
	if err != nil {
		return err
	}
	if err := dump.Start(); err != nil {
		return err
	}
	ierr := dbdump.Import(ctx, dst, stdout, &importErr)
	io.Copy(io.Discard, stdout) // let restic finish if the import stopped early
	if err := dump.Wait(); err != nil {
		return fmt.Errorf("restic dump: %v: %s", err, strings.TrimSpace(dumpErr.String()))
	}

	errLines := errorLines(importErr.String())
	switch {
	case ierr != nil:
		fmt.Fprintf(r.out, "\n❌ Import failed: %v\n", ierr)
	case len(errLines) > 0:
		fmt.Fprintf(r.out, "\n⚠️  Imported into %s with %d error(s):\n", dst.Label(), len(errLines))
	default:
		fmt.Fprintf(r.out, "\n✅ Imported into %s\n", dst.Label())
	}
	for j, l := range errLines {
		if j == 10 {
			fmt.Fprintf(r.out, "  … and %d more\n", len(errLines)-10)
			break
		}
		fmt.Fprintln(r.out, "  "+l)
	}
	if len(errLines) > 0 && strings.Contains(importErr.String(), "role") {
		fmt.Fprintln(r.out, "Missing roles: create the database users first, then restore again.")
	}
	r.report(ctx, fmt.Sprintf("database %s from snapshot %s into %s", entry.DB.Label(), snap.ShortID, dst.Label()))
	return ierr
}

func (r *restorer) saveDump(ctx context.Context, target string, snap backup.Snapshot, file string) error {
	dest := r.ask("Save to", "/var/tmp/"+path.Base(file[:len(file)-len(".sql")])+"-"+snap.ShortID+".sql")
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	cmd, err := r.backup.Command(ctx, target, "dump", snap.ShortID, file)
	if err != nil {
		f.Close()
		return err
	}
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = f, &stderr
	err = cmd.Run()
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(dest)
		return fmt.Errorf("restic dump: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	fmt.Fprintf(r.out, "\n✅ Saved %s\n", dest)
	r.report(ctx, fmt.Sprintf("dump %s from snapshot %s to %s", path.Base(file), snap.ShortID, dest))
	return nil
}

// report tells the chat that a restore happened: restoring data is a
// notable event even when it is legitimate.
func (r *restorer) report(ctx context.Context, what string) {
	if r.notify == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	user := os.Getenv("SUDO_USER")
	if user == "" {
		user = os.Getenv("USER")
	}
	r.notify.Send(ctx, fmt.Sprintf("♻️ <b>Restore</b> on %s by %s\n%s",
		html.EscapeString(r.backup.Cfg.ServerName), html.EscapeString(user), html.EscapeString(what)))
}

func (r *restorer) resticOutput(ctx context.Context, target string, args ...string) ([]byte, error) {
	cmd, err := r.backup.Command(ctx, target, args...)
	if err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// choose prints numbered options and returns the picked index. Enter picks
// the first option.
func (r *restorer) choose(title string, options []string) (int, error) {
	fmt.Fprintf(r.out, "\n%s:\n", title)
	for i, o := range options {
		fmt.Fprintf(r.out, "  %2d) %s\n", i+1, o)
	}
	for {
		answer := r.ask("Choose", "1")
		if answer == "q" {
			return 0, errAborted
		}
		n, err := strconv.Atoi(answer)
		if err == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
		fmt.Fprintf(r.out, "Enter a number from 1 to %d, or q to quit.\n", len(options))
	}
}

func (r *restorer) ask(question, def string) string {
	if def != "" {
		fmt.Fprintf(r.out, "%s [%s]: ", question, def)
	} else {
		fmt.Fprintf(r.out, "%s: ", question)
	}
	line, _ := r.in.ReadString('\n')
	if line = strings.TrimSpace(line); line == "" {
		return def
	}
	return line
}

// dumpsPath finds the dump directory among a snapshot's paths. It falls back
// to any ".../dumps" path in case DATA_DIR differed when it was taken.
func dumpsPath(paths []string, want string) string {
	for _, p := range paths {
		if p == want {
			return p
		}
	}
	for _, p := range paths {
		if path.Base(p) == "dumps" {
			return p
		}
	}
	return ""
}

func errorLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, "ERROR") {
			out = append(out, strings.TrimSpace(l))
		}
	}
	return out
}

func firstOr(s []string, def string) string {
	if len(s) > 0 {
		return strings.TrimPrefix(s[0], "/")
	}
	return strings.TrimPrefix(def, "/")
}
