package wal

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func mustOpen(t *testing.T, dir string) *Journal {
	t.Helper()
	j, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return j
}

func TestOpenEmpty(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	defer func() {
		_ = j.Close()
	}()

	recs, err := j.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("expected 0 records, got %d", len(recs))
	}
}

func TestRoundTripTransaction(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)

	params, _ := json.Marshal(map[string]string{"snapshot": "foo"})
	tx, err := j.Begin(OpSnapCreate, params)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if tx.ID() != 1 {
		t.Fatalf("first txn id should be 1, got %d", tx.ID())
	}
	args, _ := json.Marshal(map[string]string{"target": "h2"})
	if err := tx.Intent(1, ActionCreateHead, args); err != nil {
		t.Fatal(err)
	}
	if err := tx.StepDone(1); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// Scan before Close so we see the records. The Close below will
	// checkpoint-and-truncate the file (no in-flight txns), so a Scan
	// after Close would see an empty journal.
	recs, err := j.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(recs) != 4 {
		t.Fatalf("expected 4 records, got %d", len(recs))
	}
	if recs[0].Type != RecTxnBegin || recs[3].Type != RecTxnCommit {
		t.Fatalf("unexpected record sequence: %+v", recs)
	}

	var begin TxnBeginPayload
	if err := json.Unmarshal(recs[0].Payload, &begin); err != nil {
		t.Fatalf("decode begin: %v", err)
	}
	if begin.Op != OpSnapCreate || begin.TxnID != 1 {
		t.Fatalf("bad begin payload: %+v", begin)
	}

	if err := j.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// After clean close + checkpoint, file is truncated to 0.
	st, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() != 0 {
		t.Fatalf("expected truncated journal, size=%d", st.Size())
	}
}

func TestReopenAfterCrash(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Intent(1, ActionCreateHead, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.StepDone(1); err != nil {
		t.Fatal(err)
	}
	// Simulate crash: drop fd + flock without checkpoint.
	if err := ForceCloseForTest(j); err != nil {
		t.Fatal(err)
	}

	j2 := mustOpen(t, dir)
	defer func() {
		_ = j2.Close()
	}()
	recs, err := j2.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 3 {
		t.Fatalf("expected 3 records (begin, intent, step_done), got %d", len(recs))
	}
}

func TestTornTailTruncated(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := ForceCloseForTest(j); err != nil {
		t.Fatal(err)
	}

	// Append 5 random bytes that look nothing like a record.
	path := filepath.Join(dir, FileName)
	stBefore, _ := os.Stat(path)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte{1, 2, 3, 4, 5}); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	j2 := mustOpen(t, dir)
	defer func() {
		_ = j2.Close()
	}()
	stAfter, _ := os.Stat(path)
	if stAfter.Size() != stBefore.Size() {
		t.Fatalf("expected torn tail truncated to %d, got %d", stBefore.Size(), stAfter.Size())
	}
	recs, err := j2.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}
}

