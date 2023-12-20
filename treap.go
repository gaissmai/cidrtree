// Package cidrtree provides fast IP to CIDR lookup (longest prefix match).
//
// This package is a specialization of the more generic [interval package] of the same author,
// but explicit for CIDRs. It has a narrow focus with a smaller and simpler API.
//
// [interval package]: https://github.com/gaissmai/interval
package cidrtree

import (
	"net/netip"
	"sync"
)

type (
	// Tree is the public handle to the hidden implementation.
	Tree struct {
		// make a treap for every IP version, not really necessary but a little bit faster
		// since the augmented field with maxUpper cidr bound does not cross the IP version domains.
		root4 *node
		root6 *node
	}

	// Route, CIDRs are keys and the values are typically the next-hop of the route or an array of next-hops in ECMP routing.
	Route struct {
		CIDR  netip.Prefix
		Value any
	}

	// node is the recursive data structure of the treap.
	// The heap priority is not stored in the node, it is calculated (crc32) when needed from the prefix.
	// The same input always produces the same binary tree since the heap priority
	// is defined by the crc of the cidr.
	node struct {
		maxUpper *node // augment the treap, see also recalc()
		left     *node
		right    *node
		value    any
		cidr     netip.Prefix
	}
)

// New initializes the cidrtree with zero or more routes.
// Duplicate prefixes are just skipped.
func New(routes ...Route) Tree {
	var t Tree
	t.InsertMutable(routes...)
	return t
}

// NewConcurrent, splits the input data into chunks, fan-out to [New] and recombine the trees (mutable) with [Union].
//
// Convenience function for initializing the cidrtree for large inputs (> 100_000).
// A good value reference for jobs is the number of logical CPUs [runtine.NumCPU] usable by the current process.
func NewConcurrent(jobs int, routes ...Route) Tree {
	// define a min chunk size, don't split in too small chunks
	const minChunkSize = 25_000

	// no fan-out for small input slice or just one job
	l := len(routes)
	if l < minChunkSize || jobs <= 1 {
		return New(routes...)
	}

	chunkSize := l/jobs + 1
	if chunkSize < minChunkSize {
		chunkSize = minChunkSize

		// don't use go routine and result channel for just one chunk
		if l < chunkSize {
			return New(routes...)
		}
	}

	var wg sync.WaitGroup
	var chunk []Route
	partialTrees := make(chan Tree)

	// fan-out
	for ; l > 0; l = len(routes) {
		// partition input into chunks
		switch {
		case l > chunkSize:
			chunk = routes[:chunkSize]
			routes = routes[chunkSize:]
		default: // rest
			chunk = routes[:l]
			routes = nil
		}

		wg.Add(1)
		go func(chunk ...Route) {
			defer wg.Done()
			partialTrees <- New(chunk...)
		}(chunk...)
	}

	// wait and close chan
	go func() {
		wg.Wait()
		close(partialTrees)
	}()

	// fan-in, mutable
	var t Tree
	for other := range partialTrees {
		t = t.Union(other, false) // immutable is false
	}
	return t
}

// Insert routes into the tree, returns the new Tree.
// Duplicate prefixes are just skipped.
func (t Tree) Insert(routes ...Route) Tree {
	for _, route := range routes {
		if route.CIDR.Addr().Is4() {
			t.root4 = t.root4.insert(makeNode(route), true)
		} else {
			t.root6 = t.root6.insert(makeNode(route), true)
		}
	}

	return t
}

// InsertMutable insert routes into the tree, changing the original tree.
// Duplicate prefixes are just skipped.
// If the original tree does not need to be preserved then this is much faster than the immutable insert.
func (t *Tree) InsertMutable(routes ...Route) {
	for _, route := range routes {
		if route.CIDR.Addr().Is4() {
			t.root4 = t.root4.insert(makeNode(route), false)
		} else {
			t.root6 = t.root6.insert(makeNode(route), false)
		}
	}
}

// insert into tree, changing nodes are copied, new treap is returned, old treap is modified if immutable is false.
func (n *node) insert(m *node, immutable bool) *node {
	if n == nil {
		// recursion stop condition
		return m
	}

	// if m is the new root?
	if m.prio() > n.prio() {
		//
		//          m
		//          | split t in ( <m | dupe | >m )
		//          v
		//       t
		//      / \
		//    l     d(upe)
		//   / \   / \
		//  l   r l   r
		//           /
		//          l
		//
		l, _, r := n.split(m.cidr, immutable)

		// no duplicate handling, take m as new root
		//
		//     m
		//   /  \
		//  <m   >m
		//
		m.left, m.right = l, r
		m.recalc() // m has changed, recalc
		return m
	}

	if immutable {
		n = n.copyNode()
	}

	cmp := compare(m.cidr, n.cidr)
	switch {
	case cmp < 0: // rec-descent
		n.left = n.left.insert(m, immutable)
		//
		//       R
		// m    l r
		//     l   r
		//
	case cmp > 0: // rec-descent
		n.right = n.right.insert(m, immutable)
		//
		//   R
		//  l r    m
		// l   r
		//
	default:
		// cmp == 0, skip duplicate
	}

	n.recalc() // n has changed, recalc
	return n
}

