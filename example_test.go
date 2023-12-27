package cidrtree_test

import (
	"fmt"
	"net/netip"
	"os"

	"github.com/gaissmai/cidrtree"
)

var input = []netip.Prefix{
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("10.0.0.0/24"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("::/0"),
	netip.MustParsePrefix("10.0.1.0/24"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("2000::/3"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("127.0.0.1/32"),
	netip.MustParsePrefix("192.168.1.0/24"),
}

func ExampleTable_Fprint() {
	rtbl := new(cidrtree.Table)
	for _, cidr := range input {
		rtbl.InsertMutable(cidr, nil)
	}
	rtbl.Fprint(os.Stdout)

	// Output:
	// ▼
	// ├─ 10.0.0.0/8 (<nil>)
	// │  ├─ 10.0.0.0/24 (<nil>)
	// │  └─ 10.0.1.0/24 (<nil>)
	// ├─ 127.0.0.0/8 (<nil>)
	// │  └─ 127.0.0.1/32 (<nil>)
	// ├─ 169.254.0.0/16 (<nil>)
	// ├─ 172.16.0.0/12 (<nil>)
	// └─ 192.168.0.0/16 (<nil>)
	//    └─ 192.168.1.0/24 (<nil>)
	// ▼
	// └─ ::/0 (<nil>)
	//    ├─ ::1/128 (<nil>)
	//    ├─ 2000::/3 (<nil>)
	//    │  └─ 2001:db8::/32 (<nil>)
	//    └─ fe80::/10 (<nil>)
}

func ExampleTable_Walk() {
	cb := func(p netip.Prefix, val any) bool {
		fmt.Printf("%v (%v)\n", p, val)
		return true
	}

	rtbl := new(cidrtree.Table)
	for _, cidr := range input {
		rtbl.InsertMutable(cidr, nil)
	}
	rtbl.Walk(cb)

	// Output:
	// 10.0.0.0/8 (<nil>)
	// 10.0.0.0/24 (<nil>)
	// 10.0.1.0/24 (<nil>)
	// 127.0.0.0/8 (<nil>)
	// 127.0.0.1/32 (<nil>)
	// 169.254.0.0/16 (<nil>)
	// 172.16.0.0/12 (<nil>)
	// 192.168.0.0/16 (<nil>)
	// 192.168.1.0/24 (<nil>)
	// ::/0 (<nil>)
	// ::1/128 (<nil>)
	// 2000::/3 (<nil>)
	// 2001:db8::/32 (<nil>)
	// fe80::/10 (<nil>)
}
