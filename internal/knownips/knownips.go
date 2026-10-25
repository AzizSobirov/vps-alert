// Package knownips remembers the addresses that have logged in over SSH, so
// a login from an address never seen before can be flagged.
//
// The file is written by both the PAM hook and the service; each write
// replaces it atomically, so it is never left half written.
package knownips

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// FileName is the list's name inside the data directory.
const FileName = "known-ips.json"

// Keep the most recently seen addresses only, so the file stays small.
const maxEntries = 1000

// Load returns the known addresses with when each was last seen. exists is
// false when the list has never been written.
func Load(path string) (ips map[string]time.Time, exists bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]time.Time{}, false
	}
	ips = map[string]time.Time{}
	if json.Unmarshal(data, &ips) != nil {
		return map[string]time.Time{}, true
	}
	return ips, true
}

// Add records logins from the given addresses at time t.
func Add(path string, t time.Time, addrs ...string) error {
	if len(addrs) == 0 {
		return nil
	}
	ips, _ := Load(path)
	for _, a := range addrs {
		if a != "" && t.After(ips[a]) {
			ips[a] = t
		}
	}
	return save(path, ips)
}

func save(path string, ips map[string]time.Time) error {
	if len(ips) > maxEntries {
		keys := make([]string, 0, len(ips))
		for k := range ips {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return ips[keys[i]].After(ips[keys[j]]) })
		for _, k := range keys[maxEntries:] {
			delete(ips, k)
		}
	}
	data, err := json.Marshal(ips)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".known-ips-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	os.Chmod(tmp.Name(), 0o600)
	return os.Rename(tmp.Name(), path)
}
