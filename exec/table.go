package exec

// Table is a WASM table.
type Table struct {
	min, max uint32
	entries  []Function
}

// NewTable creates a new WASM table.
func NewTable(min, max uint32) Table {
	t := Table{min: min, max: max, entries: make([]Function, min)}
	for i := range t.entries {
		t.entries[i] = UninitializedFunction
	}
	return t
}

// Limits returns the minimum and maximum size of the table in elements.
func (t *Table) Limits() (min uint32, max uint32) {
	return t.min, t.max
}

// Entries returns the table's entries.
func (t *Table) Entries() []Function {
	return t.entries
}
