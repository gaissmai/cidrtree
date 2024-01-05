// Package cidrtree implements fast lookup (longest-prefix-match) for IP routing tables (IPv4/IPv6).
//
// The implementation is based on treaps, which have been augmented here for CIDRs.
//
// Treaps are randomized, self-balancing binary search trees. Due to the nature of treaps
// the lookups (readers) and the update (writer) can be easily decoupled.
// This is the perfect fit for a software router or firewall.
package cidrtree

import (
	mrand "math/rand"
	"net/netip"

	"github.com/gaissmai/extnetip"
)

// Table is an IPv4 and IPv6 routing table. The zero value is ready to use.
type Table struct {
	// make a treap for every IP version, the bits of the prefix are part of the weighted priority
	root4 *node
	root6 *node
}

// node is the recursive data structure of the treap.
type node struct {
	maxUpper *node // augment the treap, see also recalc()
	left     *node
	right    *node
	value    any
	cidr     netip.Prefix
	prio     uint64
}

// Insert adds pfx to the table with value val, changing the original table.
// If pfx is already present in the table, its value is set to val.
func (t *Table) Insert(pfx netip.Prefix, val any) {
	if pfx.Addr().Is4() {
		t.root4 = t.root4.insert(makeNode(pfx, val), false)
		return
	}
	t.root6 = t.root6.insert(makeNode(pfx, val), false)
}

// InsertImmutable adds pfx to the table with value val, returning a new table.
// If pfx is already present in the table, its value is set to val.
func (t Table) InsertImmutable(pfx netip.Prefix, val any) *Table {
	if pfx.Addr().Is4() {
		t.root4 = t.root4.insert(makeNode(pfx, val), true)
		return &t
	}
	t.root6 = t.root6.insert(makeNode(pfx, val), true)
	return &t
}

// insert into treap, changing nodes are copied, new treap is returned,
// old treap is modified if immutable is false.
// If node is already present in the table, its value is set to val.
func (n *node) insert(m *node, immutable bool) *node {
	if n == nil {
		// recursion stop condition
		return m
	}

	// if m is the new root?
	if m.prio >= n.prio {
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
		l, dupe, r := n.split(m.cidr, immutable)

		// replace dupe with m. m has same key but different prio than dupe, a join() is required
		if dupe != nil {
			return l.join(m.join(r, immutable), immutable)
		}

		// no duplicate, take m as new root
		//
		//     m
		//   /  \
		//  <m   >m
		//
		m.left, m.right = l, r
		m.recalc() // m has changed, recalc
		return m
	}

	cmp := compare(m.cidr, n.cidr)
	if cmp == 0 {
		// replace duplicate item with m, but m has different prio, a join() is required
		return n.left.join(m.join(n.right, immutable), immutable)
	}

	if immutable {
		n = n.copyNode()
	}

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
	}

	n.recalc() // n has changed, recalc
	return n
}

// DeleteImmutable removes the prefix if it exists, returns the new table and true, false if not found.
func (t Table) DeleteImmutable(pfx netip.Prefix) (*Table, bool) {
	pfx = pfx.Masked() // always canonicalize!

	is4 := pfx.Addr().Is4()

	n := t.root6
	if is4 {
		n = t.root4
	}

	// split/join must be immutable
	l, m, r := n.split(pfx, true)
	n = l.join(r, true)

	if is4 {
		t.root4 = n
	} else {
		t.root6 = n
	}

	ok := m != nil
	return &t, ok
}

// Delete removes the prefix from table, returns true if it exists, false otherwise.
func (t *Table) Delete(pfx netip.Prefix) bool {
	pfx = pfx.Masked() // always canonicalize!

	is4 := pfx.Addr().Is4()

	n := t.root6
	if is4 {
		n = t.root4
	}

	// split/join is mutable
	l, m, r := n.split(pfx, false)
	n = l.join(r, false)

	if is4 {
		t.root4 = n
	} else {
		t.root6 = n
	}

	return m != nil
}

// UnionImmutable combines any two tables immutable and returns the combined table.
// If there are duplicate entries, the value is taken from the other table.
func (t Table) UnionImmutable(other *Table) *Table {
	t.root4 = t.root4.union(other.root4, true, true)
	t.root6 = t.root6.union(other.root6, true, true)
	return &t
}

