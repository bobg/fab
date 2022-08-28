package fab

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path/filepath"

	json "github.com/gibson042/canonicaljson-go"
	"github.com/pkg/errors"
)

// newDirHasher produces a new dirHasher.
// Add files to it with the File method,
// and when finished,
// obtain the hash value with the Hash method.
func newDirHasher() *dirHasher {
	return &dirHasher{
		hashes: make(map[string][]byte),
	}
}

// File adds the given contents with the given filename to the dirHasher.
func (d *dirHasher) file(name string, r io.Reader) error {
	hasher := sha256.New()
	_, err := io.Copy(hasher, r)
	if err != nil {
		return errors.Wrap(err, "hashing input")
	}
	h := hasher.Sum(nil)
	d.hashes[filepath.Base(name)] = h
	return nil
}

// Hash computes the hash of the files added to the dirHasher.
// The result is insensitive to the order of calls to File.
func (d *dirHasher) hash() (string, error) {
	j, err := json.Marshal(d.hashes)
	if err != nil {
		return "", errors.Wrap(err, "in JSON encoding")
	}
	hasher := sha256.New()
	h := hasher.Sum(j)
	return hex.EncodeToString(h), nil
}

// dirHasher computes a hash for a set of files.
// Instantiate a dirHasher,
// add files to it by repeated calls to File,
// then get the result by calling Hash.
//
// The zero value of dirHasher is not usable.
// Obtain one with newDirHasher.
type dirHasher struct {
	hashes map[string][]byte
}
