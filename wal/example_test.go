package wal_test

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/longhorn/go-common-libs/wal"
)

// Example_writeTransaction shows the normal write path: open, recover
// once, then record a transaction as intents -> prepare -> apply+done ->
// commit. The apply of each step is the caller's own idempotent work and
// is omitted here.
func Example_writeTransaction() {
	dir, err := os.MkdirTemp("", "wal-example")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	j, err := wal.Open(dir)
	if err != nil {
		log.Fatal(err)
	}

	// Recover once after Open and before any Begin so freshly-issued
	// TxnIDs cannot collide with transactions already durable on disk.
	if _, err := j.Recover(); err != nil {
		log.Fatal(err)
	}

	params, _ := json.Marshal(map[string]string{"snapshot": "snap-001"})
	tx, err := j.Begin(wal.OpSnapCreate, params)
	if err != nil {
		log.Fatal(err)
	}

	// 1) Write every step's intent first (nothing applied yet).
	if err := tx.Intent(1, wal.ActionCreateHead, nil); err != nil {
		log.Fatal(err)
	}
	if err := tx.Intent(2, wal.ActionUpdateVolumeMeta, nil); err != nil {
		log.Fatal(err)
	}

	// 2) Seal the intent set. Only after Prepare is the transaction
	//    eligible to be redone by recovery.
	if err := tx.Prepare(); err != nil {
		log.Fatal(err)
	}

	// 3) Apply each step (caller's own idempotent work), then mark it done.
	//    applyCreateHead() ...
	if err := tx.StepDone(1); err != nil {
		log.Fatal(err)
	}
	//    applyUpdateVolumeMeta() ...
	if err := tx.StepDone(2); err != nil {
		log.Fatal(err)
	}

	// Inspect the on-disk records before committing (a clean Close below
	// checkpoints and truncates the file to zero).
	recs, err := j.Scan()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("records before commit: %d\n", len(recs))

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("transaction committed")

	if err := j.Close(); err != nil {
		log.Fatal(err)
	}

	// Output:
	// records before commit: 6
	// transaction committed
}

// Example_recoverAfterCrash shows the recovery path: a process crashes
// after Prepare and one applied step; the next process reopens, recovers
// the pending transaction, replays only the steps that were not yet
// durably done, and commits.
func Example_recoverAfterCrash() {
	dir, err := os.MkdirTemp("", "wal-example")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	// --- First process: prepare, apply step 1, then crash (SIGKILL). ---
	j1, err := wal.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	tx, err := j1.Begin(wal.OpSnapCreate, nil)
	if err != nil {
		log.Fatal(err)
	}
	_ = tx.Intent(1, wal.ActionCreateHead, nil)
	_ = tx.Intent(2, wal.ActionUpdateVolumeMeta, nil)
	_ = tx.Prepare()
	_ = tx.StepDone(1) // step 1 applied + durable; step 2 not yet
	// ForceCloseForTest drops the fd + flock without a clean checkpoint,
	// simulating an abrupt process death.
	_ = wal.ForceCloseForTest(j1)

	// --- Second process: reopen and recover. ---
	j2, err := wal.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = j2.Close() }()

	analysis, err := j2.Recover()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("pending txns: %d\n", len(analysis.Pending))

	for _, pt := range analysis.Pending {
		adopted, err := wal.AdoptTxn(j2, pt.ID, pt.Op)
		if err != nil {
			log.Fatal(err)
		}

		if !pt.Prepared {
			// Intent set may be torn; abort instead of replaying.
			_ = adopted.Abort()
			fmt.Printf("txn %d aborted (not prepared)\n", pt.ID)
			continue
		}

		for _, in := range pt.PendingIntents {
			if pt.CompletedSteps[in.StepID] {
				continue // already durably applied before the crash
			}
			// replay(in) idempotently ...
			if err := adopted.StepDone(in.StepID); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("txn %d replayed step %d (%s)\n", pt.ID, in.StepID, in.Action)
		}
		if err := adopted.Commit(); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("txn %d committed\n", pt.ID)
	}

	// Output:
	// pending txns: 1
	// txn 1 replayed step 2 (UPDATE_VOLUME_META)
	// txn 1 committed
}