// Union combines two tables, changing the receiver table.
// If there are duplicate entries, the value is taken from the other table.
func (t *Table) Union(other *Table) {
	t.root4 = t.root4.union(other.root4, true, false)
	t.root6 = t.root6.union(other.root6, true, false)
}

// union two treaps.
// flag overwrite isn't public but needed as input for rec-descent calls, see below when trepa are swapped.
func (n *node) union(b *node, overwrite bool, immutable bool) *node {
	// recursion stop condition
	if n == nil {
		return b
	}
	if b == nil {
		return n
	}

	// swap treaps if needed, treap with higher prio remains as new root
	// also swap the overwrite flag
	if n.prio < b.prio {
		n, b = b, n
		overwrite = !overwrite
	}

	// immutable union, copy remaining root
	if immutable {
		n = n.copyNode()
	}

	// the treap with the lower priority is split with the root key in the treap
	// with the higher priority, skip duplicates
	l, dupe, r := b.split(n.cidr, immutable)

	// the treaps may have duplicate items
	if overwrite && dupe != nil {
		n.cidr = dupe.cidr
		n.value = dupe.value
	}

	// rec-descent
	n.left = n.left.union(l, overwrite, immutable)
	n.right = n.right.union(r, overwrite, immutable)

	n.recalc() // n has changed, recalc
	return n
}

// Walk iterates the cidrtree in ascending order.
// The callback function is called with the prefix and value of the respective node and the depth in the tree.
// If callback returns `false`, the iteration is aborted.
func (t Table) Walk(cb func(pfx netip.Prefix, val any) bool) {
	if !t.root4.walk(cb) {
		return
	}

	t.root6.walk(cb)
}

// walk tree in ascending prefix order.
func (n *node) walk(cb func(netip.Prefix, any) bool) bool {
	if n == nil {
		return true
	}

	// left
	if !n.left.walk(cb) {
		return false
	}

	// do-it
	if !cb(n.cidr, n.value) {
		return false
	}

	// right
	if !n.right.walk(cb) {
		return false
	}

	return true
}

// Lookup returns the longest-prefix-match (lpm) for given ip.
// If the ip isn't covered by any CIDR, the zero value and false is returned.
//
// Lookup does not allocate memory.
//
//	example:
//
//	▼
//	├─ 10.0.0.0/8
//	│  ├─ 10.0.0.0/24
//	│  └─ 10.0.1.0/24
//	├─ 127.0.0.0/8
//	│  └─ 127.0.0.1/32
//	├─ 169.254.0.0/16
//	├─ 172.16.0.0/12
//	└─ 192.168.0.0/16
//	   └─ 192.168.1.0/24
//	▼
//	└─ ::/0
//	   ├─ ::1/128
//	   ├─ 2000::/3
//	   │  └─ 2001:db8::/32
//	   ├─ fc00::/7
//	   ├─ fe80::/10
//	   └─ ff00::/8
//
//	    rtbl.Lookup(42.0.0.0)             returns (netip.Prefix{}, <nil>,  false)
//	    rtbl.Lookup(10.0.1.17)            returns (10.0.1.0/24,    <value>, true)
//	    rtbl.Lookup(2001:7c0:3100:1::111) returns (2000::/3,       <value>, true)
func (t Table) Lookup(ip netip.Addr) (lpm netip.Prefix, value any, ok bool) {
	if ip.Is4() {
		// don't return the depth
		lpm, value, ok, _ = t.root4.lpmIP(ip, 0)
		return
	}
	// don't return the depth
	lpm, value, ok, _ = t.root6.lpmIP(ip, 0)
	return
}

// lpmIP rec-descent
func (n *node) lpmIP(ip netip.Addr, depth int) (lpm netip.Prefix, value any, ok bool, atDepth int) {
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
		depth += 1
		n = n.left
	}

	// right backtracking
	if lpm, value, ok, atDepth = n.right.lpmIP(ip, depth+1); ok {
		return
	}

	// lpm match
	if n.cidr.Contains(ip) {
		return n.cidr, n.value, true, depth
	}

	// left rec-descent
	return n.left.lpmIP(ip, depth+1)
}

