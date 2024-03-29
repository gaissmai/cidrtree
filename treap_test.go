package cidrtree_test

import (
	"fmt"
	"net/netip"
	"reflect"
	"strings"
	"testing"

	"github.com/gaissmai/cidrtree"
)

type routeStr struct {
	cidr    string
	nextHop string
}

var routesStr = []routeStr{
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

type route struct {
	cidr    netip.Prefix
	nextHop netip.Addr
}

var routes = makeRoutes(routesStr)

func makeRoutes(rs []routeStr) []route {
	var routes []route
	for _, s := range rs {
		routes = append(routes, route{mustPfx(s.cidr), mustAddr(s.nextHop)})
	}
	return routes
}

const asStr = `10.0.0.0/8 (203.0.113.0)
10.0.0.0/24 (203.0.113.0)
10.0.1.0/24 (203.0.113.0)
127.0.0.0/8 (203.0.113.0)
127.0.0.1/32 (203.0.113.0)
169.254.0.0/16 (203.0.113.0)
172.16.0.0/12 (203.0.113.0)
192.168.0.0/16 (203.0.113.0)
192.168.1.0/24 (203.0.113.0)
::/0 (2001:db8::1)
::1/128 (2001:db8::1)
2000::/3 (2001:db8::1)
2001:db8::/32 (2001:db8::1)
fc00::/7 (2001:db8::1)
fe80::/10 (2001:db8::1)
ff00::/8 (2001:db8::1)
`

const asTopoStr = `▼
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
	var zeroTable cidrtree.Table[any]

	if zeroTable.String() != "" {
		t.Errorf("String() = %v, want \"\"", "")
	}

	w := new(strings.Builder)
	if err := zeroTable.Fprint(w); err != nil {
		t.Fatal(err)
	}

	if w.String() != "" {
		t.Errorf("Fprint(w) = %v, want \"\"", w.String())
	}

	// must not panic
	zeroTable.Walk(func(netip.Prefix, any) bool { return true })

	if _, ok := zeroTable.DeleteImmutable(zeroCIDR); ok {
		t.Errorf("Delete(), got: %v, want: false", ok)
	}

	if _, _, ok := zeroTable.Lookup(zeroIP); ok {
		t.Errorf("Lookup(), got: %v, want: false", ok)
	}

	if _, _, ok := zeroTable.LookupPrefix(zeroCIDR); ok {
		t.Errorf("LookupCIDR(), got: %v, want: false", ok)
	}

	if rtbl := zeroTable.UnionImmutable(zeroTable); *rtbl != zeroTable {
		t.Errorf("Union(), got: %#v, want: %#v", rtbl, &zeroTable)
	}
}

func TestInsertImmutable(t *testing.T) {
	t.Parallel()
	rtbl := new(cidrtree.Table[any])

	for _, route := range routes {
		rtbl = rtbl.InsertImmutable(route.cidr, route.nextHop)
	}

	if rtbl.String() != asTopoStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asTopoStr, rtbl.String())
	}
}

func TestDupInsert(t *testing.T) {
	t.Parallel()
	rtbl := new(cidrtree.Table[any])

	for _, route := range routes {
		rtbl.Insert(route.cidr, route.nextHop)
	}

	for _, dupe := range routes {
		rtbl = rtbl.InsertImmutable(dupe.cidr, dupe.nextHop)
	}

	if rtbl.String() != asTopoStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asTopoStr, rtbl.String())
	}

	for _, dupe := range routes {
		rtbl.Insert(dupe.cidr, dupe.nextHop)
	}

	if rtbl.String() != asTopoStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asTopoStr, rtbl.String())
	}

	cidr := routes[0].cidr
	_, _, ok := rtbl.LookupPrefix(routes[0].cidr)
	if !ok {
		t.Errorf("LookupCIDR(%v), expect %v, got %v", routes[0].cidr, true, ok)
	}
	// overwrite value for this cidr
	rtbl.Insert(cidr, "overwrite value")

	_, value, ok := rtbl.LookupPrefix(cidr)
	if !ok {
		t.Errorf("LookupCIDR(%v), expect %v, got %v", routes[0].cidr, true, ok)
	}
	if value != "overwrite value" {
		t.Errorf("InsertMutable duplicate, expect %q, got %q", "overwrite value", value)
	}
}

func TestInsert(t *testing.T) {
	t.Parallel()
	rtbl := new(cidrtree.Table[any])

	for _, route := range routes {
		rtbl.Insert(route.cidr, route.nextHop)
	}

	if rtbl.String() != asTopoStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asTopoStr, rtbl.String())
	}
}

func TestImmutable(t *testing.T) {
	t.Parallel()

	rtbl1 := new(cidrtree.Table[any])
	for _, route := range routes {
		rtbl1.Insert(route.cidr, route.nextHop)
	}
	rtbl2 := rtbl1.Clone()

	if !reflect.DeepEqual(rtbl1, rtbl2) {
		t.Fatal("cloned table is not deep equal to original")
	}

	probe := routes[0]
	if _, ok := rtbl1.DeleteImmutable(probe.cidr); !ok {
		t.Fatal("Delete, could not delete probe item")
	}
	if !reflect.DeepEqual(rtbl1, rtbl2) {
		t.Fatal("Delete changed receiver")
	}

	probe = routes[len(routes)-1]
	_ = rtbl1.InsertImmutable(probe.cidr, probe.nextHop)
	if !reflect.DeepEqual(rtbl1, rtbl2) {
		t.Fatal("Insert changed receiver")
	}

	ip := probe.cidr.Addr()
	_, _, _ = rtbl1.Lookup(ip)
	if !reflect.DeepEqual(rtbl1, rtbl2) {
		t.Fatal("Lookup changed receiver")
	}

	cidr := probe.cidr
	_, _, _ = rtbl1.LookupPrefix(cidr)
	if !reflect.DeepEqual(rtbl1, rtbl2) {
		t.Fatal("LookupCIDR changed receiver")
	}
}

func TestMutable(t *testing.T) {
	rtbl1 := new(cidrtree.Table[any])
	for _, route := range routes {
		rtbl1.Insert(route.cidr, route.nextHop)
	}
	rtbl2 := rtbl1.Clone()

	probe := routes[0]

	var ok bool
	if ok = rtbl1.Delete(probe.cidr); !ok {
		t.Fatal("DeleteMutable, could not delete probe item")
	}

	if reflect.DeepEqual(rtbl1, rtbl2) {
		t.Fatal("DeleteMutable didn't change receiver")
	}

	// reset table1, table2
	rtbl1 = new(cidrtree.Table[any])
	for _, route := range routes {
		rtbl1.Insert(route.cidr, route.nextHop)
	}
	rtbl2 = rtbl1.Clone()

	probe = route{cidr: mustPfx("1.2.3.4/17")}
	rtbl1.Insert(probe.cidr, probe.nextHop)

	if reflect.DeepEqual(rtbl1, rtbl2) {
		t.Fatal("InsertMutable didn't change receiver")
	}

	if _, ok := rtbl1.DeleteImmutable(probe.cidr); !ok {
		t.Fatal("InsertMutable didn't change receiver")
	}
}

func TestDeleteImmutable(t *testing.T) {
	t.Parallel()

	rtbl := new(cidrtree.Table[any])
	for _, route := range routes {
		rtbl.Insert(route.cidr, route.nextHop)
	}

	if rtbl.String() != asTopoStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asTopoStr, rtbl.String())
	}

	for _, route := range routes {
		var ok bool
		rtbl, ok = rtbl.DeleteImmutable(route.cidr)
		if !ok {
			t.Fatalf("Delete(%v), got %v, want true", route.cidr, ok)
		}
	}

	if rtbl.String() != "" {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", "", rtbl.String())
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	rtbl := new(cidrtree.Table[any])
	for _, route := range routes {
		rtbl.Insert(route.cidr, route.nextHop)
	}

	if rtbl.String() != asTopoStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asTopoStr, rtbl.String())
	}

	for _, route := range routes {
		if ok := rtbl.Delete(route.cidr); !ok {
			t.Fatalf("Delete(%v), got %v, want true", route.cidr, ok)
		}
	}

	if rtbl.String() != "" {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", "", rtbl.String())
	}
}

func TestLookupIP(t *testing.T) {
	t.Parallel()

	rtbl := new(cidrtree.Table[any])
	for _, route := range routes {
		rtbl.Insert(route.cidr, route.nextHop)
	}

	tcs := []struct {
		ip     netip.Addr
		want   netip.Prefix
		want2  netip.Addr
		wantOK bool
	}{
		{
			ip:     mustAddr("10.0.1.17"),
			want:   mustPfx("10.0.1.0/24"),
			want2:  mustAddr("203.0.113.0"),
			wantOK: true,
		},
		{
			ip:     mustAddr("10.2.3.4"),
			want:   mustPfx("10.0.0.0/8"),
			want2:  mustAddr("203.0.113.0"),
			wantOK: true,
		},
		{
			ip:     mustAddr("12.0.0.0"),
			want:   netip.Prefix{},
			want2:  netip.Addr{},
			wantOK: false,
		},
		{
			ip:     mustAddr("127.0.0.255"),
			want:   mustPfx("127.0.0.0/8"),
			want2:  mustAddr("203.0.113.0"),
			wantOK: true,
		},
		{
			ip:     mustAddr("::2"),
			want:   mustPfx("::/0"),
			want2:  mustAddr("2001:db8::1"),
			wantOK: true,
		},
		{
			ip:     mustAddr("2001:db8:affe:cafe::dead:beef"),
			want:   mustPfx("2001:db8::/32"),
			want2:  mustAddr("2001:db8::1"),
			wantOK: true,
		},
	}

	for _, tt := range tcs {
		if got, got2, ok := rtbl.Lookup(tt.ip); ok != tt.wantOK || got != tt.want {
			t.Errorf("Lookup(%v) = (%v, %v, %v),  want (%v, %v, %v)", tt.ip, got, got2, ok, tt.want, tt.want2, tt.wantOK)
		}
	}

	prefix := mustPfx("10.0.0.0/8")
	if ok := rtbl.Delete(prefix); !ok {
		t.Errorf("Delete(%v) = %v, want %v", prefix, ok, true)
	}

	ip := mustAddr("1.2.3.4")
	want := netip.Prefix{}
	want2 := any(nil)

	if got, got2, ok := rtbl.Lookup(ip); ok == true || got != want || got2 != want2 {
		t.Errorf("Lookup(%v) = %v, %v, %v, want %v, %v, %v", ip, got, got2, ok, want, want2, false)
	}

	prefix = mustPfx("::/0")
	if ok := rtbl.Delete(prefix); !ok {
		t.Errorf("Delete(%v) = %v, want %v", prefix, ok, true)
	}

	ip = mustAddr("::2")
	want = netip.Prefix{}
	want2 = any(nil)

	if got, got2, ok := rtbl.Lookup(ip); ok == true || got != want || got2 != want2 {
		t.Errorf("Lookup(%v) = %v, %v, %v, want %v, %v, %v", ip, got, got2, ok, want, want2, false)
	}

	// ##########################################

	tc := shuffleFullTable(100_000)
	rtbl2 := new(cidrtree.Table[any])
	for _, cidr := range tc {
		rtbl2.Insert(cidr, nil)
	}
	for _, cidr := range tc {
		ip := cidr.Addr()

		if _, _, ok := rtbl2.Lookup(ip); !ok {
			t.Fatalf("Lookup(%v), want true, got %v", ip, ok)
		}

		ip = ip.Prev()
		match, _, ok := rtbl2.Lookup(ip)
		if ok && match == cidr {
			t.Fatalf("Lookup(%v), match(%v) == cidr (%v), not allowed", ip, match, cidr)
		}
	}
}

func TestLookupCIDR(t *testing.T) {
	t.Parallel()

	rtbl := new(cidrtree.Table[any])
	for _, route := range routes {
		rtbl.Insert(route.cidr, route.nextHop)
	}

	tcs := []struct {
		cidr      netip.Prefix
		wantCIDR  netip.Prefix
		wantValue netip.Addr
		wantOK    bool
	}{
		{
			cidr:      mustPfx("10.0.1.0/29"),
			wantCIDR:  mustPfx("10.0.1.0/24"),
			wantValue: mustAddr("203.0.113.0"),
			wantOK:    true,
		},
		{
			cidr:      mustPfx("10.2.0.0/16"),
			wantCIDR:  mustPfx("10.0.0.0/8"),
			wantValue: mustAddr("203.0.113.0"),
			wantOK:    true,
		},
		{
			cidr:      mustPfx("12.0.0.0/8"),
			wantCIDR:  netip.Prefix{},
			wantValue: netip.Addr{},
			wantOK:    false,
		},
		{
			cidr:      mustPfx("127.0.0.2/32"),
			wantCIDR:  mustPfx("127.0.0.0/8"),
			wantValue: mustAddr("203.0.113.0"),
			wantOK:    true,
		},
		{
			cidr:      mustPfx("::2/96"),
			wantCIDR:  mustPfx("::/0"),
			wantValue: mustAddr("2001:db8::1"),
			wantOK:    true,
		},
		{
			cidr:      mustPfx("2001:db8:affe:cafe:dead:beef::/96"),
			wantCIDR:  mustPfx("2001:db8::/32"),
			wantValue: mustAddr("2001:db8::1"),
			wantOK:    true,
		},
	}

	for _, tt := range tcs {
		if got, got2, ok := rtbl.LookupPrefix(tt.cidr); ok != tt.wantOK || got != tt.wantCIDR {
			t.Errorf("LookupCIDR(%v) = (%v, %v, %v),  want (%v, %v, %v)", tt.cidr, got, got2, ok, tt.wantCIDR, tt.wantValue, tt.wantOK)
		}
	}

	prefix := mustPfx("10.0.0.0/8")
	if ok := rtbl.Delete(prefix); !ok {
		t.Errorf("Delete(%v) = %v, want %v", prefix, ok, true)
	}

	cidr := mustPfx("10.2.0.0/16")
	wantCIDR := netip.Prefix{}
	wantValue := any(nil)

	if got, got2, ok := rtbl.LookupPrefix(cidr); ok == true || got != wantCIDR || got2 != wantValue {
		t.Errorf("LookupCIDR(%v) = %v, %v, %v, want %v, %v, %v", cidr, got, got2, ok, wantCIDR, wantValue, false)
	}

	prefix = mustPfx("::/0")
	if ok := rtbl.Delete(prefix); !ok {
		t.Errorf("Delete(%v) = %v, want %v", prefix, ok, true)
	}

	cidr = mustPfx("::2/96")
	wantCIDR = netip.Prefix{}
	wantValue = any(nil)

	if got, got2, ok := rtbl.LookupPrefix(cidr); ok == true || got != wantCIDR || got2 != wantValue {
		t.Errorf("LookupCIDR(%v) = %v, %v, %v, want %v, %v, %v", cidr, got, got2, ok, wantCIDR, wantValue, false)
	}

	// ##########################################

	tc := shuffleFullTable(100_000)

	rtbl2 := new(cidrtree.Table[any])
	for _, cidr := range tc {
		rtbl2.Insert(cidr, nil)
	}
	for _, cidr := range tc {
		if _, _, ok := rtbl2.LookupPrefix(cidr); !ok {
			t.Fatalf("LookupCIDR(%v), want true, got %v", cidr, ok)
		}
	}
}

func TestUnion(t *testing.T) {
	t.Parallel()
	rtbl := new(cidrtree.Table[any])
	rtbl2 := new(cidrtree.Table[any])
	for _, route := range routes {
		rtbl.Insert(route.cidr, route.nextHop)
		rtbl2.Insert(route.cidr, route.nextHop)
	}

	rtbl.Union(*rtbl2)
	if rtbl.String() != asTopoStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asTopoStr, rtbl.String())
	}

	clone := rtbl.Clone()
	rtbl.Union(cidrtree.Table[any]{})
	if !reflect.DeepEqual(rtbl, clone) {
		t.Fatal("UnionMutable with zero value changed original")
	}

	rtbl3 := rtbl.UnionImmutable(*rtbl2)
	if rtbl3.String() != asTopoStr {
		t.Errorf("Fprint()\nwant:\n%sgot:\n%s", asTopoStr, rtbl.String())
	}
}

func TestUnionDupe(t *testing.T) {
	t.Parallel()
	rtbl1 := new(cidrtree.Table[any])
	rtbl2 := new(cidrtree.Table[any])
	for _, cidr := range shuffleFullTable(100_000) {
		rtbl1.Insert(cidr, 1)
		// dupe cidr with different value
		rtbl2.Insert(cidr, 2)
	}
	// both tables have identical CIDRs but with different values
	// overwrite all values with value=2
	rtbl1.Union(*rtbl2)

	var wrongValue bool

	// callback as closure
	cb := func(pfx netip.Prefix, val any) bool {
		if v, ok := val.(int); ok && v != 2 {
			wrongValue = true
			return false
		}
		return true
	}

	rtbl1.Walk(cb)
	if wrongValue {
		t.Error("Union with duplicate CIDRs didn't overwrite")
	}
}

func TestFprint(t *testing.T) {
	t.Parallel()
	rtbl := new(cidrtree.Table[any])
	for _, route := range routes {
		rtbl.Insert(route.cidr, route.nextHop)
	}

	w := new(strings.Builder)
	if err := rtbl.Fprint(w); err != nil {
		t.Fatal(err)
	}

	if w.String() != asTopoStr {
		t.Errorf("Fprint, not as expected, got:\n%s", w.String())
	}
}

func TestWalk(t *testing.T) {
	t.Parallel()
	rtbl := new(cidrtree.Table[any])
	for _, route := range routes {
		rtbl.Insert(route.cidr, route.nextHop)
	}
	w := new(strings.Builder)

	cb := func(pfx netip.Prefix, val any) bool {
		fmt.Fprintf(w, "%v (%v)\n", pfx, val)
		return true
	}

	rtbl.Walk(cb)
	if w.String() != asStr {
		t.Fatalf("Walk, expected:\n%sgot:\n%s", asStr, w.String())
	}
}

func TestWalkStartStop(t *testing.T) {
	t.Parallel()
	rtbl := new(cidrtree.Table[any])
	for _, route := range routes {
		rtbl.Insert(route.cidr, route.nextHop)
	}
	w := new(strings.Builder)

	cb := func(pfx netip.Prefix, val any) bool {
		if pfx.Addr().Is4() {
			// skip
			return true
		}
		if pfx == mustPfx("fc00::/7") {
			// stop
			return false
		}

		fmt.Fprintf(w, "%v (%v)\n", pfx, val)
		return true
	}

	rtbl.Walk(cb)

	expect := `::/0 (2001:db8::1)
::1/128 (2001:db8::1)
2000::/3 (2001:db8::1)
2001:db8::/32 (2001:db8::1)
`

	if w.String() != expect {
		t.Fatalf("Walk, expected:\n%sgot:\n%s", expect, w.String())
	}
}
