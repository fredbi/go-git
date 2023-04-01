package object

import (
	"sync"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	"github.com/go-git/go-git/v5/utils/merkletrie/noder"
)

// FRED DEBUG
var (
	cachedTrees   *treeCache
	onceTreeCache sync.Once

	cachedTreeNoders   *treeNoderCache
	onceTreeNoderCache sync.Once

	treeWalkers = sync.Pool{
		New: func() interface{} {
			return new(TreeWalker)
		},
	}
	cachedMerkles   *merkleCache
	onceMerkleCache sync.Once
)

type (
	treeNoderCache struct {
		sync.Map
	}

	treeCache struct {
		sync.Map
	}
	merkleCache struct {
		sync.Map
	}

	treeNoderKey struct {
		plumbing.Hash
		filemode.FileMode
	}
)

func onceInitTreeCache() {
	onceTreeCache.Do(func() {
		cachedTrees = &treeCache{}
	})
}

func onceInitTreeNoderCache() {
	onceTreeNoderCache.Do(func() {
		cachedTreeNoders = &treeNoderCache{}
	})
}

func onceInitMerkleCache() {
	onceMerkleCache.Do(func() {
		cachedMerkles = &merkleCache{}
	})
}

func (c *treeCache) Get(h plumbing.Hash) *Tree {
	val, ok := c.Load(h)
	if !ok {
		return nil
	}

	return val.(*Tree)
}

func (c *treeCache) Put(h plumbing.Hash, t *Tree) {
	c.Store(h, t)
}

func (c *treeNoderCache) Get(h plumbing.Hash, m filemode.FileMode) *treeNoder {
	val, ok := c.Load(treeNoderKey{Hash: h, FileMode: m})
	if !ok {
		return nil
	}

	return val.(*treeNoder)
}

func (c *treeNoderCache) Put(h plumbing.Hash, m filemode.FileMode, t *treeNoder) {
	c.Store(treeNoderKey{Hash: h, FileMode: m}, t)
}

func (c *merkleCache) Get(from, to noder.Noder) merkletrie.Changes {
	key := merkleKey(from, to)

	val, ok := c.Load(key)
	if !ok {
		return nil
	}

	return val.(merkletrie.Changes)
}

func (c *merkleCache) Put(from, to noder.Noder, changes merkletrie.Changes) {
	key := merkleKey(from, to)

	c.Store(key, changes)
}

func merkleKey(from, to noder.Noder) [48]byte {
	fh := from.Hash()
	th := to.Hash()

	var res [48]byte
	copy(res[:], fh[:])
	copy(res[24:], th[:])

	return res
}

func getTreeWalker(t *Tree, recursive bool, seen map[plumbing.Hash]bool) *TreeWalker {
	w := treeWalkers.Get().(*TreeWalker)

	if w.stack == nil {
		w.stack = make([]*treeEntryIter, 0, startingStackSize)
	} else {
		w.stack = w.stack[:0]
	}

	w.stack = append(w.stack, &treeEntryIter{t, 0})
	w.recursive = recursive
	w.seen = seen
	w.s = t.s
	w.t = t

	return w
}

func putTreeWalker(w *TreeWalker) {
	treeWalkers.Put(w)
}
