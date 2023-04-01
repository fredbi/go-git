package fsnoder

import (
	"reflect"
	"sort"

	"github.com/go-git/go-git/v5/utils/merkletrie/noder"

	. "gopkg.in/check.v1"
)

type DirSuite struct{}

var _ = Suite(&DirSuite{})

func (s *DirSuite) TestIsDir(c *C) {
	noName, err := newDir("", nil)
	c.Assert(err, IsNil)
	c.Assert(noName.IsDir(), Equals, true)

	empty, err := newDir("empty", nil)
	c.Assert(err, IsNil)
	c.Assert(empty.IsDir(), Equals, true)

	root, err := newDir("foo", []noder.Noder{empty})
	c.Assert(err, IsNil)
	c.Assert(root.IsDir(), Equals, true)
}

func assertChildren(c *C, n noder.Noder, expected []noder.Noder) {
	numChildren, err := n.NumChildren()
	c.Assert(err, IsNil)
	c.Assert(numChildren, Equals, len(expected))

	children, err := n.Children()
	c.Assert(err, IsNil)
	c.Assert(children, sortedSliceEquals, expected)
}

type sortedSliceEqualsChecker struct {
	*CheckerInfo
}

var sortedSliceEquals Checker = &sortedSliceEqualsChecker{
	&CheckerInfo{
		Name:   "sortedSliceEquals",
		Params: []string{"obtained", "expected"},
	},
}

func (checker *sortedSliceEqualsChecker) Check(
	params []interface{}, names []string) (result bool, error string) {
	a, ok := params[0].([]noder.Noder)
	if !ok {
		return false, "first parameter must be a []noder.Noder"
	}
	b, ok := params[1].([]noder.Noder)
	if !ok {
		return false, "second parameter must be a []noder.Noder"
	}
	sort.Sort(byName(a))
	sort.Sort(byName(b))

	return reflect.DeepEqual(a, b), ""
}

func (s *DirSuite) TestNewDirectoryNoNameAndEmpty(c *C) {
	root, err := newDir("", nil)
	c.Assert(err, IsNil)

	c.Assert(root.Hash(), DeepEquals,
		[24]byte{0xca, 0x40, 0xf8, 0x67, 0x57, 0x8c, 0x32, 0x1c})
	c.Assert(root.Name(), Equals, "")
	assertChildren(c, root, noder.NoChildren)
	c.Assert(root.String(), Equals, "()")
}

func (s *DirSuite) TestNewDirectoryEmpty(c *C) {
	root, err := newDir("root", nil)
	c.Assert(err, IsNil)

	c.Assert(root.Hash(), DeepEquals,
		[24]byte{0xca, 0x40, 0xf8, 0x67, 0x57, 0x8c, 0x32, 0x1c})
	c.Assert(root.Name(), Equals, "root")
	assertChildren(c, root, noder.NoChildren)
	c.Assert(root.String(), Equals, "root()")
}

func (s *DirSuite) TestEmptyDirsHaveSameHash(c *C) {
	d1, err := newDir("foo", nil)
	c.Assert(err, IsNil)

	d2, err := newDir("bar", nil)
	c.Assert(err, IsNil)

	c.Assert(d1.Hash(), DeepEquals, d2.Hash())
}

func (s *DirSuite) TestNewDirWithEmptyDir(c *C) {
	empty, err := newDir("empty", nil)
	c.Assert(err, IsNil)

	root, err := newDir("", []noder.Noder{empty})
	c.Assert(err, IsNil)

	// obtained []uint8 = []byte{0xc5, 0x31, 0xe, 0x1f, 0xc4, 0xb5, 0xff, 0xb5}
	c.Assert(root.Hash(), DeepEquals,
		[24]uint8{0xc5, 0x31, 0xe, 0x1f, 0xc4, 0xb5, 0xff, 0xb5})
	c.Assert(root.Name(), Equals, "")
	assertChildren(c, root, []noder.Noder{empty})
	c.Assert(root.String(), Equals, "(empty())")
}

func (s *DirSuite) TestNewDirWithOneEmptyFile(c *C) {
	empty, err := newFile("name", "")
	c.Assert(err, IsNil)

	root, err := newDir("", []noder.Noder{empty})
	c.Assert(err, IsNil)
	c.Assert(root.Hash(), DeepEquals,
		[24]uint8{0x1e, 0x8b, 0x24, 0x9e, 0xa0, 0x59, 0xdb, 0x42})
	c.Assert(root.Name(), Equals, "")
	assertChildren(c, root, []noder.Noder{empty})
	c.Assert(root.String(), Equals, "(name<>)")
}

