package backup

import "testing"

func TestParseSummary(t *testing.T) {
	out := []byte(`{"message_type":"status","percent_done":0.5}
{"message_type":"summary","files_new":12,"data_added":2048,"total_bytes_processed":1048576,"snapshot_id":"abcdef0123456789"}
`)
	var r Result
	parseSummary(out, &r)
	if r.SnapshotID != "abcdef01" || r.FilesNew != 12 || r.DataAdded != 2048 || r.TotalBytes != 1048576 {
		t.Errorf("got %+v", r)
	}
}
