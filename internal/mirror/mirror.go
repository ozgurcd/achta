// Package mirror compares one master document with explicitly named mirrors.
package mirror

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
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

// DigestInput is one recorded digest file to compare with the master.
type DigestInput struct {
	Path string
	Name string
	Data []byte
}

// MirrorResult records the byte-comparison result for one mirror.
type MirrorResult struct {
	Path         string `json:"path"`
	Status       string `json:"status"`
	MasterSHA256 string `json:"master_sha256"`
	MirrorSHA256 string `json:"mirror_sha256,omitempty"`
}

// DigestResult identifies the exact recorded digest selected for the master.
type DigestResult struct {
	File           string `json:"file"`
	Line           int    `json:"line"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	RecordedSHA256 string `json:"recorded_sha256"`
	MasterSHA256   string `json:"master_sha256"`
}

// Result is the stable mirror-check result.
type Result struct {
	SchemaVersion string         `json:"schema_version"`
	Status        string         `json:"status"`
	Master        string         `json:"master"`
	MasterSHA256  string         `json:"master_sha256"`
	Mirrors       []MirrorResult `json:"mirrors"`
	Digest        *DigestResult  `json:"digest,omitempty"`
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

// CheckWithDigest checks zero or more mirrors and one recorded digest against
// the exact master bytes. The digest file must contain exactly one applicable
// canonical record selected by an exact master-basename match.
func CheckWithDigest(masterPath string, master []byte, inputs []Input, input DigestInput) (Result, error) {
	var result Result
	if len(inputs) > 0 {
		checked, err := Check(masterPath, master, inputs)
		if err != nil {
			return checked, err
		}
		result = checked
	} else {
		result = Result{
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
		result.MasterSHA256 = digest(master)
	}
	if input.Path == "" {
		return result, errors.New("digest path is required")
	}
	if len(input.Data) > MaxFile {
		return result, errors.New("digest file exceeds the size limit")
	}

	record, err := selectDigestRecord(masterPath, input.Name, input.Data)
	if err != nil {
		return result, err
	}
	item := &DigestResult{
		File:           input.Path,
		Line:           record.line,
		Name:           record.name,
		Status:         "match",
		RecordedSHA256: record.sha256,
		MasterSHA256:   result.MasterSHA256,
	}
	if item.RecordedSHA256 != item.MasterSHA256 {
		item.Status = "differs"
		result.Status = "fail"
	}
	result.Digest = item
	return result, nil
}

type digestRecord struct {
	line   int
	name   string
	sha256 string
}

func selectDigestRecord(masterPath, selectedName string, data []byte) (digestRecord, error) {
	lines := bytes.Split(data, []byte{'\n'})
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return digestRecord{}, errors.New("digest file has no records")
	}

	wantName := selectedName
	if wantName == "" {
		wantName = filepath.Base(masterPath)
	}
	if len(wantName) > 4096 {
		return digestRecord{}, errors.New("digest record name exceeds 4096 bytes")
	}
	if bytes.ContainsAny([]byte(wantName), "\r\n") {
		return digestRecord{}, errors.New("digest record name contains a line ending")
	}
	var selected digestRecord
	matches := 0
	for index, line := range lines {
		lineNumber := index + 1
		if len(line) < 67 || line[64] != ' ' || line[65] != ' ' || !lowerHex(line[:64]) || len(line[66:]) == 0 {
			return digestRecord{}, fmt.Errorf("digest record line %d is not canonical lowercase sha256<two-spaces>name", lineNumber)
		}
		name := string(line[66:])
		if name != wantName {
			continue
		}
		matches++
		selected = digestRecord{line: lineNumber, name: name, sha256: string(line[:64])}
	}
	if matches == 0 {
		return digestRecord{}, fmt.Errorf("digest file has no record named %q", wantName)
	}
	if matches > 1 {
		return digestRecord{}, fmt.Errorf("digest file has %d records named %q", matches, wantName)
	}
	return selected, nil
}

func lowerHex(value []byte) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func digest(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