func (s *DirSuite) TestNewDirWithOneFile(c *C) {
	a, err := newFile("a", "1")
	c.Assert(err, IsNil)

	root, err := newDir("", []noder.Noder{a})
	c.Assert(err, IsNil)
	c.Assert(root.Hash(), DeepEquals,
		[24]uint8{0x40, 0x36, 0xc9, 0x2, 0xbf, 0x64, 0x93, 0x28})
	c.Assert(root.Name(), Equals, "")
	assertChildren(c, root, []noder.Noder{a})
	c.Assert(root.String(), Equals, "(a<1>)")
}

func (s *DirSuite) TestDirsWithSameFileHaveSameHash(c *C) {
	f1, err := newFile("a", "1")
	c.Assert(err, IsNil)
	r1, err := newDir("", []noder.Noder{f1})
	c.Assert(err, IsNil)

	f2, err := newFile("a", "1")
	c.Assert(err, IsNil)
	r2, err := newDir("", []noder.Noder{f2})
	c.Assert(err, IsNil)

	c.Assert(r1.Hash(), DeepEquals, r2.Hash())
}

func (s *DirSuite) TestDirsWithDifferentFileContentHaveDifferentHash(c *C) {
	f1, err := newFile("a", "1")
	c.Assert(err, IsNil)
	r1, err := newDir("", []noder.Noder{f1})
	c.Assert(err, IsNil)

	f2, err := newFile("a", "2")
	c.Assert(err, IsNil)
	r2, err := newDir("", []noder.Noder{f2})
	c.Assert(err, IsNil)

	c.Assert(r1.Hash(), Not(DeepEquals), r2.Hash())
}

func (s *DirSuite) TestDirsWithDifferentFileNameHaveDifferentHash(c *C) {
	f1, err := newFile("a", "1")
	c.Assert(err, IsNil)
	r1, err := newDir("", []noder.Noder{f1})
	c.Assert(err, IsNil)

	f2, err := newFile("b", "1")
	c.Assert(err, IsNil)
	r2, err := newDir("", []noder.Noder{f2})
	c.Assert(err, IsNil)

	c.Assert(r1.Hash(), Not(DeepEquals), r2.Hash())
}

func (s *DirSuite) TestDirsWithDifferentFileHaveDifferentHash(c *C) {
	f1, err := newFile("a", "1")
	c.Assert(err, IsNil)
	r1, err := newDir("", []noder.Noder{f1})
	c.Assert(err, IsNil)

	f2, err := newFile("b", "2")
	c.Assert(err, IsNil)
	r2, err := newDir("", []noder.Noder{f2})
	c.Assert(err, IsNil)

	c.Assert(r1.Hash(), Not(DeepEquals), r2.Hash())
}

func (s *DirSuite) TestDirWithEmptyDirHasDifferentHashThanEmptyDir(c *C) {
	f, err := newFile("a", "")
	c.Assert(err, IsNil)
	r1, err := newDir("", []noder.Noder{f})
	c.Assert(err, IsNil)

	d, err := newDir("a", nil)
	c.Assert(err, IsNil)
	r2, err := newDir("", []noder.Noder{d})
	c.Assert(err, IsNil)

	c.Assert(r1.Hash(), Not(DeepEquals), r2.Hash())
}

func (s *DirSuite) TestNewDirWithTwoFilesSameContent(c *C) {
	a1, err := newFile("a", "1")
	c.Assert(err, IsNil)
	b1, err := newFile("b", "1")
	c.Assert(err, IsNil)

	root, err := newDir("", []noder.Noder{a1, b1})
	c.Assert(err, IsNil)

	c.Assert(root.Hash(), DeepEquals,
		[24]uint8{0x12, 0xfc, 0xbd, 0x6, 0xc3, 0xb6, 0x30, 0x9b})
	c.Assert(root.Name(), Equals, "")
	assertChildren(c, root, []noder.Noder{b1, a1})
	c.Assert(root.String(), Equals, "(a<1> b<1>)")
}

func (s *DirSuite) TestNewDirWithTwoFilesDifferentContent(c *C) {
	a1, err := newFile("a", "1")
	c.Assert(err, IsNil)
	b2, err := newFile("b", "2")
	c.Assert(err, IsNil)

	root, err := newDir("", []noder.Noder{a1, b2})
	c.Assert(err, IsNil)

	c.Assert(root.Hash(), DeepEquals,
		[24]uint8{0x93, 0x26, 0x9b, 0x56, 0x7d, 0xf9, 0x9b, 0x95})
	c.Assert(root.Name(), Equals, "")
	assertChildren(c, root, []noder.Noder{b2, a1})
}

