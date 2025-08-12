package multierr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiError(t *testing.T) {
	t.Run("NewMultiError creates empty map", func(t *testing.T) {
		me := NewMultiError()
		assert.NotNil(t, me)
		assert.Equal(t, 0, len(me))
	})

	t.Run("Append adds errors under reason", func(t *testing.T) {
		me := NewMultiError()
		me.Append("reason1", errors.New("err1"))
		me.Append("reason1", errors.New("err2"))
		me.Append("reason2", errors.New("err3"))

		assert.Len(t, me["reason1"], 2)
		assert.Len(t, me["reason2"], 1)
	})

	t.Run("Append ignores empty reason or nil error", func(t *testing.T) {
		me := NewMultiError()
		me.Append("", errors.New("err1"))
		me.Append("reason", nil)
		assert.Equal(t, 0, len(me))
	})

	t.Run("AppendMultiError merges errors correctly", func(t *testing.T) {
		me1 := NewMultiError()
		me1.Append("r1", errors.New("e1"))

		me2 := NewMultiError()
		me2.Append("r1", errors.New("e2"))
		me2.Append("r2", errors.New("e3"))

		me1.AppendMultiError(me2)

		assert.Len(t, me1["r1"], 2)
		assert.Len(t, me1["r2"], 1)
	})

	t.Run("Reset clears all errors", func(t *testing.T) {
		me := NewMultiError()
		me.Append("r1", errors.New("e1"))
		me.Reset()
		assert.Equal(t, 0, len(me))
	})

	t.Run("JoinReasons returns sorted reasons", func(t *testing.T) {
		me := NewMultiError()
		me.Append("b_reason", errors.New("e1"))
		me.Append("a_reason", errors.New("e2"))
		assert.Equal(t, "a_reason; b_reason", me.JoinReasons())
		assert.Equal(t, "", NewMultiError().JoinReasons())
	})

	t.Run("Error formats as 'reason: err1; err2; ...'", func(t *testing.T) {
		me := NewMultiError()
		me.Append("r1", errors.New("e1"))
		me.Append("r1", errors.New("e2"))
		me.Append("r2", errors.New("e3"))

		got := me.Error()
		// The order is non-deterministic due to map iteration order,
		// so allow both possible expected strings.
		expected1 := "r1: e1; e2; r2: e3"
		expected2 := "r2: e3; r1: e1; e2"

		assert.True(t, got == expected1 || got == expected2,
			"got %q, expected %q or %q", got, expected1, expected2)
	})

	t.Run("Error returns empty string for empty map", func(t *testing.T) {
		assert.Equal(t, "", NewMultiError().Error())
	})

	t.Run("ErrorByReason returns errors only for the specified reason", func(t *testing.T) {
		me := NewMultiError()
		me.Append("r1", errors.New("e1"))
		me.Append("r1", errors.New("e2"))
		me.Append("r2", errors.New("e3"))

		gotR1 := me.ErrorByReason("r1")
		expectedR1 := "e1; e2"
		assert.Equal(t, expectedR1, gotR1)

		gotR2 := me.ErrorByReason("r2")
		expectedR2 := "e3"
		assert.Equal(t, expectedR2, gotR2)

		gotR3 := me.ErrorByReason("r3") // non-existent reason
		assert.Equal(t, "", gotR3)
	})
}