// TestCRCDetectsCorruption verifies that a single corrupted byte in an
// otherwise-complete record is detected as mid-stream corruption: Open
// must refuse so OpenWithQuarantine can rename the file aside for
// inspection rather than silently discarding the durable bytes.
func TestCRCDetectsCorruption(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	tx, err := j.Begin(OpSnapCreate, []byte(`{"x":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := ForceCloseForTest(j); err != nil {
		t.Fatal(err)
	}

	// Flip a byte deep inside the first record's payload.
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < headerSize+5 {
		t.Fatalf("file too short: %d", len(data))
	}
	data[headerSize+2] ^= 0xff
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	// Open must refuse mid-stream corruption (no silent truncation).
	if _, err := Open(dir); err == nil {
		t.Fatal("expected Open to fail on bad CRC")
	}

	// File must be untouched: nothing was truncated.
	stAfter, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if int(stAfter.Size()) != len(data) {
		t.Fatalf("corrupt journal must not be truncated by Open: size=%d want=%d", stAfter.Size(), len(data))
	}

	// OpenWithQuarantine takes over: file is renamed aside and a fresh
	// empty journal is opened in its place.
	j2, info, err := OpenWithQuarantine(dir)
	if err != nil {
		t.Fatalf("OpenWithQuarantine: %v", err)
	}
	defer func() { _ = j2.Close() }()
	if info == nil || info.OpenError == nil {
		t.Fatalf("expected QuarantineInfo with OpenError, got %+v", info)
	}
	if _, err := os.Stat(info.QuarantinedPath); err != nil {
		t.Fatalf("quarantined file must exist: %v", err)
	}
	newSize, _ := j2.Size()
	if newSize != 0 {
		t.Fatalf("replacement journal should be empty, size=%d", newSize)
	}
}

func TestFlockExcludesSecondOpen(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	defer func() {
		_ = j.Close()
	}()

	if _, err := Open(dir); !errors.Is(err, ErrJournalLocked) {
		t.Fatalf("expected ErrJournalLocked, got %v", err)
	}
}

func TestCheckpointTruncates(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	defer func() {
		_ = j.Close()
	}()

	tx, _ := j.Begin(OpSnapCreate, nil)
	_ = tx.Commit()

	sz, _ := j.Size()
	if sz == 0 {
		t.Fatal("expected non-empty journal before checkpoint")
	}
	if err := j.Checkpoint(); err != nil {
		t.Fatal(err)
	}
	sz, _ = j.Size()
	if sz != 0 {
		t.Fatalf("expected 0 size after checkpoint, got %d", sz)
	}

	// nextTxnID should not regress.
	tx2, _ := j.Begin(OpSnapCreate, nil)
	if tx2.ID() <= 1 {
		t.Fatalf("txn ID regressed after checkpoint: %d", tx2.ID())
	}
	_ = tx2.Commit()
}

func TestPrepareThenCrashReplay(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)

	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Intent(1, ActionCreateHead, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.Intent(2, ActionUpdateVolumeMeta, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.Prepare(); err != nil {
		t.Fatal(err)
	}
	// Simulate crash before any STEP_DONE: drop fd + flock without checkpoint.
	if err := ForceCloseForTest(j); err != nil {
		t.Fatal(err)
	}

	j2 := mustOpen(t, dir)
	defer func() { _ = j2.Close() }()
	a, err := j2.Recover()
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(a.Pending) != 1 {
		t.Fatalf("expected 1 pending txn, got %d", len(a.Pending))
	}
	pt := a.Pending[0]
	if !pt.Prepared {
		t.Fatal("expected Prepared=true after PREPARE record")
	}
	if len(pt.PendingIntents) != 2 {
		t.Fatalf("expected 2 intents, got %d", len(pt.PendingIntents))
	}
	if len(pt.CompletedSteps) != 0 {
		t.Fatalf("expected no completed steps, got %v", pt.CompletedSteps)
	}
	// Recover must have advanced nextTxnID so a fresh Begin doesn't collide.
	tx2, err := j2.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if tx2.ID() <= pt.ID {
		t.Fatalf("next txn id %d must be > pending %d", tx2.ID(), pt.ID)
	}
	_ = tx2.Abort()
}

func TestCloseSkipsCheckpointWithInFlightTxn(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)

	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Intent(1, ActionCreateHead, nil); err != nil {
		t.Fatal(err)
	}
	// Close while the txn is still in flight: records must survive so the
	// next Open can recover them.
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() == 0 {
		t.Fatal("expected non-empty journal: in-flight txn must not be checkpointed away")
	}

	j2 := mustOpen(t, dir)
	defer func() { _ = j2.Close() }()
	recs, err := j2.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records (begin, intent), got %d", len(recs))
	}
}

func TestCheckpointRejectsInFlightTxn(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	defer func() { _ = j.Close() }()

	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Checkpoint(); err == nil {
		t.Fatal("Checkpoint must refuse to run with an in-flight txn")
	}
	// Journal must not have been truncated.
	sz, _ := j.Size()
	if sz == 0 {
		t.Fatal("refused checkpoint must leave the journal intact")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// After the txn ends, Checkpoint succeeds.
	if err := j.Checkpoint(); err != nil {
		t.Fatalf("Checkpoint after Commit: %v", err)
	}
}

func TestAdoptTxnReplaysPendingTransaction(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)

	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Intent(1, ActionCreateHead, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.Intent(2, ActionUpdateVolumeMeta, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.Prepare(); err != nil {
		t.Fatal(err)
	}
	pendingID := tx.ID()
	if err := ForceCloseForTest(j); err != nil {
		t.Fatal(err)
	}

	j2 := mustOpen(t, dir)
	a, err := j2.Recover()
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(a.Pending) != 1 || a.Pending[0].ID != pendingID {
		t.Fatalf("unexpected recovery: %+v", a.Pending)
	}

	adopted, err := AdoptTxn(j2, a.Pending[0].ID, a.Pending[0].Op)
	if err != nil {
		t.Fatalf("AdoptTxn: %v", err)
	}
	for _, intent := range a.Pending[0].PendingIntents {
		if err := adopted.StepDone(intent.StepID); err != nil {
			t.Fatalf("StepDone: %v", err)
		}
	}
	if err := adopted.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := j2.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reopen: no pending txns left, file is truncated by the clean Close.
	j3 := mustOpen(t, dir)
	defer func() { _ = j3.Close() }()
	a, err = j3.Recover()
	if err != nil {
		t.Fatalf("Recover #2: %v", err)
	}
	if len(a.Pending) != 0 {
		t.Fatalf("expected no pending after adopted commit, got %+v", a.Pending)
	}
}

func TestCloseSkipsCheckpointWithUnadoptedRecoveredPending(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)

	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Intent(1, ActionCreateHead, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.Prepare(); err != nil {
		t.Fatal(err)
	}
	pendingID := tx.ID()
	// Crash before Commit/Abort.
	if err := ForceCloseForTest(j); err != nil {
		t.Fatal(err)
	}

	// Reopen and Recover but do NOT AdoptTxn the pending txn. A clean
	// Close in this state must not truncate the journal, otherwise the
	// pending txn becomes invisible to any future process.
	j2 := mustOpen(t, dir)
	a, err := j2.Recover()
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(a.Pending) != 1 || a.Pending[0].ID != pendingID {
		t.Fatalf("unexpected recovery: %+v", a.Pending)
	}
	if err := j2.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	st, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() == 0 {
		t.Fatal("expected non-empty journal: unadopted recovered pending txn must not be checkpointed away")
	}

	// Third Open must still see the pending txn so it can be replayed.
	j3 := mustOpen(t, dir)
	defer func() { _ = j3.Close() }()
	a3, err := j3.Recover()
	if err != nil {
		t.Fatalf("Recover #3: %v", err)
	}
	if len(a3.Pending) != 1 || a3.Pending[0].ID != pendingID {
		t.Fatalf("third Open lost pending txn: %+v", a3.Pending)
	}
}

func TestCheckpointRejectsRecoveredPendingTxn(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)

	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Intent(1, ActionCreateHead, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.Prepare(); err != nil {
		t.Fatal(err)
	}
	pendingID := tx.ID()
	if err := ForceCloseForTest(j); err != nil {
		t.Fatal(err)
	}

	j2 := mustOpen(t, dir)
	defer func() { _ = j2.Close() }()
	if _, err := j2.Recover(); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if err := j2.Checkpoint(); err == nil {
		t.Fatal("Checkpoint must refuse to run with an unadopted recovered pending txn")
	}
	// Adopt + resolve the pending txn; Checkpoint must then succeed.
	adopted, err := AdoptTxn(j2, pendingID, OpSnapCreate)
	if err != nil {
		t.Fatalf("AdoptTxn: %v", err)
	}
	if err := adopted.Abort(); err != nil {
		t.Fatalf("Abort: %v", err)
	}
	if err := j2.Checkpoint(); err != nil {
		t.Fatalf("Checkpoint after AdoptTxn+Abort: %v", err)
	}
}

func TestAdoptTxnFailsOnClosedJournal(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptTxn(j, 1, OpSnapCreate); err == nil {
		t.Fatal("AdoptTxn must reject a closed journal")
	}
}

func TestAdoptTxnRejectsUnknownID(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	defer func() { _ = j.Close() }()

	// On a fresh journal nothing has been recovered, so the
	// recovered-pending set is empty and every id is rejected.
	if _, err := AdoptTxn(j, 1, OpSnapCreate); err == nil {
		t.Fatal("AdoptTxn must reject ids not in the recovered-pending set")
	}
	if _, err := AdoptTxn(j, 42, OpSnapCreate); err == nil {
		t.Fatal("AdoptTxn must reject phantom ids")
	}
}

func TestAdoptTxnRejectsFinishedID(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	finishedID := tx.ID()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := ForceCloseForTest(j); err != nil {
		t.Fatal(err)
	}

	j2 := mustOpen(t, dir)
	defer func() { _ = j2.Close() }()
	a, err := j2.Recover()
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(a.Pending) != 0 {
		t.Fatalf("expected no pending after committed txn, got %+v", a.Pending)
	}
	// finishedID < nextTxnID but COMMIT is durable; must NOT be adoptable.
	if _, err := AdoptTxn(j2, finishedID, OpSnapCreate); err == nil {
		t.Fatal("AdoptTxn must reject ids whose COMMIT is already durable")
	}
}

func TestAdoptTxnRejectsDoubleAdopt(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Prepare(); err != nil {
		t.Fatal(err)
	}
	pendingID := tx.ID()
	if err := ForceCloseForTest(j); err != nil {
		t.Fatal(err)
	}

	j2 := mustOpen(t, dir)
	defer func() { _ = j2.Close() }()
	if _, err := j2.Recover(); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if _, err := AdoptTxn(j2, pendingID, OpSnapCreate); err != nil {
		t.Fatalf("first AdoptTxn: %v", err)
	}
	if _, err := AdoptTxn(j2, pendingID, OpSnapCreate); err == nil {
		t.Fatal("second AdoptTxn on the same id must be rejected")
	}
}

func TestRecoverRefusesInFlightTxn(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	closed := false
	defer func() {
		if !closed {
			_ = j.Close()
		}
	}()

	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Recover must refuse: it would otherwise install tx.ID in the
	// recovered-pending set and never let the journal checkpoint after
	// tx commits.
	if _, err := j.Recover(); err == nil {
		t.Fatal("Recover must refuse to run with an in-flight txn")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// After the txn is resolved, Recover succeeds.
	if _, err := j.Recover(); err != nil {
		t.Fatalf("Recover after Commit: %v", err)
	}
	// And Close can now checkpoint+truncate the journal cleanly.
	if err := j.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	closed = true
}

func TestTxnRejectsWritesAfterJournalClose(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Close the journal while the txn is still in flight; subsequent
	// writes through tx must fail cleanly with "journal is closed"
	// rather than panicking on a closed *os.File.
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Intent(1, ActionCreateHead, nil); err == nil {
		t.Fatal("Intent on closed journal must return an error")
	}
	if err := tx.StepDone(1); err == nil {
		t.Fatal("StepDone on closed journal must return an error")
	}
	if err := tx.Prepare(); err == nil {
		t.Fatal("Prepare on closed journal must return an error")
	}
	if err := tx.Commit(); err == nil {
		t.Fatal("Commit on closed journal must return an error")
	}
}

func TestPrepareIsNotIdempotent(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	defer func() { _ = j.Close() }()

	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Prepare(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Prepare(); err == nil {
		t.Fatal("second Prepare must return an error")
	}
	// Only one TXN_PREPARE record should be on disk.
	recs, err := j.Scan()
	if err != nil {
		t.Fatal(err)
	}
	prepCount := 0
	for _, r := range recs {
		if r.Type == RecTxnPrepare {
			prepCount++
		}
	}
	if prepCount != 1 {
		t.Fatalf("expected exactly 1 TXN_PREPARE record on disk, got %d", prepCount)
	}
	_ = tx.Commit()
}

func TestBeginDoesNotLeakIDOnAppendFailure(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	defer func() { _ = j.Close() }()

	// Close the underlying fd so the next appendRecordLocked fails.
	if err := j.f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Begin(OpSnapCreate, nil); err == nil {
		t.Fatal("Begin should fail when the underlying fd is closed")
	}
	// nextTxnID must not have been bumped.
	if j.nextTxnID != 1 {
		t.Fatalf("nextTxnID leaked: got %d want 1", j.nextTxnID)
	}
}

// TestOpenWithQuarantineConcurrentRace exercises the rename window:
// many goroutines race to OpenWithQuarantine the same corrupt journal.
// Exactly one of them must end up holding a working journal; the
// others must report a clean error and must not have produced a
// split-brain (two journal.log inodes claiming to be the live WAL).
func TestOpenWithQuarantineConcurrentRace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)

	// Plant a corrupt journal: valid magic+version+payload-len, garbage CRC.
	j := mustOpen(t, dir)
	tx, err := j.Begin(OpSnapCreate, []byte(`{"x":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := ForceCloseForTest(j); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[headerSize+2] ^= 0xff
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	const goroutines = 8
	results := make(chan struct {
		j    *Journal
		info *QuarantineInfo
		err  error
	}, goroutines)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			jj, info, err := OpenWithQuarantine(dir)
			results <- struct {
				j    *Journal
				info *QuarantineInfo
				err  error
			}{jj, info, err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	winners := 0
	var winner *Journal
	quarantineCount := 0
	for r := range results {
		if r.j != nil {
			winners++
			winner = r.j
		}
		if r.info != nil {
			quarantineCount++
		}
		if r.err != nil && !errors.Is(r.err, ErrJournalLocked) {
			t.Errorf("unexpected error from concurrent OpenWithQuarantine: %v", r.err)
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly 1 winning OpenWithQuarantine, got %d", winners)
	}
	if quarantineCount < 1 {
		t.Fatalf("expected at least 1 quarantine attempt, got %d", quarantineCount)
	}

	// The winner must own a fresh, empty, lockable journal.
	sz, _ := winner.Size()
	if sz != 0 {
		t.Fatalf("winner journal should be empty, size=%d", sz)
	}
	// No other process can open it.
	if _, err := Open(dir); !errors.Is(err, ErrJournalLocked) {
		t.Fatalf("expected ErrJournalLocked from a second Open after race resolution, got %v", err)
	}
	_ = winner.Close()
}

func TestRecordTypeString(t *testing.T) {
	cases := map[RecordType]string{
		RecTxnBegin:   "TXN_BEGIN",
		RecIntent:     "INTENT",
		RecStepDone:   "STEP_DONE",
		RecTxnCommit:  "TXN_COMMIT",
		RecTxnAbort:   "TXN_ABORT",
		RecCheckpoint: "CHECKPOINT",
		RecTxnPrepare: "TXN_PREPARE",
		RecordType(0): "UNKNOWN",
	}
	for typ, want := range cases {
		if got := typ.String(); got != want {
			t.Errorf("%d: got %q want %q", typ, got, want)
		}
	}
}

func TestScanFileReadOnlyDoesNotTakeFlock(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Intent(1, ActionCreateHead, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.Prepare(); err != nil {
		t.Fatal(err)
	}

	// Journal still open and flocked. ScanFile must not block.
	path := filepath.Join(dir, FileName)
	recs, err := ScanFile(path)
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("expected 3 records, got %d", len(recs))
	}
	if recs[0].Type != RecTxnBegin || recs[1].Type != RecIntent || recs[2].Type != RecTxnPrepare {
		t.Fatalf("unexpected types: %+v", recs)
	}
	_ = tx.Commit()
	_ = j.Close()
}

func TestScanFileTornTailIsClean(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Intent(1, ActionCreateHead, nil); err != nil {
		t.Fatal(err)
	}
	_ = tx.Commit()

	// Drop locks so we can corrupt.
	if err := ForceCloseForTest(j); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, FileName)
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Append 7 garbage bytes (less than a header).
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte{1, 2, 3, 4, 5, 6, 7}); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	recs, err := ScanFile(path)
	if err != nil {
		t.Fatalf("ScanFile should silently stop at torn tail, got %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("expected 3 valid records, got %d (size before corruption=%d)", len(recs), st.Size())
	}
}

// TestOpenWithQuarantineRenamesUnreadableFile makes journal.log a
// directory so the underlying os.OpenFile in Open() fails with EISDIR.
// OpenWithQuarantine should rename it aside and succeed against a
// fresh empty file.
func TestOpenWithQuarantineRenamesUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, FileName)

	// Make the journal path a directory: O_RDWR open fails.
	if err := os.Mkdir(src, 0700); err != nil {
		t.Fatal(err)
	}
	// Drop a marker inside so we can confirm the original tree is preserved.
	marker := filepath.Join(src, "marker")
	if err := os.WriteFile(marker, []byte("evidence"), 0600); err != nil {
		t.Fatal(err)
	}

	j, info, err := OpenWithQuarantine(dir)
	if err != nil {
		t.Fatalf("OpenWithQuarantine: %v", err)
	}
	defer func() {
		_ = j.Close()
	}()
	if info == nil {
		t.Fatal("expected QuarantineInfo, got nil")
	}
	if info.OpenError == nil {
		t.Fatal("expected non-nil OpenError")
	}
	if info.OriginalPath != src {
		t.Fatalf("OriginalPath: got %q want %q", info.OriginalPath, src)
	}

	// Quarantined path must exist with the original contents.
	st, err := os.Stat(info.QuarantinedPath)
	if err != nil {
		t.Fatalf("stat quarantined: %v", err)
	}
	if !st.IsDir() {
		t.Fatal("expected quarantined path to still be the original (dir)")
	}
	if _, err := os.Stat(filepath.Join(info.QuarantinedPath, "marker")); err != nil {
		t.Fatalf("marker missing in quarantined dir: %v", err)
	}
	// A new regular journal file should now exist in place of the original.
	stNew, err := os.Stat(src)
	if err != nil {
		t.Fatalf("stat new journal: %v", err)
	}
	if stNew.IsDir() {
		t.Fatal("new journal path should be a regular file, not a directory")
	}

	// Fresh journal must be writable.
	tx, err := j.Begin(OpSnapCreate, nil)
	if err != nil {
		t.Fatalf("Begin on fresh quarantined journal: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

// TestOpenWithQuarantinePreservesLockError ensures we never quarantine
// a journal that another process owns.
func TestOpenWithQuarantinePreservesLockError(t *testing.T) {
	dir := t.TempDir()
	j := mustOpen(t, dir)
	defer func() {
		_ = j.Close()
	}()

	_, info, err := OpenWithQuarantine(dir)
	if !errors.Is(err, ErrJournalLocked) {
		t.Fatalf("expected ErrJournalLocked, got %v", err)
	}
	if info != nil {
		t.Fatal("must not quarantine a flock-contended journal")
	}
	// Original journal file must still exist with the same name.
	if _, err := os.Stat(filepath.Join(dir, FileName)); err != nil {
		t.Fatalf("original journal must be untouched: %v", err)
	}
	// No .broken-* sibling created.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), FileName+".broken-") {
			t.Fatalf("must not produce sibling %q on lock contention", e.Name())
		}
	}
}

// TestOpenWithQuarantineNoFilePropagatesError ensures we don't pretend
// to quarantine when there's nothing to rename (e.g. parent dir missing
// or other Open failure unrelated to file content).
func TestOpenWithQuarantineNoFilePropagatesError(t *testing.T) {
	// Non-existent directory: Open will fail to create the file.
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	_, info, err := OpenWithQuarantine(missing)
	if err == nil {
		t.Fatal("expected error for missing parent directory")
	}
	if info != nil {
		t.Fatalf("must not quarantine when source file does not exist; got %+v", info)
	}
}
