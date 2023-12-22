// Package cidrtree implements fast lookup (longest-prefix-match) for IP routing tables (IPv4/IPv6).
//
// The implementation is based on treaps, which have been augmented here for CIDRs.
//
// Treaps are randomized, self-balancing binary search trees. Due to the nature of treaps,
// the lookups (readers) and updates (writers) can be decoupled,
// which is a perfect fit for a software router or firewall.
package cidrtree

import (
	"hash/fnv"
	"net/netip"

	"github.com/gaissmai/extnetip"
)

type (
	// Table is an IPv4 and IPv6 routing table. The zero value is ready to use.
	Table struct {
		// make a treap for every IP version, not really necessary but faster
		// since the augmented field with maxUpper cidr bound does not cross the IP version domains.
		root4 *node
		root6 *node
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

// New returns a pointer to the zero value of Table.
func New() *Table {
	return &Table{}
}

// Insert routes into the table, returns the new table.
// Duplicate prefixes are just skipped.
func (t Table) Insert(pfx netip.Prefix, val any) *Table {
	if pfx.Addr().Is4() {
		t.root4 = t.root4.insert(makeNode(pfx, val), true)
		return &t
	}
	t.root6 = t.root6.insert(makeNode(pfx, val), true)
	return &t
}

// InsertMutable insert routes into the table, changing the original table.
// Duplicate prefixes are just skipped.
// If the original table does not need to be preserved then this is much faster than the immutable insert.
func (t *Table) InsertMutable(pfx netip.Prefix, val any) {
	if pfx.Addr().Is4() {
		t.root4 = t.root4.insert(makeNode(pfx, val), false)
		return
	}
	t.root6 = t.root6.insert(makeNode(pfx, val), false)
}

// insert into treap, changing nodes are copied, new treap is returned,
// old treap is modified if immutable is false.
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

// Delete removes the cdir if it exists, returns the new table and true, false if not found.
func (t Table) Delete(cidr netip.Prefix) (*Table, bool) {
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
	return &t, ok
}

// DeleteMutable removes the cidr from table, returns true if it exists, false otherwise.
// If the original table does not need to be preserved then this is much faster than the immutable delete.
func (t *Table) DeleteMutable(cidr netip.Prefix) bool {
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

// Union combines any two tables. The tables tables are not changed.
// Duplicates are skipped.
func (t Table) Union(other *Table) *Table {
	t.root4 = t.root4.union(other.root4, true)
	t.root6 = t.root6.union(other.root6, true)
	return &t
}

// UnionMutable combines two tables, changing the receiver table.
// Duplicates are skipped.
func (t *Table) UnionMutable(other *Table) {
	t.root4 = t.root4.union(other.root4, false)
	t.root6 = t.root6.union(other.root6, false)
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

// Walk iterates the cidrtree in ascending order.
// The callback function is called with the Route struct of the respective node.
// If callback returns `false`, the iteration is aborted.
func (t *Table) Walk(cb func(pfx netip.Prefix, val any) bool) {
	if !t.root4.walk(cb) {
		return
	}

	t.root6.walk(cb)
}

// walk tree in ascending prefix order.
func (n *node) walk(cb func(pfy netip.Prefix, val any) bool) bool {
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

// LookupIP returns the longest-prefix-match (lpm) for given ip.
// If the ip isn't covered by any CIDR, the zero value and false is returned.
// LookupIP does not allocate memory.
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
//	    rtbl.LookupIP(42.0.0.0)             returns (netip.Prefix{}, <nil>,  false)
//	    rtbl.LookupIP(10.0.1.17)            returns (10.0.1.0/24,    <value>, true)
//	    rtbl.LookupIP(2001:7c0:3100:1::111) returns (2000::/3,       <value>, true)
func (t Table) LookupIP(ip netip.Addr) (lpm netip.Prefix, value any, ok bool) {
	if ip.Is4() {
		return t.root4.lookupIP(ip)
	}
	return t.root6.lookupIP(ip)
}

// lookupIP rec-descent
func (n *node) lookupIP(ip netip.Addr) (lpm netip.Prefix, value any, ok bool) {
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
	if lpm, value, ok = n.right.lookupIP(ip); ok {
		return
	}

	// lpm match
	if n.cidr.Contains(ip) {
		return n.cidr, n.value, true
	}

	// left rec-descent
	return n.left.lookupIP(ip)
}

// LookupCIDR returns the longest-prefix-match (lpm) for given prefix.
// If the prefix isn't equal or covered by any CIDR in the table, the zero value and false is returned.
//
// LookupCIDR does not allocate memory.
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
//	    rtbl.LookupCIDR(42.0.0.0/8)         returns (netip.Prefix{}, <nil>,  false)
//	    rtbl.LookupCIDR(10.0.1.0/29)        returns (10.0.1.0/24,    <value>, true)
//	    rtbl.LookupCIDR(192.168.0.0/16)     returns (192.168.0.0/16, <value>, true)
//	    rtbl.LookupCIDR(2001:7c0:3100::/40) returns (2000::/3,       <value>, true)
func (t Table) LookupCIDR(pfx netip.Prefix) (lpm netip.Prefix, value any, ok bool) {
	if pfx.Addr().Is4() {
		return t.root4.lookupCIDR(pfx)
	}
	return t.root6.lookupCIDR(pfx)
}

// lookupCIDR rec-descent
func (n *node) lookupCIDR(pfx netip.Prefix) (lpm netip.Prefix, value any, ok bool) {
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
			return n.cidr, n.value, true
		}

		if cmp < 0 {
			break // ok, proceed with this cidr
		}

		// fast traverse to left
		n = n.left
	}

	// right backtracking
	if lpm, value, ok = n.right.lookupCIDR(pfx); ok {
		return
	}

	// lpm match:
	// CIDRs are equal ...
	if n.cidr == pfx {
		return n.cidr, n.value, true
	}

	// ... or supernets
	if n.cidr.Contains(pfx.Addr()) {
		return n.cidr, n.value, true
	}

	// ... or disjunct

	// left rec-descent
	return n.left.lookupCIDR(pfx)
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

func (n *node) prio() uint64 {
	// MarshalBinary would allocate
	raw := n.cidr.Addr().As16()
	bits := byte(n.cidr.Bits())

	data := make([]byte, 0, 17)
	data = append(data, raw[:]...)
	data = append(data, bits)

	h := fnv.New64()
	h.Write(data[:])
	return h.Sum64()
}

// makeNode, create new node with cidr.
func makeNode(pfx netip.Prefix, val any) *node {
	n := new(node)
	n.cidr = pfx.Masked() // always store the prefix in normalized form
	n.value = val
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

// ipTooBig returns true if ip is greater than prefix last address.
func ipTooBig(ip netip.Addr, p netip.Prefix) bool {
	_, pLast := extnetip.Range(p)
	return ip.Compare(pLast) > 0
}

// pfxTooBig returns true if k last address is greater than p last address.
func pfxTooBig(k netip.Prefix, p netip.Prefix) bool {
	_, ip := extnetip.Range(k)
	return ipTooBig(ip, p)
}