// LookupPrefix returns the longest-prefix-match (lpm) for given prefix.
// If the prefix isn't equal or covered by any CIDR in the table, the zero value and false is returned.
//
// LookupPrefix does not allocate memory.
//
//	example:
//
//	▼
//	├─ 10.0.0.0/8
//	│  ├─ 10.0.0.0/24
//	│  └─ 10.0.1.0/24
//	├─ 127.0.0.0/8
//	│  └─ 127.0.0.1/32
//	├─ 169.254.0.0/16
//	├─ 172.16.0.0/12
//	└─ 192.168.0.0/16
//	   └─ 192.168.1.0/24
//	▼
//	└─ ::/0
//	   ├─ ::1/128
//	   ├─ 2000::/3
//	   │  └─ 2001:db8::/32
//	   ├─ fc00::/7
//	   ├─ fe80::/10
//	   └─ ff00::/8
//
//	    rtbl.LookupPrefix(42.0.0.0/8)         returns (netip.Prefix{}, <nil>,  false)
//	    rtbl.LookupPrefix(10.0.1.0/29)        returns (10.0.1.0/24,    <value>, true)
//	    rtbl.LookupPrefix(192.168.0.0/16)     returns (192.168.0.0/16, <value>, true)
//	    rtbl.LookupPrefix(2001:7c0:3100::/40) returns (2000::/3,       <value>, true)
func (t Table) LookupPrefix(pfx netip.Prefix) (lpm netip.Prefix, value any, ok bool) {
	if pfx.Addr().Is4() {
		// don't return the depth
		lpm, value, ok, _ = t.root4.lpmCIDR(pfx, 0)
		return
	}
	// don't return the depth
	lpm, value, ok, _ = t.root6.lpmCIDR(pfx, 0)
	return
}

// lpmCIDR rec-descent
func (n *node) lpmCIDR(pfx netip.Prefix, depth int) (lpm netip.Prefix, value any, ok bool, atDepth int) {
	for {
		// recursion stop condition
		if n == nil {
			return
		}

		// fast exit with (augmented) max upper value
		if pfxTooBig(pfx, n.maxUpper.cidr) {
			// recursion stop condition
			return
		}

		// if cidr is already less-or-equal pfx
		cmp := compare(n.cidr, pfx)

		// match!
		if cmp == 0 {
			return n.cidr, n.value, true, depth
		}

		if cmp < 0 {
			break // ok, proceed with this cidr
		}

		// fast traverse to left
		depth += 1
		n = n.left
	}

	// right backtracking
	if lpm, value, ok, atDepth = n.right.lpmCIDR(pfx, depth+1); ok {
		return
	}

	// lpm match:
	// CIDRs are equal ...
	if n.cidr == pfx {
		return n.cidr, n.value, true, depth
	}

	// ... or supernets
	if n.cidr.Contains(pfx.Addr()) {
		return n.cidr, n.value, true, depth
	}

	// ... or disjunct

	// left rec-descent
	return n.left.lpmCIDR(pfx, depth+1)
}

// Clone, deep cloning of the routing table.
func (t Table) Clone() *Table {
	t.root4 = t.root4.clone()
	t.root6 = t.root6.clone()
	return &t
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

	if n.prio > m.prio {
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
func makeNode(pfx netip.Prefix, val any) *node {
	n := new(node)
	n.cidr = pfx.Masked() // always store the prefix in normalized form
	n.value = val
	n.prio = mrand.Uint64()
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
	if a == b {
		return 0
	}

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

// cmpRR compares the prefixes last address.
func cmpRR(a, b netip.Prefix) int {
	if a == b {
		return 0
	}
	_, aLast := extnetip.Range(a)
	_, bLast := extnetip.Range(b)

	return aLast.Compare(bLast)
}

// ipTooBig returns true if ip is greater than prefix last ip address.
//
//		  false                    true
//		    |                        |
//		    V                        V
//
//	  ------- other -------->
func ipTooBig(ip netip.Addr, other netip.Prefix) bool {
	_, pLastIP := extnetip.Range(other)
	return ip.Compare(pLastIP) > 0
}

// pfxTooBig returns true if prefix last address is greater than other last ip address.
//
//	------------ pfx --------------> true
//	------ pfx ----> false
//
//	------- other -------->
func pfxTooBig(pfx netip.Prefix, other netip.Prefix) bool {
	_, pfxLastIP := extnetip.Range(pfx)
	return ipTooBig(pfxLastIP, other)
}
