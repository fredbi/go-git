package object

import (
	"io"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/utils/merkletrie/noder"
)

// A treenoder is a helper type that wraps git trees into merkletrie
// noders.
//
// As a merkletrie noder doesn't understand the concept of modes (e.g.
// file permissions), the treenoder includes the mode of the git tree in
// the hash, so changes in the modes will be detected as modifications
// to the file contents by the merkletrie difftree algorithm.  This is
// consistent with how the "git diff-tree" command works.
type treeNoder struct {
	parent   *Tree  // the root node is its own parent
	name     string // empty string for the root node
	mode     filemode.FileMode
	nhash    plumbing.Hash // git hash of the current node
	hash     [24]byte      // noder hash, appended with mode
	children []noder.Noder // memoized
	walker   *TreeWalker
}

// NewTreeRootNode returns the root node of a Tree
func NewTreeRootNode(tree *Tree) noder.Noder {
	if tree == nil {
		return &treeNoder{}
	}

	if tree.treeOptions != nil && tree.caches != nil && tree.caches.treeNoders != nil {
		if n := tree.caches.treeNoders.Get(tree.Hash, filemode.Dir); n != nil {
			return n
		}
	}

	n := &treeNoder{
		parent: tree,
		name:   "",
		mode:   filemode.Dir,
		nhash:  tree.Hash,
		hash:   makeNoderHash(tree.Hash, filemode.Dir),
	}

	if tree.treeOptions != nil && tree.caches != nil && tree.caches.treeNoders != nil {
		tree.caches.treeNoders.Put(tree.Hash, filemode.Dir, n)
	}

	return n
}

func makeNoderHash(hash plumbing.Hash, mode filemode.FileMode) [24]byte {
	var h [24]byte
	copy(h[:], hash[:])

	if mode == filemode.Deprecated {
		copy(h[20:], filemode.Regular.Bytes())

		return h
	}

	copy(h[20:], mode.Bytes())

	return h
}

func (t *treeNoder) Skip() bool {
	return false
}

func (t *treeNoder) isRoot() bool {
	return t.name == ""
}

func (t *treeNoder) String() string {
	return "treeNoder <" + t.name + ">"
}

func (t *treeNoder) Hash() [24]byte {
	return t.hash
}

func (t *treeNoder) Name() string {
	return t.name
}

func (t *treeNoder) IsDir() bool {
	return t.mode == filemode.Dir
}

// Children will return the children of a treenoder as treenoders,
// building them from the children of the wrapped git tree.
func (t *treeNoder) Children() ([]noder.Noder, error) {
	if t.mode != filemode.Dir {
		return noder.NoChildren, nil
	}

	// children are memoized for efficiency
	if t.children != nil {
		return t.children, nil
	}

	// the parent of the returned children will be ourself as a tree if
	// we are a not the root treenoder.  The root is special as it
	// is is own parent.
	parent := t.parent
	if !t.isRoot() {
		var err error
		if parent, err = t.parent.Tree(t.name); err != nil {
			return nil, err
		}
	}

	return t.transformChildren(parent)
}

// Returns the children of a tree as treenoders.
// Efficiency is key here.
func (t *treeNoder) transformChildren(tree *Tree) ([]noder.Noder, error) {
	var err error
	var e TreeEntry

	// there will be more tree entries than children in the tree,
	// due to submodules and empty directories, but I think it is still
	// worth it to pre-allocate the whole array now, even if sometimes
	// is bigger than needed.
	var ret []noder.Noder // it is actually better to leave the go runtime allocate as needed rather than to preallocate.

	if t.walker == nil {
		t.walker = newTreeWalker(tree, false, nil, false) // don't recurse
	} else {
		t.walker.reset()
	}
	// don't defer walker.Close() for efficiency reasons.
	for {
		_, e, err = t.walker.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		var n *treeNoder
		if tree.caches != nil && tree.caches.treeNoders != nil {
			n = tree.caches.treeNoders.Get(e.Hash, e.Mode)
		}
		if n == nil {
			n = &treeNoder{
				parent: tree,
				name:   e.Name,
				mode:   e.Mode,
				nhash:  e.Hash,
				hash:   makeNoderHash(e.Hash, e.Mode),
			}
			if tree.caches != nil && tree.caches.treeNoders != nil {
				tree.caches.treeNoders.Put(e.Hash, e.Mode, n)
			}
		}

		ret = append(ret, n)
	}

	return ret, nil
}

// len(t.tree.Entries) != the number of elements walked by treewalker
// for some reason because of empty directories, submodules, etc, so we
// have to walk here.
func (t *treeNoder) NumChildren() (int, error) {
	children, err := t.Children()
	if err != nil {
		return 0, err
	}

	return len(children), nil
}
