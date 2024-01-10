package cidrtree

import (
	"fmt"
	"io"
	"math"
	"net/netip"
)

// fprintBST writes a horizontal tree diagram of the binary search tree (BST) to w.
//
// Note: This is for debugging purposes only.
func (t Table[V]) fprintBST(w io.Writer) error {
	if t.root4 != nil {
		if _, err := fmt.Fprint(w, "R "); err != nil {
			return err
		}
		if err := t.root4.fprintBST(w, ""); err != nil {
			return err
		}
	}

	if t.root6 != nil {
		if _, err := fmt.Fprint(w, "R "); err != nil {
			return err
		}
		if err := t.root6.fprintBST(w, ""); err != nil {
			return err
		}
	}

	return nil
}

// fprintBST recursive helper.
func (n *node[V]) fprintBST(w io.Writer, pad string) error {
	// stringify this node
	_, err := fmt.Fprintf(w, "%v [prio:%.4g] [subtree maxUpper: %v]\n", n.cidr, float64(n.prio)/math.MaxUint64, n.maxUpper.cidr)
	if err != nil {
		return err
	}

	// prepare glyphe, spacer and padding for next level
	var glyphe string
	var spacer string

	// left wing
	if n.left != nil {
		if n.right != nil {
			glyphe = "├─l "
			spacer = "│   "
		} else {
			glyphe = "└─l "
			spacer = "    "
		}
		if _, err := fmt.Fprint(w, pad+glyphe); err != nil {
			return err
		}
		if err := n.left.fprintBST(w, pad+spacer); err != nil {
			return err
		}
	}

	// right wing
	if n.right != nil {
		glyphe = "└─r "
		spacer = "    "
		if _, err := fmt.Fprint(w, pad+glyphe); err != nil {
			return err
		}
		if err := n.right.fprintBST(w, pad+spacer); err != nil {
			return err
		}
	}

	return nil
}

// statistics, returns the maxDepth, average and standard deviation of the nodes.
// If the skip function is not nil, a true return value defines which nodes must be skipped in the statistics.
//
// Note: This is for debugging and testing purposes only during development.
func (t Table[V]) statistics(skip func(netip.Prefix, any, int) bool) (size int, maxDepth int, average, deviation float64) {
	// key is depth, value is the sum of nodes with this depth
	depths := make(map[int]int)

	// closure callback, get the depths, sum up the size
	cb := func(pfx netip.Prefix, val any, depth int) bool {
		if skip != nil && skip(pfx, val, depth) {
			return true
		}

		depths[depth] += 1
		size += 1
		return true
	}

	t.root4.walkWithDepth(cb, 0)
	t.root6.walkWithDepth(cb, 0)

	var weightedSum, sum int
	for k, v := range depths {
		weightedSum += k * v
		sum += v
		if k > maxDepth {
			maxDepth = k
		}
	}

	average = float64(weightedSum) / float64(sum)

	var variance float64
	for k := range depths {
		variance += math.Pow(float64(k)-average, 2.0)
	}
	variance = variance / float64(sum)
	deviation = math.Sqrt(variance)

	return size, maxDepth, average, deviation
}

// walkWithDepth in ascending prefix order.
func (n *node[V]) walkWithDepth(cb func(netip.Prefix, any, int) bool, depth int) bool {
	if n == nil {
		return true
	}

	// left
	if !n.left.walkWithDepth(cb, depth+1) {
		return false
	}

	// do-it
	if !cb(n.cidr, n.value, depth) {
		return false
	}

	// right
	if !n.right.walkWithDepth(cb, depth+1) {
		return false
	}

	return true
}
