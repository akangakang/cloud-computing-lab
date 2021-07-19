package journal_entry

import "fmt"

type InvalidJournalEntryError struct {
	entry *MasterEntry
}

func (i *InvalidJournalEntryError) Error() string {
	return fmt.Sprintf("Invalid journal entry: %s\n", i.entry)
}

func NewInvalidJournalEntryError(entry *MasterEntry) *InvalidJournalEntryError {
	return &InvalidJournalEntryError{entry: entry}
}
