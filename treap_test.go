package cidrtree_test

import (
	"net/netip"
	"reflect"
	"strings"
	"testing"

	"github.com/gaissmai/cidrtree"
)

type routeT struct {
	cidr    string
	nexthop string
}

var routes = []routeT{
	{"fe80::/10", "2001:db8::1"},
	{"172.16.0.0/12", "203.0.113.0"},
	{"10.0.0.0/24", "203.0.113.0"},
	{"::1/128", "2001:db8::1"},
	{"192.168.0.0/16", "203.0.113.0"},
	{"10.0.0.0/8", "203.0.113.0"},
	{"::/0", "2001:db8::1"},
	{"10.0.1.0/24", "203.0.113.0"},
	{"169.254.0.0/16", "203.0.113.0"},
	{"2000::/3", "2001:db8::1"},
	{"2001:db8::/32", "2001:db8::1"},
	{"127.0.0.0/8", "203.0.113.0"},
	{"127.0.0.1/32", "203.0.113.0"},
	{"fc00::/7", "2001:db8::1"},
	{"192.168.1.0/24", "203.0.113.0"},
	{"ff00::/8", "2001:db8::1"},
}

var items = makeRoutes(routes)

func makeRoutes(rs []routeT) (items []cidrtree.KeyVal) {
	for _, route := range rs {
		items = append(items, cidrtree.KeyVal{netip.MustParsePrefix(route.cidr), netip.MustParseAddr(route.nexthop)})
	}
	return items
}

