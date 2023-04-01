package merkletrie

import (
	"sync"

	"github.com/go-git/go-git/v5/utils/merkletrie/noder"
)

var noderPaths = sync.Pool{
	New: func() interface{} {
		noders := make(noder.Path, 0, 10)

		return noders
	},
}

func getNoderPath(size int) noder.Path {
	p := noderPaths.Get().(noder.Path)

	if cap(p) < size {
		p = make(noder.Path, 0, size)
	} else {
		p = p[:0]
	}

	return p
}

func putNoderPath(p noder.Path) {
	noderPaths.Put(p)
}