// Delete removes the cdir if it exists, returns the new tree and true, false if not found.
func (t Tree) Delete(cidr netip.Prefix) (Tree, bool) {
	cidr = cidr.Masked() // always canonicalize!

	is4 := cidr.Addr().Is4()

	n := t.root6
	if is4 {
		n = t.root4
	}

	// split/join must be immutable
	l, m, r := n.split(cidr, true)
	n = l.join(r, true)

	if is4 {
		t.root4 = n
	} else {
		t.root6 = n
	}

	ok := m != nil
	return t, ok
}

// DeleteMutable removes the cidr from tree, returns true if it exists, false otherwise.
// If the original tree does not need to be preserved then this is much faster than the immutable delete.
func (t *Tree) DeleteMutable(cidr netip.Prefix) bool {
	cidr = cidr.Masked() // always canonicalize!

	is4 := cidr.Addr().Is4()

	n := t.root6
	if is4 {
		n = t.root4
	}

	// split/join is mutable
	l, m, r := n.split(cidr, false)
	n = l.join(r, false)

	if is4 {
		t.root4 = n
	} else {
		t.root6 = n
	}

	return m != nil
}

// Union combines any two trees. Duplicates are skipped.
//
// The "immutable" flag controls whether the two trees are allowed to be modified.
func (t Tree) Union(other Tree, immutable bool) Tree {
	t.root4 = t.root4.union(other.root4, immutable)
	t.root6 = t.root6.union(other.root6, immutable)
	return t
}

func (n *node) union(b *node, immutable bool) *node {
	// recursion stop condition
	if n == nil {
		return b
	}
	if b == nil {
		return n
	}

	// swap treaps if needed, treap with higher prio remains as new root
	if n.prio() < b.prio() {
		n, b = b, n
	}

	// immutable union, copy remaining root
	if immutable {
		n = n.copyNode()
	}

	// the treap with the lower priority is split with the root key in the treap
	// with the higher priority, skip duplicates
	l, _, r := b.split(n.cidr, immutable)

	// rec-descent
	n.left = n.left.union(l, immutable)
	n.right = n.right.union(r, immutable)

	n.recalc() // n has changed, recalc
	return n
}

// Lookup returns the longest-prefix-match for ip and associated value.
// If the ip isn't covered by any CIDR, the zero value and false is returned.
// The algorithm for Lookup does NOT allocate memory.
//
//  example:
//
//  ▼
//  ├─ 10.0.0.0/8
//  │  ├─ 10.0.0.0/24
//  │  └─ 10.0.1.0/24
//  ├─ 127.0.0.0/8
//  │  └─ 127.0.0.1/32
//  ├─ 169.254.0.0/16
//  ├─ 172.16.0.0/12
//  └─ 192.168.0.0/16
//     └─ 192.168.1.0/24
//  ▼
//  └─ ::/0
//     ├─ ::1/128
//     ├─ 2000::/3
//     │  └─ 2001:db8::/32
//     ├─ fc00::/7
//     ├─ fe80::/10
//     └─ ff00::/8
//
//      tree.Lookup("42.0.0.0")             returns (netip.Prefix{}, <nil>,  false)
//      tree.Lookup("10.0.1.17")            returns (10.0.1.0/24,    <value>, true)
//      tree.Lookup("2001:7c0:3100:1::111") returns (2000::/3,       <value>, true)
//
func (t Tree) Lookup(ip netip.Addr) (cidr netip.Prefix, value any, ok bool) {
	if ip.Is4() {
		return t.root4.lookup(ip)
	}
	return t.root6.lookup(ip)
}

// lookup rec-descent
func (n *node) lookup(ip netip.Addr) (cidr netip.Prefix, value any, ok bool) {
	for {
		// recursion stop condition
		if n == nil {
			return
		}

		// fast exit with (augmented) max upper value
		if ipTooBig(ip, n.maxUpper.cidr) {
			// recursion stop condition
			return
		}

		// if cidr is already less-or-equal ip
		if n.cidr.Addr().Compare(ip) <= 0 {
			break // ok, proceed with this cidr
		}

		// fast traverse to left
		n = n.left
	}

	// right backtracking
	if cidr, value, ok = n.right.lookup(ip); ok {
		return
	}

	// lpm match
	if n.cidr.Contains(ip) {
		return n.cidr, n.value, true
	}

	// left rec-descent
	return n.left.lookup(ip)
}

// Clone, deep cloning of the CIDR tree.
func (t Tree) Clone() Tree {
	t.root4 = t.root4.clone()
	t.root6 = t.root6.clone()
	return t
}

