package cidrtree_test

import (
	"net/netip"
	"reflect"
	"strings"
	"testing"

	"github.com/gaissmai/cidrtree"
)

var cidrStrings = []string{
	"fe80::/10",
	"172.16.0.0/12",
	"10.0.0.0/24",
	"::1/128",
	"192.168.0.0/16",
	"10.0.0.0/8",
	"::/0",
	"10.0.1.0/24",
	"169.254.0.0/16",
	"2000::/3",
	"2001:db8::/32",
	"127.0.0.0/8",
	"127.0.0.1/32",
	"fc00::/7",
	"192.168.1.0/24",
	"ff00::/8",
}

var cidrs = makeCIDRs(cidrStrings...)

func makeCIDRs(s ...string) (cidrs []netip.Prefix) {
	for _, cidrString := range s {
		cidrs = append(cidrs, netip.MustParsePrefix(cidrString))
	}
	return cidrs
}

const asStr = `▼
├─ 10.0.0.0/8
│  ├─ 10.0.0.0/24
│  └─ 10.0.1.0/24
├─ 127.0.0.0/8
│  └─ 127.0.0.1/32
├─ 169.254.0.0/16
├─ 172.16.0.0/12
└─ 192.168.0.0/16
   └─ 192.168.1.0/24
▼
└─ ::/0
   ├─ ::1/128
   ├─ 2000::/3
   │  └─ 2001:db8::/32
   ├─ fc00::/7
   ├─ fe80::/10
   └─ ff00::/8
`

func TestZeroValue(t *testing.T) {
	t.Parallel()

	var zeroIP netip.Addr
	var zeroCIDR netip.Prefix
	var zeroTree cidrtree.Tree

	if zeroTree.String() != "" {
		t.Errorf("String() = %v, want \"\"", "")
	}

	w := new(strings.Builder)
	if err := zeroTree.Fprint(w); err != nil {
		t.Fatal(err)
	}

	if w.String() != "" {
		t.Errorf("Fprint(w) = %v, want \"\"", w.String())
	}

	if w.String() != "" {
		t.Errorf("FprintBST(w) = %v, want \"\"", w.String())
	}

	if _, ok := zeroTree.Delete(zeroCIDR); ok {
		t.Errorf("Delete(), got: %v, want: false", ok)
	}

	if _, ok := zeroTree.Lookup(zeroIP); ok {
		t.Errorf("Lookup(), got: %v, want: false", ok)
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	_ = cidrtree.New()

	tree := cidrtree.New(cidrs...)

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}
}

func TestNewConcurrent(t *testing.T) {
	t.Parallel()
	cidrs := sliceCIDRs(100_000)

	// test zero
	tree1 := cidrtree.New()
	tree2 := cidrtree.NewConcurrent(0)

	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("New() differs with NewConcurrent()")
	}

	tree1 = cidrtree.New(cidrs[0])
	tree2 = cidrtree.NewConcurrent(1, cidrs[0])

	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("New() differs with NewConcurrent()")
	}

	tree1 = cidrtree.New(cidrs[:2]...)
	tree2 = cidrtree.NewConcurrent(2, cidrs[:2]...)

	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("New() differs with NewConcurrent()")
	}

	tree1 = cidrtree.New(cidrs[:30_000]...)
	tree2 = cidrtree.NewConcurrent(3, cidrs[:30_000]...)

	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("New() differs with NewConcurrent()")
	}

	tree1 = cidrtree.New(cidrs...)
	tree2 = cidrtree.NewConcurrent(4, cidrs...)

	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("New() differs with NewConcurrent()")
	}
}

func TestInsert(t *testing.T) {
	t.Parallel()
	tree := cidrtree.New()

	for _, cidr := range cidrs {
		tree = tree.Insert(cidr)
	}

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}
}

func TestDupInsert(t *testing.T) {
	t.Parallel()
	tree := cidrtree.New()

	for _, cidr := range cidrs {
		tree = tree.Insert(cidr)
	}

	for _, dupe := range cidrs {
		tree = tree.Insert(dupe)
	}

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}

	for _, dupe := range cidrs {
		tree.InsertMutable(dupe)
	}

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}
}

func TestInsertMutable(t *testing.T) {
	t.Parallel()
	tree := cidrtree.New()

	for _, cidr := range cidrs {
		tree.InsertMutable(cidr)
	}

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}
}

func TestImmutable(t *testing.T) {
	t.Parallel()

	tree1 := cidrtree.New(cidrs...)
	tree2 := tree1.Clone()

	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("cloned tree is not deep equal to original")
	}

	probe := cidrs[0]
	if _, ok := tree1.Delete(probe); !ok {
		t.Fatal("Delete, could not delete probe item")
	}
	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("Delete changed receiver")
	}

	probe = cidrs[len(cidrs)-1]
	_ = tree1.Insert(probe)
	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("Insert changed receiver")
	}

	ip := probe.Addr()
	_, _ = tree1.Lookup(ip)
	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("Lookup changed receiver")
	}
}

