package filesystem

import (
	"io"
	"os"
	"path"
	"unsafe"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/utils/merkletrie/noder"

	"github.com/go-git/go-billy/v5"
)

var zeroHash = [24]byte{}

func ignore(key string) bool {
	if key == ".git" {
		return true
	}

	return false
}

// The node represents a file or a directory in a billy.Filesystem. It
// implements the interface noder.Noder of merkletrie package.
//
// This implementation implements a "standard" hash method being able to be
// compared with any other noder.Noder implementation inside of go-git.
type node struct {
	fs         billy.Filesystem
	submodules map[string]plumbing.Hash

	path       string
	hash       [24]byte
	children   []noder.Noder
	isDir      bool
	childrenOK bool
}

// NewRootNode returns the root node based on a given billy.Filesystem.
//
// In order to provide the submodule hash status, a map[string]plumbing.Hash
// should be provided where the key is the path of the submodule and the commit
// of the submodule HEAD
func NewRootNode(
	fs billy.Filesystem,
	submodules map[string]plumbing.Hash,
) noder.Noder {
	return &node{
		fs:         fs,
		submodules: submodules,
		isDir:      true,
	}
}

// Hash the hash of a filesystem is the result of concatenating the computed
// plumbing.Hash of the file as a Blob and its plumbing.FileMode; that way the
// difftree algorithm will detect changes in the contents of files and also in
// their mode.
//
// The hash of a directory is always a 24-bytes slice of zero values
func (n *node) Hash() [24]byte {
	return n.hash
}

func (n *node) Name() string {
	return path.Base(n.path)
}

func (n *node) IsDir() bool {
	return n.isDir
}

func (n *node) Skip() bool {
	return false
}

func (n *node) Children() ([]noder.Noder, error) {
	if err := n.calculateChildren(); err != nil {
		return nil, err
	}

	return n.children, nil
}

func (n *node) NumChildren() (int, error) {
	if !n.childrenOK {
		if err := n.calculateChildren(); err != nil {
			return -1, err
		}
	}

	return len(n.children), nil
}

func (n *node) calculateChildren() error {
	if !n.IsDir() {
		return nil
	}

	if len(n.children) != 0 {
		return nil
	}

	files, err := n.fs.ReadDir(n.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	n.children = make([]noder.Noder, 0, len(files))
	for _, file := range files {
		if ignore(file.Name()) {
			continue
		}

		c, err := n.newChildNode(file)
		if err != nil {
			return err
		}

		n.children = append(n.children, c)
	}

	n.childrenOK = true

	return nil
}

func (n *node) newChildNode(file os.FileInfo) (*node, error) {
	path := path.Join(n.path, file.Name())

	hash, err := n.calculateHash(path, file)
	if err != nil {
		return nil, err
	}

	node := &node{
		fs:         n.fs,
		submodules: n.submodules,

		path:  path,
		hash:  hash,
		isDir: file.IsDir(),
	}

	if hash, isSubmodule := n.submodules[path]; isSubmodule {
		var h [24]byte
		copy(h[:], hash[:])
		copy(h[20:], filemode.Submodule.Bytes())
		node.hash = h
		node.isDir = false
	}

	return node, nil
}

func (n *node) calculateHash(path string, file os.FileInfo) ([24]byte, error) {
	if file.IsDir() {
		return zeroHash, nil
	}

	var hash plumbing.Hash
	var err error
	if file.Mode()&os.ModeSymlink != 0 {
		hash, err = n.doCalculateHashForSymlink(path, file)
	} else {
		hash, err = n.doCalculateHashForRegular(path, file)
	}

	if err != nil {
		return zeroHash, err
	}

	mode, err := filemode.NewFromOSFileMode(file.Mode())
	if err != nil {
		return zeroHash, err
	}

	var h [24]byte
	copy(h[:], hash[:])
	copy(h[20:], mode.Bytes())

	return h, nil
}

func (n *node) doCalculateHashForRegular(path string, file os.FileInfo) (plumbing.Hash, error) {
	f, err := n.fs.Open(path)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	defer f.Close()

	h := plumbing.NewHasher(plumbing.BlobObject, file.Size())
	//var buf [4096]byte
	//if _, err := io.CopyBuffer(h, f, buf[:]); err != nil {
	if _, err := io.Copy(h, f); err != nil {
		return plumbing.ZeroHash, err
	}

	return h.Sum(), nil
}

func (n *node) doCalculateHashForSymlink(path string, file os.FileInfo) (plumbing.Hash, error) {
	target, err := n.fs.Readlink(path)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	h := plumbing.NewHasher(plumbing.BlobObject, file.Size())
	if _, err := h.Write(hackZeroAlloc(target)); err != nil {
		return plumbing.ZeroHash, err
	}

	return h.Sum(), nil
}

func (n *node) String() string {
	return n.path
}

// internalString representation of a string by the golang runtime
type internalString struct {
	Data unsafe.Pointer
	Len  int
}

// hackZeroAlloc reuses a common hack found in the standard library
// to avoid allocating the underlying bytes of a string when converting.
//
// This assumes that the caller does not use the returned []byte slices after
// having relinquished the input string to the garbage collector.
func hackZeroAlloc(s string) []byte {
	addr := (*internalString)(unsafe.Pointer(&s)).Data

	return unsafe.Slice((*byte)(addr), len(s))
}
