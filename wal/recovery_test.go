package wal

import (
	"encoding/json"
	"testing"
)

func mkRecord(t *testing.T, typ RecordType, v any) Record {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return Record{Type: typ, Payload: b}
}

// writeJournalFile writes recs to a real on-disk journal in dir using
// the production framing/CRC/fsync path, then closes it abruptly. After
// it returns the bytes are durable so a subsequent Open observes exactly
// these records. Records are appended verbatim (bypassing the Txn API)
// so recovery-level malformed sequences (e.g. an INTENT without a
// TXN_BEGIN, or a duplicate TXN_BEGIN) can be laid down on disk.
func writeJournalFile(t *testing.T, dir string, recs []Record) {
	t.Helper()
	j := mustOpen(t, dir)
	j.mu.Lock()
	for _, r := range recs {
		if err := j.appendRecordLocked(r.Type, r.Payload); err != nil {
			j.mu.Unlock()
			t.Fatalf("append %s record: %v", r.Type, err)
		}
	}
	j.mu.Unlock()
	if err := ForceCloseForTest(j); err != nil {
		t.Fatalf("ForceCloseForTest: %v", err)
	}
}

// analyzeJournalFile reopens the on-disk journal in dir, scans the framed
// records back off disk, and runs Analyze on them. This exercises the
// full write -> fsync -> reopen -> binary-decode -> analyze path rather
// than analyzing hand-built in-memory records.
func analyzeJournalFile(t *testing.T, dir string) (*Analysis, error) {
	t.Helper()
	j := mustOpen(t, dir)
	defer func() { _ = ForceCloseForTest(j) }()
	recs, err := j.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return Analyze(recs)
}

func TestAnalyzeEmpty(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, nil)

	a, err := analyzeJournalFile(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Pending) != 0 {
		t.Fatalf("expected no pending, got %+v", a.Pending)
	}
	if a.NextTxnID != 1 {
		t.Fatalf("expected next id 1, got %d", a.NextTxnID)
	}
}

func TestAnalyzeFinishedAndPending(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecTxnPrepare, TxnEndPayload{TxnID: 1}),
		mkRecord(t, RecStepDone, StepDonePayload{TxnID: 1, StepID: 1}),
		mkRecord(t, RecTxnCommit, TxnEndPayload{TxnID: 1}),

		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 2, Op: OpSnapRevert}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 2, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecStepDone, StepDonePayload{TxnID: 2, StepID: 1}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 2, StepID: 2, Action: ActionUpdateVolumeMeta}),
		// crash before STEP_DONE for step 2

		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 3, Op: OpSnapRemoveMark}),
		// crash before any intent
	})

	a, err := analyzeJournalFile(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.NextTxnID != 4 {
		t.Fatalf("next id: got %d want 4", a.NextTxnID)
	}
	if len(a.Pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(a.Pending))
	}

	// Pending[0]: txn 2 — has intent without step_done.
	p := a.Pending[0]
	if p.ID != 2 || p.Op != OpSnapRevert {
		t.Fatalf("pending[0]: %+v", p)
	}
	if !p.CompletedSteps[1] || p.CompletedSteps[2] {
		t.Fatalf("pending[0] completed: %+v", p.CompletedSteps)
	}
	if p.LastIntent == nil || p.LastIntent.StepID != 2 {
		t.Fatalf("pending[0] last intent: %+v", p.LastIntent)
	}

	// Pending[1]: txn 3 — no intents, safe to abort.
	p = a.Pending[1]
	if p.ID != 3 || p.LastIntent != nil {
		t.Fatalf("pending[1]: %+v", p)
	}
}

func TestAnalyzeUnknownTxnID(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecIntent, IntentPayload{TxnID: 99, StepID: 1, Action: ActionCreateHead}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for INTENT without TXN_BEGIN")
	}
}

func TestAnalyzeRejectsDuplicateBegin(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapRevert}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for duplicate TXN_BEGIN")
	}
}

func TestAnalyzeRejectsDuplicateIntentStep(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 1, Action: ActionUpdateVolumeMeta}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for duplicate INTENT step id")
	}
}

func TestAnalyzeRejectsCommitWithIncompleteStep(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 2, Action: ActionUpdateVolumeMeta}),
		mkRecord(t, RecTxnPrepare, TxnEndPayload{TxnID: 1}),
		mkRecord(t, RecStepDone, StepDonePayload{TxnID: 1, StepID: 1}),
		// step 2 never applied
		mkRecord(t, RecTxnCommit, TxnEndPayload{TxnID: 1}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for TXN_COMMIT with an incomplete step")
	}
}

