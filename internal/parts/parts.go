package parts

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	LockSchema      = "achta.parts-lock.v1"
	OperationSchema = "achta.parts-operation.v1"
	VerifySchema    = "achta.parts-verify.v1"
	MaxPart         = 1 << 20
	MaxLock         = 1 << 20
	MaxParts        = 1024
	MaxDirEntries   = 4096
)

type Part struct {
	Name string
	Data []byte
}

type Entry struct {
	Name   string
	SHA256 string
}

type Lock struct {
	Version string
	Entries []Entry
}

type Check struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	LockedSHA256 string `json:"locked_sha256,omitempty"`
	ActualSHA256 string `json:"actual_sha256,omitempty"`
}

type VerifyResult struct {
	SchemaVersion string  `json:"schema_version"`
	Status        string  `json:"status"`
	Directory     string  `json:"dir"`
	LockFile      string  `json:"lock"`
	Version       string  `json:"version"`
	Locked        int     `json:"locked"`
	OnDisk        int     `json:"on_disk"`
	Checks        []Check `json:"checks"`
}

func Parse(data []byte) (Lock, error) {
	if len(data) == 0 || len(data) > MaxLock {
		return Lock{}, errors.New("lock is empty or exceeds the size limit")
	}
	if !bytes.HasSuffix(data, []byte("\n")) || bytes.Contains(data, []byte("\r")) {
		return Lock{}, errors.New("lock must use LF lines and end with a newline")
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) < 4 || lines[0] != "# "+LockSchema || !strings.HasPrefix(lines[1], "VERSION ") || lines[2] != "" {
		return Lock{}, fmt.Errorf("lock does not match %s", LockSchema)
	}
	version := strings.TrimPrefix(lines[1], "VERSION ")
	if _, err := versionNumber(version); err != nil {
		return Lock{}, err
	}
	if len(lines)-3 > MaxParts {
		return Lock{}, fmt.Errorf("lock exceeds %d parts", MaxParts)
	}
	lock := Lock{Version: version, Entries: make([]Entry, 0, len(lines)-3)}
	previous := ""
	for _, line := range lines[3:] {
		fields := strings.Split(line, " ")
		if len(fields) != 2 || !ValidName(fields[0]) || !validDigest(fields[1]) {
			return Lock{}, fmt.Errorf("invalid lock entry %q", bounded(line))
		}
		if previous != "" && fields[0] <= previous {
			return Lock{}, errors.New("lock entries must be unique and bytewise filename-sorted")
		}
		lock.Entries = append(lock.Entries, Entry{Name: fields[0], SHA256: fields[1]})
		previous = fields[0]
	}
	if len(lock.Entries) == 0 {
		return Lock{}, errors.New("lock must contain at least one part")
	}
	return lock, nil
}

func Update(current *Lock, bump bool, input []Part) (Lock, []byte, error) {
	parts, err := normalizeParts(input)
	if err != nil {
		return Lock{}, nil, err
	}
	version := "v1"
	if current != nil {
		if _, err := versionNumber(current.Version); err != nil {
			return Lock{}, nil, err
		}
		version = current.Version
	} else if bump {
		return Lock{}, nil, errors.New("--bump requires an existing lock")
	}
	if bump {
		n, _ := versionNumber(version)
		if n == ^uint64(0) {
			return Lock{}, nil, errors.New("lock version cannot be incremented")
		}
		version = "v" + strconv.FormatUint(n+1, 10)
	}
	lock := Lock{Version: version, Entries: make([]Entry, 0, len(parts))}
	for _, part := range parts {
		lock.Entries = append(lock.Entries, Entry{Name: part.Name, SHA256: digest(part.Data)})
	}
	return lock, Render(lock), nil
}

func Render(lock Lock) []byte {
	var output strings.Builder
	output.WriteString("# ")
	output.WriteString(LockSchema)
	output.WriteString("\nVERSION ")
	output.WriteString(lock.Version)
	output.WriteString("\n\n")
	for _, entry := range lock.Entries {
		output.WriteString(entry.Name)
		output.WriteByte(' ')
		output.WriteString(entry.SHA256)
		output.WriteByte('\n')
	}
	return []byte(output.String())
}

// RULE: PARTS-LOCK-1
func Verify(directory, lockFile string, lock Lock, input []Part) (VerifyResult, error) {
	parts, err := normalizeParts(input)
	if err != nil {
		return VerifyResult{}, err
	}
	if directory == "" || lockFile == "" {
		return VerifyResult{}, errors.New("directory and lock paths are required")
	}
	result := VerifyResult{
		SchemaVersion: VerifySchema,
		Status:        "pass",
		Directory:     directory,
		LockFile:      lockFile,
		Version:       lock.Version,
		Locked:        len(lock.Entries),
		OnDisk:        len(parts),
		Checks:        []Check{},
	}
	onDisk := make(map[string]string, len(parts))
	for _, part := range parts {
		onDisk[part.Name] = digest(part.Data)
	}
	locked := make(map[string]bool, len(lock.Entries))
	for _, entry := range lock.Entries {
		locked[entry.Name] = true
		actual, exists := onDisk[entry.Name]
		check := Check{Name: entry.Name, LockedSHA256: entry.SHA256}
		switch {
		case !exists:
			check.Status = "absent"
			result.Status = "fail"
		case actual != entry.SHA256:
			check.Status = "differs"
			check.ActualSHA256 = actual
			result.Status = "fail"
		default:
			check.Status = "match"
			check.ActualSHA256 = actual
		}
		result.Checks = append(result.Checks, check)
	}
	for _, part := range parts {
		if !locked[part.Name] {
			result.Status = "fail"
			result.Checks = append(result.Checks, Check{Name: part.Name, Status: "unlocked", ActualSHA256: onDisk[part.Name]})
		}
	}
	return result, nil
}

func ValidName(name string) bool {
	if len(name) < 5 || len(name) > 128 || name[0] == '.' || !strings.HasSuffix(name, ".txt") {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func normalizeParts(input []Part) ([]Part, error) {
	if len(input) == 0 || len(input) > MaxParts {
		return nil, fmt.Errorf("part set must contain between 1 and %d files", MaxParts)
	}
	parts := append([]Part(nil), input...)
	sort.Slice(parts, func(i, j int) bool { return parts[i].Name < parts[j].Name })
	for i, part := range parts {
		if !ValidName(part.Name) {
			return nil, fmt.Errorf("invalid part name %q", bounded(part.Name))
		}
		if len(part.Data) > MaxPart {
			return nil, fmt.Errorf("part %q exceeds the size limit", part.Name)
		}
		if i > 0 && part.Name == parts[i-1].Name {
			return nil, fmt.Errorf("duplicate part %q", part.Name)
		}
	}
	return parts, nil
}

func versionNumber(version string) (uint64, error) {
	if len(version) < 2 || version[0] != 'v' || version[1] == '0' && len(version) > 2 {
		return 0, fmt.Errorf("invalid lock version %q", bounded(version))
	}
	n, err := strconv.ParseUint(version[1:], 10, 64)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("invalid lock version %q", bounded(version))
	}
	return n, nil
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return false
	}
	return true
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}

func bounded(value string) string {
	if len(value) > 128 {
		return value[:128] + "..."
	}
	return value
}
