package fake

// Stringer is a simple helper that implements fmt.Stringer for tests.
// It prefixes the stored value with "S:" to make assertions explicit.
type Stringer struct{ V string }

func (t Stringer) String() string { return "S:" + t.V }
