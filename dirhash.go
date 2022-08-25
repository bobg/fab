package fab

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path/filepath"

	json "github.com/gibson042/canonicaljson-go"
	"github.com/pkg/errors"
)

// NewDirHasher produces a new DirHasher.
// Add files to it with the File method,
// and when finished,
// obtain the hash value with the Hash method.
func NewDirHasher() *DirHasher {
	return &DirHasher{
		hashes: make(map[string][]byte),
	}
}

// File adds the given contents with the given filename to the DirHasher.
func (d *DirHasher) File(name string, r io.Reader) error {
	hasher := sha256.New()
	_, err := io.Copy(hasher, r)
	if err != nil {
		return errors.Wrap(err, "hashing input")
	}
	h := hasher.Sum(nil)
	d.hashes[filepath.Base(name)] = h
	return nil
}

// Hash computes the hash of the files added to the DirHasher.
// The result is insensitive to the order of calls to File.
func (d *DirHasher) Hash() (string, error) {
	j, err := json.Marshal(d.hashes)
	if err != nil {
		return "", errors.Wrap(err, "in JSON encoding")
	}
	hasher := sha256.New()
	h := hasher.Sum(j)
	return hex.EncodeToString(h), nil
}

// DirHasher computes a hash for a set of files.
// Instantiate a DirHasher,
// add files to it by repeated calls to File,
// then get the result by calling Hash.
//
// The zero value of DirHasher is not usable.
// Obtain one with NewDirHasher.
type DirHasher struct {
	hashes map[string][]byte
}
