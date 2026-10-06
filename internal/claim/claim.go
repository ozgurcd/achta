// Package claim provides advisory, checkout-local ownership. It never mutates
// Git history, the index, or tracked files, and never expires a holder silently.
package claim

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ozgurcd/achta/internal/gitstate"
	"github.com/ozgurcd/achta/internal/safefile"
)

const Schema = "achta.claim.v1"
const StateSchema = "achta.claim-state.v1"
const maxState = 1 << 20

type Holder struct {
	Slice string    `json:"slice"`
	Time  time.Time `json:"time"`
	Note  string    `json:"note"`
}

type Release struct {
	Holder string    `json:"holder"`
	By     string    `json:"by"`
	Time   time.Time `json:"time"`
	Forced bool      `json:"forced"`
	Reason string    `json:"reason"`
}

type State struct {
	SchemaVersion string    `json:"schema_version"`
	Claim         *Holder   `json:"claim"`
	Releases      []Release `json:"releases"`
}

type Result struct {
	SchemaVersion string    `json:"schema_version"`
	Operation     string    `json:"operation"`
	Status        string    `json:"status"`
	Reason        string    `json:"reason,omitempty"`
	Claim         *Holder   `json:"claim"`
	AgeSeconds    int64     `json:"age_seconds"`
	Releases      []Release `json:"releases"`
	*gitstate.Checkout
}

type Options struct {
	Repo, Slice, Note, Reason string
	Force                     bool
	Now                       time.Time
}

// ValidateText rejects bounded secret-shaped and control-bearing inputs without
// echoing them. Path inputs also pass through the CLI's existing filename guard.
func ValidateText(value string, limit int) error {
	if !utf8.ValidString(value) || len(value) > limit || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return errors.New("invalid or oversized claim input")
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "://") || strings.Contains(lower, "-----begin") || strings.Contains(lower, "eyj") {
		return errors.New("refusing secret-like claim input")
	}
	for _, word := range strings.FieldsFunc(lower, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
		switch word {
		case "secret", "secrets", "token", "tokens", "credential", "credentials", "password", "passwords", "passwd", "cookie", "cookies", "totp", "license", "recovery", "bearer":
			return errors.New("refusing secret-like claim input")
		}
	}
	return nil
}