const asStr = `▼
├─ 10.0.0.0/8 (203.0.113.0)
│  ├─ 10.0.0.0/24 (203.0.113.0)
│  └─ 10.0.1.0/24 (203.0.113.0)
├─ 127.0.0.0/8 (203.0.113.0)
│  └─ 127.0.0.1/32 (203.0.113.0)
├─ 169.254.0.0/16 (203.0.113.0)
├─ 172.16.0.0/12 (203.0.113.0)
└─ 192.168.0.0/16 (203.0.113.0)
   └─ 192.168.1.0/24 (203.0.113.0)
▼
└─ ::/0 (2001:db8::1)
   ├─ ::1/128 (2001:db8::1)
   ├─ 2000::/3 (2001:db8::1)
   │  └─ 2001:db8::/32 (2001:db8::1)
   ├─ fc00::/7 (2001:db8::1)
   ├─ fe80::/10 (2001:db8::1)
   └─ ff00::/8 (2001:db8::1)
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

	if _, _, ok := zeroTree.Lookup(zeroIP); ok {
		t.Errorf("Lookup(), got: %v, want: false", ok)
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	_ = cidrtree.New()

	tree := cidrtree.New(items...)

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}
}

func TestNewConcurrent(t *testing.T) {
	t.Parallel()
	cidrs := sliceItems(100_000)

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

	for _, cidr := range items {
		tree = tree.Insert(cidr)
	}

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}
}

func TestDupInsert(t *testing.T) {
	t.Parallel()
	tree := cidrtree.New()

	for _, cidr := range items {
		tree = tree.Insert(cidr)
	}

	for _, dupe := range items {
		tree = tree.Insert(dupe)
	}

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}

	for _, dupe := range items {
		tree.InsertMutable(dupe)
	}

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}
}

func TestInsertMutable(t *testing.T) {
	t.Parallel()
	tree := cidrtree.New()

	for _, cidr := range items {
		tree.InsertMutable(cidr)
	}

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}
}

func TestImmutable(t *testing.T) {
	t.Parallel()

	tree1 := cidrtree.New(items...)
	tree2 := tree1.Clone()

	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("cloned tree is not deep equal to original")
	}

	probe := items[0]
	if _, ok := tree1.Delete(probe.CIDR); !ok {
		t.Fatal("Delete, could not delete probe item")
	}
	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("Delete changed receiver")
	}

	probe = items[len(items)-1]
	_ = tree1.Insert(probe)
	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("Insert changed receiver")
	}

	ip := probe.CIDR.Addr()
	_, _, _ = tree1.Lookup(ip)
	if !reflect.DeepEqual(tree1, tree2) {
		t.Fatal("Lookup changed receiver")
	}
}

func TestMutable(t *testing.T) {
	tree1 := cidrtree.New(items...)
	tree2 := tree1.Clone()

	probe := items[0]

	var ok bool
	if ok = (&tree1).DeleteMutable(probe.CIDR); !ok {
		t.Fatal("DeleteMutable, could not delete probe item")
	}

	if reflect.DeepEqual(tree1, tree2) {
		t.Fatal("DeleteMutable didn't change receiver")
	}

	// reset tree1, tree2
	tree1 = cidrtree.New(items...)
	tree2 = tree1.Clone()

	probe = cidrtree.KeyVal{netip.MustParsePrefix("1.2.3.4/17"), nil}
	(&tree1).InsertMutable(probe)

	if reflect.DeepEqual(tree1, tree2) {
		t.Fatal("InsertMutable didn't change receiver")
	}

	if _, ok := tree1.Delete(probe.CIDR); !ok {
		t.Fatal("InsertMutable didn't change receiver")
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	tree := cidrtree.New(items...)

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}

	for _, item := range items {
		var ok bool
		tree, ok = tree.Delete(item.CIDR)
		if !ok {
			t.Fatalf("Delete(%v), got %v, want true", item.CIDR, ok)
		}
	}

	if tree.String() != "" {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", "", tree.String())
	}
}

func TestDeleteMutable(t *testing.T) {
	t.Parallel()

	tree := cidrtree.New(items...)

	if tree.String() != asStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asStr, tree.String())
	}

	for _, item := range items {
		if ok := tree.DeleteMutable(item.CIDR); !ok {
			t.Fatalf("Delete(%v), got %v, want true", item.CIDR, ok)
		}
	}

	if tree.String() != "" {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", "", tree.String())
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()

	tree := cidrtree.New(items...)

	tcs := []struct {
		ip     netip.Addr
		want   netip.Prefix
		want2  netip.Addr
		wantOK bool
	}{
		{
			ip:     netip.MustParseAddr("10.0.1.17"),
			want:   netip.MustParsePrefix("10.0.1.0/24"),
			want2:  netip.MustParseAddr("203.0.113.0"),
			wantOK: true,
		},
		{
			ip:     netip.MustParseAddr("10.2.3.4"),
			want:   netip.MustParsePrefix("10.0.0.0/8"),
			want2:  netip.MustParseAddr("203.0.113.0"),
			wantOK: true,
		},
		{
			ip:     netip.MustParseAddr("12.0.0.0"),
			want:   netip.Prefix{},
			want2:  netip.Addr{},
			wantOK: false,
		},
		{
			ip:     netip.MustParseAddr("127.0.0.255"),
			want:   netip.MustParsePrefix("127.0.0.0/8"),
			want2:  netip.MustParseAddr("203.0.113.0"),
			wantOK: true,
		},
		{
			ip:     netip.MustParseAddr("::2"),
			want:   netip.MustParsePrefix("::/0"),
			want2:  netip.MustParseAddr("2001:db8::1"),
			wantOK: true,
		},
		{
			ip:     netip.MustParseAddr("2001:db8:affe:cafe::dead:beef"),
			want:   netip.MustParsePrefix("2001:db8::/32"),
			want2:  netip.MustParseAddr("2001:db8::1"),
			wantOK: true,
		},
	}

	for _, tt := range tcs {
		if got, got2, ok := tree.Lookup(tt.ip); ok != tt.wantOK || got != tt.want {
			t.Errorf("Lookup(%v) = (%v, %v, %v),  want (%v, %v, %v)", tt.ip, got, got2, ok, tt.want, tt.want2, tt.wantOK)
		}
	}

	prefix := netip.MustParsePrefix("10.0.0.0/8")
	if ok := tree.DeleteMutable(prefix); !ok {
		t.Errorf("Delete(%v) = %v, want %v", prefix, ok, true)
	}

	ip := netip.MustParseAddr("1.2.3.4")
	want := netip.Prefix{}
	want2 := any(nil)

	if got, got2, ok := tree.Lookup(ip); ok == true || got != want || got2 != want2 {
		t.Errorf("Lookup(%v) = %v, %v, %v, want %v, %v, %v", ip, got, got2, ok, want, want2, false)
	}

	prefix = netip.MustParsePrefix("::/0")
	if ok := tree.DeleteMutable(prefix); !ok {
		t.Errorf("Delete(%v) = %v, want %v", prefix, ok, true)
	}

	ip = netip.MustParseAddr("::2")
	want = netip.Prefix{}
	want2 = any(nil)

	if got, got2, ok := tree.Lookup(ip); ok == true || got != want || got2 != want2 {
		t.Errorf("Lookup(%v) = %v, %v, %v, want %v, %v, %v", ip, got, got2, ok, want, want2, false)
	}

	// ##########################################

	tc := sliceItems(100_000)
	tree = cidrtree.New(tc...)
	for _, item := range tc {
		ip := item.CIDR.Addr()

		if _, _, ok := tree.Lookup(ip); !ok {
			t.Fatalf("Lookup(%v), want true, got %v", ip, ok)
		}

		ip = ip.Prev()
		match, _, ok := tree.Lookup(ip)
		if ok && match == item.CIDR {
			t.Fatalf("Lookup(%v), match(%v) == cidr (%v), not allowed", ip, match, item.CIDR)
		}
	}
}

func TestUnion(t *testing.T) {
	t.Parallel()
	tree := cidrtree.New(items...)
	clone := tree.Clone()

	if !reflect.DeepEqual(tree, clone) {
		t.Fatal("Clone isn't deep equal to original tree.")
	}

	var tree2 cidrtree.Tree
	for _, cidr := range items {
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
	tree := cidrtree.New(items...)

	w := new(strings.Builder)
	if err := tree.Fprint(w); err != nil {
		t.Fatal(err)
	}

	if w.String() != asStr {
		t.Fatal("Fprint, expected and got differs")
	}
}
