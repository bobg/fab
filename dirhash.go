package fab

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path/filepath"

	json "github.com/gibson042/canonicaljson-go"
	"github.com/pkg/errors"
)

func NewDirHasher() *DirHasher {
	return &DirHasher{
		hashes: make(map[string][]byte),
	}
}

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

func (d *DirHasher) Hash() (string, error) {
	j, err := json.Marshal(d.hashes)
	if err != nil {
		return "", errors.Wrap(err, "in JSON encoding")
	}
	hasher := sha256.New()
	h := hasher.Sum(j)
	return hex.EncodeToString(h), nil
}

type DirHasher struct {
	hashes map[string][]byte
}
