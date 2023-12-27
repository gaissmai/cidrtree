package cidrtree

import (
	"bufio"
	"compress/gzip"
	crand "crypto/rand"
	"log"
	mrand "math/rand"
	"net/netip"
	"os"
	"strings"
	"testing"
)

func TestFprintBST(t *testing.T) {
	rtbl := new(Table)
	for i := 1; i <= 40; i++ {
		rtbl.InsertMutable(randPfx4(), nil)
		rtbl.InsertMutable(randPfx6(), nil)
	}
	size, maxDepth, average, deviation := rtbl.statistics()
	t.Logf("v4/v6: size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

	t.Log()

	w := new(strings.Builder)
	if err := rtbl.fprintBST(w); err != nil {
		t.Fatal(err)
	}

	t.Log(w.String())
}

func TestStatisticsRandom(t *testing.T) {
	for i := 10; i <= 100_000; i *= 10 {
		rtbl := new(Table)
		for c := 0; c <= i; c++ {
			rtbl.InsertMutable(randPfx(), nil)
		}
		size, maxDepth, average, deviation := rtbl.statistics()
		t.Logf("v4/v6: size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)
	}
}

func TestStatisticsFullTable(t *testing.T) {
	rtbl := new(Table)
	for _, cidr := range fullTable {
		rtbl.InsertMutable(cidr, nil)
	}
	size, maxDepth, average, deviation := rtbl.statistics()
	t.Logf("FullTable: size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)
}

func TestLPMRandom(t *testing.T) {
	for i := 10; i <= 100_000; i *= 10 {
		rtbl := new(Table)
		for c := 0; c <= i; c++ {
			rtbl.InsertMutable(randPfx(), nil)
		}
		size, maxDepth, average, _ := rtbl.statistics()
		var lpm netip.Prefix
		var depth int

		addr := randAddr()
		if addr.Is4() {
			lpm, _, _, depth = rtbl.root4.lpmIP(addr, 0)
		} else {
			lpm, _, _, depth = rtbl.root6.lpmIP(addr, 0)
		}
		t.Logf("%40v -> %-20v [%2v : %2.0f : %2v] [Depth: match:average:max],  size: %7v", addr, lpm, depth, average, maxDepth, size)
	}
}

func TestLPMFullTableWithDefaultRoutes(t *testing.T) {
	rtbl := new(Table)
	for _, cidr := range fullTable {
		rtbl.InsertMutable(cidr, nil)
	}
	dg4 := netip.MustParsePrefix("0.0.0.0/0")
	dg6 := netip.MustParsePrefix("::/0")

	rtbl.InsertMutable(dg4, nil)
	rtbl.InsertMutable(dg6, nil)

	size, maxDepth, average, deviation := rtbl.statistics()
	t.Logf("FullTable: size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)
	t.Log()

	var lpm netip.Prefix
	var depth int
	var addr netip.Addr

	for i := 0; i <= 20; i++ {
		if i <= 10 {
			addr = randAddr4()
			lpm, _, _, depth = rtbl.root4.lpmIP(addr, 0)
			t.Logf("%40v -> %-20v matched at: %d", addr, lpm, depth)
			continue
		}
		addr = randAddr6()
		lpm, _, _, depth = rtbl.root6.lpmIP(addr, 0)
		t.Logf("%40v -> %-20v matched at: %d", addr, lpm, depth)
	}
}

// ###################################################
// ### helpers
// ###################################################

func randAddr4() netip.Addr {
	var b [4]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(err)
	}
	return netip.AddrFrom4(b)
}

func randAddr6() netip.Addr {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(err)
	}
	return netip.AddrFrom16(b)
}

func randAddr() netip.Addr {
	coin := mrand.Intn(2)
	if coin == 0 {
		return randAddr4()
	}
	return randAddr6()
}

func randPfx4() netip.Prefix {
	bits := mrand.Intn(33)
	pfx, err := randAddr4().Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

func randPfx6() netip.Prefix {
	bits := mrand.Intn(129)
	pfx, err := randAddr6().Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

func randPfx() netip.Prefix {
	coin := mrand.Intn(2)
	if coin == 0 {
		return randPfx4()
	}
	return randPfx6()
}

// ########################################
// ### full internet prefix list, gzipped
// ########################################

var (
	prefixFile = "testdata/prefixes.txt.gz"
	fullTable  = loadFullTable()
)

func loadFullTable() []netip.Prefix {
	var routes []netip.Prefix

	file, err := os.Open(prefixFile)
	if err != nil {
		log.Fatal(err)
	}

	rgz, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(rgz)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		cidr := netip.MustParsePrefix(line)
		routes = append(routes, cidr)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("reading from %v, %v", rgz, err)
	}
	return routes
}
