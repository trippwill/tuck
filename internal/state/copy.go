package state

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

func FileChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

func UpsertCopy(record Copy) error {
	_, _, err := mutateRegistry(func(registry Registry) (Registry, bool, error) {
		for i, existing := range registry.Copies {
			if sameCopy(existing, record) {
				registry.Copies[i] = record
				return registry, true, nil
			}
		}
		registry.Copies = append(registry.Copies, record)
		return registry, true, nil
	})
	return err
}

func RemoveCopy(record Copy) error {
	_, _, err := mutateRegistry(func(registry Registry) (Registry, bool, error) {
		copies := registry.Copies[:0]
		changed := false
		for _, existing := range registry.Copies {
			if sameCopy(existing, record) || sameTarget(existing, record) {
				changed = true
				continue
			}
			copies = append(copies, existing)
		}
		registry.Copies = copies
		return registry, changed, nil
	})
	return err
}

func (r Registry) CopyByEntry(source, context, packageName, rel string) (Copy, bool) {
	for _, record := range r.Copies {
		if record.Source == source && record.Context == context && record.Package == packageName && record.Path == rel {
			return record, true
		}
	}
	return Copy{}, false
}

func (r Registry) CopyByTarget(source, context, target string) (Copy, bool) {
	for _, record := range r.Copies {
		if record.Source == source && record.Context == context && record.Target == target {
			return record, true
		}
	}
	return Copy{}, false
}

func sameCopy(a, b Copy) bool {
	return a.Source == b.Source && a.Context == b.Context && a.Package == b.Package && a.Path == b.Path
}

func sameTarget(a, b Copy) bool {
	return b.Target != "" && a.Source == b.Source && a.Context == b.Context && a.Target == b.Target
}
