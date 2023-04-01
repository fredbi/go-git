package fsnoder

import (
	"fmt"
	"hash/fnv"
	"strings"
	"unicode/utf8"

	"github.com/go-git/go-git/v5/utils/merkletrie/noder"
)

// file values represent file-like noders in a merkle trie.
type file struct {
	name     string // relative
	contents []byte
	hash     [24]byte // memoized
}

// newFile returns a noder representing a file with the given contents.
func newFile(name, contents string) (*file, error) {
	return newFileBytes([]byte(name), []byte(contents))
}

func newFileBytes(name, contents []byte) (*file, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("files cannot have empty names")
	}
	h := fnv.New64a()
	h.Write(contents) // it nevers returns an error.
	var hash [24]byte
	copy(hash[:], h.Sum(nil))

	return &file{
		name:     string(name),
		contents: contents,
		hash:     hash,
	}, nil
}

// The hash of a file is just its contents.
// Empty files will have the fnv64 basis offset as its hash.
func (f *file) Hash() [24]byte {
	return f.hash
}

func (f *file) Name() string {
	return f.name
}

func (f *file) IsDir() bool {
	return false
}

func (f *file) Children() ([]noder.Noder, error) {
	return noder.NoChildren, nil
}

func (f *file) NumChildren() (int, error) {
	return 0, nil
}

func (f *file) Skip() bool {
	return false
}

const (
	fileStartMark = '<'
	fileEndMark   = '>'
)

// String returns a string formatted as: name<contents>.
func (f *file) String() string {
	var buf strings.Builder
	buf.Grow(len(f.name) + utf8.RuneLen(fileStartMark) + len(f.contents) + utf8.RuneLen(fileEndMark))
	buf.WriteString(f.name)
	buf.WriteRune(fileStartMark)
	buf.Write(f.contents)
	buf.WriteRune(fileEndMark)

	return buf.String()
}
