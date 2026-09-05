// Package mirror compares one master document with explicitly named mirrors.
package mirror

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

const (
	// Schema is the stable mirror-check machine interface.
	Schema = "achta.mirror-check.v1"
	// MaxFile bounds every document read by mirror check.
	MaxFile = 16 << 20
)

// Input is one already confined and bounded mirror document.
type Input struct {
	Path   string
	Data   []byte
	Absent bool
}

// MirrorResult records the byte-comparison result for one mirror.
type MirrorResult struct {
	Path         string `json:"path"`
	Status       string `json:"status"`
	MasterSHA256 string `json:"master_sha256"`
	MirrorSHA256 string `json:"mirror_sha256,omitempty"`
}

// Result is the stable mirror-check result.
type Result struct {
	SchemaVersion string         `json:"schema_version"`
	Status        string         `json:"status"`
	Master        string         `json:"master"`
	MasterSHA256  string         `json:"master_sha256"`
	Mirrors       []MirrorResult `json:"mirrors"`
}

// Check computes SHA-256 over the exact input bytes and compares each mirror
// in caller-supplied order. An absent mirror is an evaluated mismatch.
func Check(masterPath string, master []byte, inputs []Input) (Result, error) {
	result := Result{
		SchemaVersion: Schema,
		Status:        "pass",
		Master:        masterPath,
		Mirrors:       []MirrorResult{},
	}
	if masterPath == "" {
		return result, errors.New("master path is required")
	}
	if len(master) > MaxFile {
		return result, errors.New("master exceeds the size limit")
	}
	if len(inputs) == 0 {
		return result, errors.New("at least one mirror is required")
	}

	masterDigest := digest(master)
	result.MasterSHA256 = masterDigest
	for _, input := range inputs {
		if input.Path == "" {
			return result, errors.New("mirror path is required")
		}
		item := MirrorResult{Path: input.Path, MasterSHA256: masterDigest}
		if input.Absent {
			item.Status = "absent"
			result.Status = "fail"
			result.Mirrors = append(result.Mirrors, item)
			continue
		}
		if len(input.Data) > MaxFile {
			return result, fmt.Errorf("mirror %q exceeds the size limit", input.Path)
		}
		item.MirrorSHA256 = digest(input.Data)
		if item.MirrorSHA256 == masterDigest {
			item.Status = "match"
		} else {
			item.Status = "differs"
			result.Status = "fail"
		}
		result.Mirrors = append(result.Mirrors, item)
	}
	return result, nil
}

func digest(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