func TestMutable(t *testing.T) {
	tree1 := cidrtree.New(cidrs...)
	tree2 := tree1.Clone()

	probe := cidrs[0]

	var ok bool
	if ok = (&tree1).DeleteMutable(probe); !ok {
		t.Fatal("DeleteMutable, could not delete probe item")
	}

	if reflect.DeepEqual(tree1, tree2) {
		t.Fatal("DeleteMutable didn't change receiver")
	}

	// reset tree1, tree2
	tree1 = cidrtree.New(cidrs...)
	tree2 = tree1.Clone()

	probe = netip.MustParsePrefix("1.2.3.4/17")
	(&tree1).InsertMutable(probe)

	if reflect.DeepEqual(tree1, tree2) {
		t.Fatal("InsertMutable didn't change receiver")
	}

	if _, ok := tree1.Delete(probe); !ok {
		t.Fatal("InsertMutable didn't change receiver")
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	tree := cidrtree.New(cidrs...)

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}

	for _, cidr := range cidrs {
		var ok bool
		tree, ok = tree.Delete(cidr)
		if !ok {
			t.Fatalf("Delete(%v), got %v, want true", cidr, ok)
		}
	}

	if tree.String() != "" {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", "", tree.String())
	}
}

func TestDeleteMutable(t *testing.T) {
	t.Parallel()

	tree := cidrtree.New(cidrs...)

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}

	for _, cidr := range cidrs {
		if ok := tree.DeleteMutable(cidr); !ok {
			t.Fatalf("Delete(%v), got %v, want true", cidr, ok)
		}
	}

	if tree.String() != "" {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", "", tree.String())
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()

	tree := cidrtree.New(cidrs...)

	tcs := []struct {
		ip     netip.Addr
		want   netip.Prefix
		wantOK bool
	}{
		{
			ip:     netip.MustParseAddr("10.0.1.17"),
			want:   netip.MustParsePrefix("10.0.1.0/24"),
			wantOK: true,
		},
		{
			ip:     netip.MustParseAddr("10.2.3.4"),
			want:   netip.MustParsePrefix("10.0.0.0/8"),
			wantOK: true,
		},
		{
			ip:     netip.MustParseAddr("12.0.0.0"),
			want:   netip.Prefix{},
			wantOK: false,
		},
		{
			ip:     netip.MustParseAddr("127.0.0.255"),
			want:   netip.MustParsePrefix("127.0.0.0/8"),
			wantOK: true,
		},
		{
			ip:     netip.MustParseAddr("::2"),
			want:   netip.MustParsePrefix("::/0"),
			wantOK: true,
		},
		{
			ip:     netip.MustParseAddr("2001:db8:affe:cafe::dead:beef"),
			want:   netip.MustParsePrefix("2001:db8::/32"),
			wantOK: true,
		},
	}

	for _, tt := range tcs {
		if got, ok := tree.Lookup(tt.ip); ok != tt.wantOK || got != tt.want {
			t.Errorf("Lookup(%v) = (%v, %v),  want (%v, %v)", tt.ip, got, ok, tt.want, tt.wantOK)
		}
	}

	prefix := netip.MustParsePrefix("10.0.0.0/8")
	if ok := tree.DeleteMutable(prefix); !ok {
		t.Errorf("Delete(%v) = %v, want %v", prefix, ok, true)
	}

	ip := netip.MustParseAddr("1.2.3.4")
	want := netip.Prefix{}

	if got, ok := tree.Lookup(ip); ok == true || got != want {
		t.Errorf("Lookup(%v) = %v, %v, want %v, %v", ip, got, ok, want, false)
	}

	prefix = netip.MustParsePrefix("::/0")
	if ok := tree.DeleteMutable(prefix); !ok {
		t.Errorf("Delete(%v) = %v, want %v", prefix, ok, true)
	}

	ip = netip.MustParseAddr("::2")
	want = netip.Prefix{}

	if got, ok := tree.Lookup(ip); ok == true || got != want {
		t.Errorf("Lookup(%v) = %v, %v, want %v, %v", ip, got, ok, want, false)
	}

	// ##########################################

	tc := sliceCIDRs(100_000)
	tree = cidrtree.New(tc...)
	for _, cidr := range tc {
		ip := cidr.Addr()

		if _, ok := tree.Lookup(ip); !ok {
			t.Fatalf("Lookup(%v), want true, got %v", ip, ok)
		}

		ip = ip.Prev()
		match, ok := tree.Lookup(ip)
		if ok && match == cidr {
			t.Fatalf("Lookup(%v), match(%v) == cidr (%v), not allowed", ip, match, cidr)
		}
	}
}

func TestUnion(t *testing.T) {
	t.Parallel()
	tree := cidrtree.New(cidrs...)
	clone := tree.Clone()

	if !reflect.DeepEqual(tree, clone) {
		t.Fatal("Clone isn't deep equal to original tree.")
	}

	var tree2 cidrtree.Tree
	for _, cidr := range cidrs {
		tree2 = tree2.Union(cidrtree.New(cidr), false)
	}

	if !reflect.DeepEqual(tree, clone) {
		t.Fatal("tree2 isn't deep equal to original tree.")
	}

	// dupe union
	tree = tree.Union(tree2, true)

	if !reflect.DeepEqual(tree, clone) {
		t.Fatal("Clone isn't deep equal to original tree.")
	}
}

func TestFprint(t *testing.T) {
	t.Parallel()
	tree := cidrtree.New(cidrs...)

	w := new(strings.Builder)
	if err := tree.Fprint(w); err != nil {
		t.Fatal(err)
	}

	if w.String() != asStr {
		t.Fatal("Fprint, expected and got differs")
	}
}