func (s *DirSuite) TestCrazy(c *C) {
	//           ""
	//            |
	//   -------------------------
	//   |    |      |      |    |
	//  a1    B     c1     d2    E
	//        |                  |
	//   -------------           E
	//   |   |   |   |           |
	//   A   B   X   c1          E
	//           |               |
	//          a1               e1
	e1, err := newFile("e", "1")
	c.Assert(err, IsNil)
	E, err := newDir("e", []noder.Noder{e1})
	c.Assert(err, IsNil)
	E, err = newDir("e", []noder.Noder{E})
	c.Assert(err, IsNil)
	E, err = newDir("e", []noder.Noder{E})
	c.Assert(err, IsNil)

	A, err := newDir("a", nil)
	c.Assert(err, IsNil)
	B, err := newDir("b", nil)
	c.Assert(err, IsNil)
	a1, err := newFile("a", "1")
	c.Assert(err, IsNil)
	X, err := newDir("x", []noder.Noder{a1})
	c.Assert(err, IsNil)
	c1, err := newFile("c", "1")
	c.Assert(err, IsNil)
	B, err = newDir("b", []noder.Noder{c1, B, X, A})
	c.Assert(err, IsNil)

	a1, err = newFile("a", "1")
	c.Assert(err, IsNil)
	c1, err = newFile("c", "1")
	c.Assert(err, IsNil)
	d2, err := newFile("d", "2")
	c.Assert(err, IsNil)

	root, err := newDir("", []noder.Noder{a1, d2, E, B, c1})
	c.Assert(err, IsNil)

	c.Assert(root.Hash(), DeepEquals,
		[24]uint8{0x90, 0x2c, 0x2, 0xfd, 0x78, 0xd7, 0x6d, 0x48})
	c.Assert(root.Name(), Equals, "")
	assertChildren(c, root, []noder.Noder{E, c1, B, a1, d2})
	c.Assert(root.String(), Equals,
		"(a<1> b(a() b() c<1> x(a<1>)) c<1> d<2> e(e(e(e<1>))))")
}

func (s *DirSuite) TestDirCannotHaveDirWithNoName(c *C) {
	noName, err := newDir("", nil)
	c.Assert(err, IsNil)

	_, err = newDir("", []noder.Noder{noName})
	c.Assert(err, Not(IsNil))
}

func (s *DirSuite) TestDirCannotHaveDuplicatedFiles(c *C) {
	f1, err := newFile("a", "1")
	c.Assert(err, IsNil)

	f2, err := newFile("a", "1")
	c.Assert(err, IsNil)

	_, err = newDir("", []noder.Noder{f1, f2})
	c.Assert(err, Not(IsNil))
}

func (s *DirSuite) TestDirCannotHaveDuplicatedFileNames(c *C) {
	a1, err := newFile("a", "1")
	c.Assert(err, IsNil)

	a2, err := newFile("a", "2")
	c.Assert(err, IsNil)

	_, err = newDir("", []noder.Noder{a1, a2})
	c.Assert(err, Not(IsNil))
}

func (s *DirSuite) TestDirCannotHaveDuplicatedDirNames(c *C) {
	d1, err := newDir("a", nil)
	c.Assert(err, IsNil)

	d2, err := newDir("a", nil)
	c.Assert(err, IsNil)

	_, err = newDir("", []noder.Noder{d1, d2})
	c.Assert(err, Not(IsNil))
}

func (s *DirSuite) TestDirCannotHaveDirAndFileWithSameName(c *C) {
	f, err := newFile("a", "")
	c.Assert(err, IsNil)

	d, err := newDir("a", nil)
	c.Assert(err, IsNil)

	_, err = newDir("", []noder.Noder{f, d})
	c.Assert(err, Not(IsNil))
}

func (s *DirSuite) TestUnsortedString(c *C) {
	b, err := newDir("b", nil)
	c.Assert(err, IsNil)

	z, err := newDir("z", nil)
	c.Assert(err, IsNil)

	a1, err := newFile("a", "1")
	c.Assert(err, IsNil)

	c2, err := newFile("c", "2")
	c.Assert(err, IsNil)

	d3, err := newFile("d", "3")
	c.Assert(err, IsNil)

	d, err := newDir("d", []noder.Noder{c2, z, d3, a1, b})
	c.Assert(err, IsNil)

	c.Assert(d.String(), Equals, "d(a<1> b() c<2> d<3> z())")
}
