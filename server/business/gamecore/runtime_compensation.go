package gamecore

import "fmt"

// RemoveForCompensation removes a live instance that was registered while a
// surrounding transaction was still pending. Callers must not expose or apply
// commands to the instance before the transaction commits.
func (d *LiveDirectory) RemoveForCompensation(id InstanceID) error {
	entry, err := d.entry(id)
	if err != nil {
		return err
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.closed || entry.pending != nil {
		return fmt.Errorf("%w: instance %s cannot be compensated", ErrFinalizationPending, id)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if current := d.entries[id]; current != entry {
		return fmt.Errorf("%w: %s", ErrInstanceNotFound, id)
	}
	delete(d.entries, id)
	entry.closed = true
	return nil
}
