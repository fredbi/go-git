package object

type (
	TreeOption func(*treeOptions)

	treeOptions struct {
		*caches
	}
)

func applyTreeOptions(opts []TreeOption) *treeOptions {
	o := &treeOptions{}
	for _, apply := range opts {
		apply(o)
	}

	if o.caches == nil {
		o.caches = defaultCaches()
	}

	return o
}

// TreeWithDiffOptions injects the settings for DiffTree to tree building.
//
// In particular, this allows to share a tree cache.
func TreeWithDiffOptions(opts *DiffTreeOptions) TreeOption {
	return func(o *treeOptions) {
		o.caches = opts.caches
	}
}
