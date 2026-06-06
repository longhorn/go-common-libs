package wal

import (
	"encoding/json"
	"testing"
)

func TestAnalyzeEmpty(t *testing.T) {
	a, err := Analyze(nil)
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

func mkRecord(t *testing.T, typ RecordType, v any) Record {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return Record{Type: typ, Payload: b}
}

func TestAnalyzeFinishedAndPending(t *testing.T) {
	recs := []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 1, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecStepDone, StepDonePayload{TxnID: 1, StepID: 1}),
		mkRecord(t, RecTxnCommit, TxnEndPayload{TxnID: 1}),

		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 2, Op: OpSnapRevert}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 2, StepID: 1, Action: ActionCreateHead}),
		mkRecord(t, RecStepDone, StepDonePayload{TxnID: 2, StepID: 1}),
		mkRecord(t, RecIntent, IntentPayload{TxnID: 2, StepID: 2, Action: ActionUpdateVolumeMeta}),
		// crash before STEP_DONE for step 2

		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 3, Op: OpSnapRemoveMark}),
		// crash before any intent
	}

	a, err := Analyze(recs)
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
	recs := []Record{
		mkRecord(t, RecIntent, IntentPayload{TxnID: 99, StepID: 1, Action: ActionCreateHead}),
	}
	if _, err := Analyze(recs); err == nil {
		t.Fatal("expected error for INTENT without TXN_BEGIN")
	}
}

func TestAnalyzeRejectsDuplicateBegin(t *testing.T) {
	recs := []Record{
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapCreate}),
		mkRecord(t, RecTxnBegin, TxnBeginPayload{TxnID: 1, Op: OpSnapRevert}),
	}
	if _, err := Analyze(recs); err == nil {
		t.Fatal("expected error for duplicate TXN_BEGIN")
	}
}
