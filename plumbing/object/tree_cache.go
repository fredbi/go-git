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
	cachedStrings   *stringCache
	onceStringCache sync.Once
)

type (
	treeNoderCache struct {
		noders map[treeNoderKey]*treeNoder
	}

	treeCache struct {
		trees map[plumbing.Hash]*Tree
	}
	merkleCache struct {
		changes map[[48]byte]merkletrie.Changes
	}

	treeNoderKey struct {
		plumbing.Hash
		filemode.FileMode
	}
	stringCache struct { // memoized strings
		strings map[string]string
	}
)

func onceInitTreeCache() {
	onceTreeCache.Do(func() {
		cachedTrees = &treeCache{
			trees: make(map[plumbing.Hash]*Tree, 1000),
		}
	})
}

func onceInitTreeNoderCache() {
	onceTreeNoderCache.Do(func() {
		cachedTreeNoders = &treeNoderCache{
			noders: make(map[treeNoderKey]*treeNoder, 1000),
		}
	})
}

func onceInitMerkleCache() {
	onceMerkleCache.Do(func() {
		cachedMerkles = &merkleCache{
			changes: make(map[[48]byte]merkletrie.Changes, 1000),
		}
	})
}

func onceInitStringCache() {
	onceStringCache.Do(func() {
		cachedStrings = &stringCache{strings: make(map[string]string, 1000)}
	})
}

func (c *treeCache) Get(h plumbing.Hash) *Tree {
	val, ok := c.trees[h]
	if !ok {
		return nil
	}

	return val
}

func (c *treeCache) Put(h plumbing.Hash, t *Tree) {
	c.trees[h] = t
}

func (c *treeNoderCache) Get(h plumbing.Hash, m filemode.FileMode) *treeNoder {
	val, ok := c.noders[treeNoderKey{Hash: h, FileMode: m}]
	if !ok {
		return nil
	}

	return val
}

func (c *treeNoderCache) Put(h plumbing.Hash, m filemode.FileMode, t *treeNoder) {
	c.noders[treeNoderKey{Hash: h, FileMode: m}] = t
}

func (c *merkleCache) Get(from, to noder.Noder) merkletrie.Changes {
	key := merkleKey(from, to)

	val, ok := c.changes[key]
	if !ok {
		return nil
	}

	return val
}

func (c *merkleCache) Put(from, to noder.Noder, changes merkletrie.Changes) {
	key := merkleKey(from, to)

	c.changes[key] = changes
}

func merkleKey(from, to noder.Noder) [48]byte {
	fh := from.Hash()
	th := to.Hash()

	var res [48]byte
	copy(res[:], fh[:])
	copy(res[24:], th[:])

	return res
}

func (c *stringCache) Get(key []byte) string {
	if val, ok := c.strings[string(key)]; ok {
		return val
	}

	s := string(key)
	c.strings[s] = s

	return s
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