// Run evaluates at the caller's fixed clock. Writes serialize using a nonblocking
// OS lock; read-only operations see complete old or new atomic state snapshots.
func Run(operation string, opts Options) (Result, error) {
	r := Result{SchemaVersion: Schema, Operation: operation, Status: "pass", Releases: []Release{}}
	if opts.Now.IsZero() {
		return r, errors.New("claim clock is required")
	}
	if operation != "take" && operation != "status" && operation != "check" && operation != "release" {
		return r, errors.New("unknown claim operation")
	}
	for _, value := range []string{opts.Repo, opts.Slice, opts.Note, opts.Reason} {
		if err := ValidateText(value, 4096); err != nil {
			return r, err
		}
	}
	if operation != "status" && (strings.TrimSpace(opts.Slice) == "" || len(opts.Slice) > 128) {
		return r, errors.New("--slice is required and limited to 128 bytes")
	}
	if (opts.Note != "" && operation != "take") || ((opts.Force || opts.Reason != "") && operation != "release") || (operation == "status" && opts.Slice != "") {
		return r, errors.New("flag does not apply to this claim operation")
	}
	if opts.Force != (strings.TrimSpace(opts.Reason) != "") {
		return r, errors.New("--force and nonblank --reason must be supplied together")
	}
	dir, err := gitstate.CheckoutGitDir(opts.Repo)
	if err != nil {
		return r, err
	}
	// Canonical OS aliases (such as macOS /var) are allowed, descendant links
	// in the claim files themselves are refused by safefile and O_NOFOLLOW.
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return r, errors.New("claim directory is unavailable")
	}
	if operation == "take" || operation == "release" {
		lock, err := os.OpenFile(filepath.Join(dir, "achta-claim.lock"), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
		if err != nil {
			return r, errors.New("cannot open claim lock")
		}
		defer lock.Close()
		info, err := lock.Stat()
		if err != nil || !info.Mode().IsRegular() {
			return r, errors.New("unsafe claim lock")
		}
		if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			return r, errors.New("claim operation already in progress")
		}
		defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	}
	path := filepath.Join(dir, "achta-claim.json")
	snapshot, err := safefile.Read(dir, path, maxState)
	missing := errors.Is(err, os.ErrNotExist)
	if err != nil && !missing {
		return r, errors.New("cannot read safe claim state")
	}
	state := State{SchemaVersion: StateSchema, Releases: []Release{}}
	if !missing {
		state = State{}
		decoder := json.NewDecoder(bytes.NewReader(snapshot.Data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&state); err != nil {
			return r, errors.New("malformed claim state")
		}
		if decoder.Decode(new(any)) != io.EOF || state.SchemaVersion != StateSchema {
			return r, errors.New("unsupported claim state")
		}
		if state.Claim != nil {
			if state.Claim.Time.IsZero() || state.Claim.Time.After(opts.Now) || strings.TrimSpace(state.Claim.Slice) == "" {
				return r, errors.New("invalid claim holder or clock")
			}
			for _, value := range []string{state.Claim.Slice, state.Claim.Note} {
				if err := ValidateText(value, 4096); err != nil {
					return r, err
				}
			}
		}
		for _, release := range state.Releases {
			if release.Time.IsZero() || release.By == "" || release.Holder == "" || (release.Forced && strings.TrimSpace(release.Reason) == "") {
				return r, errors.New("invalid release history")
			}
			for _, value := range []string{release.Holder, release.By, release.Reason} {
				if err := ValidateText(value, 4096); err != nil {
					return r, err
				}
			}
		}
	}
	r.Claim, r.Releases = state.Claim, state.Releases
	if r.Claim != nil {
		r.AgeSeconds = int64(opts.Now.Sub(r.Claim.Time) / time.Second)
	}
	if operation == "status" || operation == "check" {
		checkout, err := gitstate.InspectCheckout(opts.Repo)
		if err != nil {
			return r, err
		}
		r.Checkout = &checkout
		if operation == "status" {
			return r, nil
		}
		if r.Claim != nil && r.Claim.Slice == opts.Slice {
			return r, nil
		}
		if r.Claim == nil {
			switch {
			case checkout.TrackedChangeCount > 0:
				r.Status, r.Reason = "fail", "checkout has tracked changes"
			case !checkout.UpstreamAvailable:
				r.Status, r.Reason = "fail", "checkout has no resolvable local upstream"
			case *checkout.UnpushedCommitCount > 0:
				r.Status, r.Reason = "fail", "checkout has unpushed commits"
			}
			return r, nil
		}
	}
	if r.Claim != nil && r.Claim.Slice != opts.Slice && !(operation == "release" && opts.Force) {
		r.Status, r.Reason = "fail", fmt.Sprintf("held by %s; age %ds", r.Claim.Slice, r.AgeSeconds)
		return r, nil
	}
	switch operation {
	case "take":
		if r.Claim != nil {
			return r, nil
		}
		state.Claim = &Holder{Slice: opts.Slice, Time: opts.Now.UTC(), Note: opts.Note}
	case "release":
		if r.Claim == nil {
			r.Status, r.Reason = "fail", "no claim to release"
			return r, nil
		}
		state.Releases = append(state.Releases, Release{Holder: r.Claim.Slice, By: opts.Slice, Time: opts.Now.UTC(), Forced: opts.Force, Reason: opts.Reason})
		state.Claim = nil
	default:
		return r, errors.New("invalid claim transition")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return r, errors.New("cannot encode claim state")
	}
	if missing {
		err = safefile.Create(dir, path, data, maxState, 0600)
	} else {
		err = snapshot.Replace(data, maxState)
	}
	if err != nil {
		return r, errors.New("cannot atomically write claim state")
	}
	r.Claim, r.Releases, r.AgeSeconds = state.Claim, state.Releases, 0
	return r, nil
}