func TestAnalyzeRejectsCommitWithoutPrepare(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecStepDone, StepDonePayload{TxnID: 1, StepID: 1}),
		mkRecord(t, RecTxnCommit, TxnEndPayload{TxnID: 1}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for TXN_COMMIT without TXN_PREPARE")
	}
}

func TestAnalyzeAbortNeedsNoCompletion(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecTxnAbort, TxnEndPayload{TxnID: 1}),
	})

	a, err := analyzeJournalFile(t, dir)
	if err != nil {
		t.Fatalf("abort of an incomplete txn must be accepted: %v", err)
	}
	if len(a.Pending) != 0 {
		t.Fatalf("aborted txn must not be pending, got %+v", a.Pending)
	}
}

// TestAnalyzeRejectsAbortAfterPrepare verifies that a durable TXN_ABORT for a
// prepared txn is treated as a broken writer. After PREPARE, recovery promises
// to roll the txn forward, so a step may have been applied without its
// STEP_DONE; finishing the txn via abort would let a checkpoint erase the only
// replay plan. Analyze must error to force quarantine.
func TestAnalyzeRejectsAbortAfterPrepare(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecTxnPrepare, TxnEndPayload{TxnID: 1}),
		mkRecord(t, RecTxnAbort, TxnEndPayload{TxnID: 1}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for TXN_ABORT of a prepared txn")
	}
}

// TestAnalyzeRejectsIntentAfterPrepare verifies that an INTENT appended after
// TXN_PREPARE is treated as a broken writer. PREPARE seals the intent set, so
// a crash just before the late intent would replay a different set than a
// crash just after it; Analyze must error to force quarantine.
func TestAnalyzeRejectsIntentAfterPrepare(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecTxnPrepare, TxnEndPayload{TxnID: 1}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 2, Action: ActionCreateHead}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for INTENT after TXN_PREPARE")
	}
}

// TestAnalyzeRejectsIntentAfterFinished verifies that an INTENT appended after
// the txn's terminal record is treated as a broken writer.
func TestAnalyzeRejectsIntentAfterFinished(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecTxnAbort, TxnEndPayload{TxnID: 1}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 2, Action: ActionCreateHead}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for INTENT after the txn finished")
	}
}

// TestAnalyzeRejectsUnknownRecordType verifies that a CRC-valid frame with an
// unrecognized record type is rejected rather than silently ignored, so
// recovery cannot checkpoint away a record whose semantics it never
// understood (a malformed writer or a future format version).
func TestAnalyzeRejectsUnknownRecordType(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecordType(9999), map[string]int{"x": 1}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for an unknown record type")
	}
}

// TestAnalyzeRejectsPrepareForUnknownTxn verifies a TXN_PREPARE with no
// matching TXN_BEGIN is a broken-writer error rather than being ignored.
func TestAnalyzeRejectsPrepareForUnknownTxn(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnPrepare, TxnEndPayload{TxnID: 7}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for TXN_PREPARE of an unknown txn")
	}
}

// TestAnalyzeRejectsPrepareAfterFinished verifies that BEGIN -> ABORT ->
// PREPARE is rejected: a PREPARE after the txn already has a durable
// terminal outcome contradicts the record stream. Left unchecked, a
// following COMMIT would seal a txn the WAL already aborted.
func TestAnalyzeRejectsPrepareAfterFinished(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecTxnAbort, TxnEndPayload{TxnID: 1}),
		mkRecord(t, RecTxnPrepare, TxnEndPayload{TxnID: 1}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for TXN_PREPARE after the txn finished")
	}
}

// TestAnalyzeRejectsDuplicatePrepare verifies a second TXN_PREPARE for the
// same txn is rejected as a broken writer.
func TestAnalyzeRejectsDuplicatePrepare(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecTxnPrepare, TxnEndPayload{TxnID: 1}),
		mkRecord(t, RecTxnPrepare, TxnEndPayload{TxnID: 1}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for a duplicate TXN_PREPARE")
	}
}

// TestAnalyzeRejectsStepDoneAfterFinished verifies a STEP_DONE recorded
// after the txn already has a terminal outcome is a broken-writer error.
func TestAnalyzeRejectsStepDoneAfterFinished(t *testing.T) {
	dir := t.TempDir()
	writeJournalFile(t, dir, []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecTxnAbort, TxnEndPayload{TxnID: 1}),
		mkRecord(t, RecStepDone, StepDonePayload{TxnID: 1, StepID: 1}),
	})

	if _, err := analyzeJournalFile(t, dir); err == nil {
		t.Fatal("expected error for STEP_DONE after the txn finished")
	}
}