func (n *node) clone() *node {
	if n == nil {
		return n
	}
	n = n.copyNode()

	n.left = n.left.clone()
	n.right = n.right.clone()

	n.recalc()

	return n
}

// ##############################################################
//        main treap algo methods: split and join
// ##############################################################

// split the treap into all nodes that compare less-than, equal
// and greater-than the provided cidr (BST key). The resulting nodes are
// properly formed treaps or nil.
// If the split must be immutable, first copy concerned nodes.
func (n *node) split(cidr netip.Prefix, immutable bool) (left, mid, right *node) {
	// recursion stop condition
	if n == nil {
		return nil, nil, nil
	}

	if immutable {
		n = n.copyNode()
	}

	cmp := compare(n.cidr, cidr)

	switch {
	case cmp < 0:
		l, m, r := n.right.split(cidr, immutable)
		n.right = l
		n.recalc() // n has changed, recalc
		return n, m, r
		//
		//       (k)
		//      R
		//     l r   ==> (R.r, m, r) = R.r.split(k)
		//    l   r
		//
	case cmp > 0:
		l, m, r := n.left.split(cidr, immutable)
		n.left = r
		n.recalc() // n has changed, recalc
		return l, m, n
		//
		//   (k)
		//      R
		//     l r   ==> (l, m, R.l) = R.l.split(k)
		//    l   r
		//
	default:
		l, r := n.left, n.right
		n.left, n.right = nil, nil
		n.recalc() // n has changed, recalc
		return l, n, r
		//
		//     (k)
		//      R
		//     l r   ==> (R.l, R, R.r)
		//    l   r
		//
	}
}

// join combines two disjunct treaps. All nodes in treap n have keys <= that of treap m
// for this algorithm to work correctly. If the join must be immutable, first copy concerned nodes.
func (n *node) join(m *node, immutable bool) *node {
	// recursion stop condition
	if n == nil {
		return m
	}
	if m == nil {
		return n
	}

	if n.prio() > m.prio() {
		//     n
		//    l r    m
		//          l r
		//
		if immutable {
			n = n.copyNode()
		}
		n.right = n.right.join(m, immutable)
		n.recalc() // n has changed, recalc
		return n
	}
	//
	//            m
	//      n    l r
	//     l r
	//
	if immutable {
		m = m.copyNode()
	}
	m.left = n.join(m.left, immutable)
	m.recalc() // m has changed, recalc
	return m
}

// ###########################################################
//            mothers little helpers
// ###########################################################

// makeNode, create new node with cidr.
func makeNode(route Route) *node {
	n := new(node)
	n.cidr = route.CIDR.Masked() // always store the prefix in canonical form
	n.value = route.Value
	n.recalc() // init the augmented field with recalc
	return n
}

// copyNode, make a shallow copy of the pointers and the cidr.
func (n *node) copyNode() *node {
	c := *n
	return &c
}

// recalc the augmented fields in treap node after each creation/modification
// with values in descendants.
// Only one level deeper must be considered. The treap datastructure is very easy to augment.
func (n *node) recalc() {
	if n == nil {
		return
	}

	n.maxUpper = n

	if n.right != nil {
		if cmpRR(n.right.maxUpper.cidr, n.maxUpper.cidr) > 0 {
			n.maxUpper = n.right.maxUpper
		}
	}

	if n.left != nil {
		if cmpRR(n.left.maxUpper.cidr, n.maxUpper.cidr) > 0 {
			n.maxUpper = n.left.maxUpper
		}
	}
}

// compare two prefixes and sort by the left address,
// or if equal always sort the superset to the left.
func compare(a, b netip.Prefix) int {
	// compare left points of cidrs
	ll := a.Addr().Compare(b.Addr())

	if ll != 0 {
		return ll
	}

	// ll == 0, sort superset to the left
	aBits := a.Bits()
	bBits := b.Bits()

	switch {
	case aBits < bBits:
		return -1
	case aBits > bBits:
		return 1
	}

	return 0
}

// cmpRR compares (indirect) the prefixes last address.
func cmpRR(a, b netip.Prefix) int {
	if a == b {
		return 0
	}

	ll := a.Addr().Compare(b.Addr())
	overlaps := a.Overlaps(b)

	switch {
	case ll < 0:
		if overlaps {
			return 1
		}
		return -1
	case ll > 0:
		if overlaps {
			return -1
		}
		return 1
	}

	// ll == 0 && rr != 0
	if a.Bits() > b.Bits() {
		return -1
	}
	return 1
}

// ipTooBig returns true if ip is greater than prefix last address.
// The test must be indirect since netip has no method to get the last address of the prefix.
func ipTooBig(ip netip.Addr, p netip.Prefix) bool {
	if p.Contains(ip) {
		return false
	}
	if ip.Compare(p.Addr()) > 0 {
		// ... but not contained, indirect proof for tooBig
		return true
	}
	return false
}
