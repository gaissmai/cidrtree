package cidrtree_test

import (
	"fmt"
	"net/netip"
	"os"

	"github.com/gaissmai/cidrtree"
)

func addr(s string) netip.Addr {
	return netip.MustParseAddr(s)
}

func prfx(s string) netip.Prefix {
	return netip.MustParsePrefix(s)
}

var input = []netip.Prefix{
	prfx("fe80::/10"),
	prfx("172.16.0.0/12"),
	prfx("10.0.0.0/24"),
	prfx("::1/128"),
	prfx("192.168.0.0/16"),
	prfx("10.0.0.0/8"),
	prfx("::/0"),
	prfx("10.0.1.0/24"),
	prfx("169.254.0.0/16"),
	prfx("2000::/3"),
	prfx("2001:db8::/32"),
	prfx("127.0.0.0/8"),
	prfx("127.0.0.1/32"),
	prfx("192.168.1.0/24"),
}

func ExampleTable_Lookup() {
	rtbl := new(cidrtree.Table)
	for _, cidr := range input {
		rtbl.Insert(cidr, nil)
	}
	rtbl.Fprint(os.Stdout)

	fmt.Println()

	ip := addr("42.0.0.0")
	lpm, value, ok := rtbl.Lookup(ip)
	fmt.Printf("Lookup: %-20v lpm: %-15v value: %v, ok: %v\n", ip, lpm, value, ok)

	ip = addr("10.0.1.17")
	lpm, value, ok = rtbl.Lookup(ip)
	fmt.Printf("Lookup: %-20v lpm: %-15v value: %v, ok: %v\n", ip, lpm, value, ok)

	ip = addr("2001:7c0:3100:1::111")
	lpm, value, ok = rtbl.Lookup(ip)
	fmt.Printf("Lookup: %-20v lpm: %-15v value: %v, ok: %v\n", ip, lpm, value, ok)

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
	//
	// Lookup: 42.0.0.0             lpm: invalid Prefix  value: <nil>, ok: false
	// Lookup: 10.0.1.17            lpm: 10.0.1.0/24     value: <nil>, ok: true
	// Lookup: 2001:7c0:3100:1::111 lpm: 2000::/3        value: <nil>, ok: true
}

func ExampleTable_Walk() {
	cb := func(p netip.Prefix, val any) bool {
		fmt.Printf("%v (%v)\n", p, val)
		return true
	}

	rtbl := new(cidrtree.Table)
	for _, cidr := range input {
		rtbl.Insert(cidr, nil)
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
