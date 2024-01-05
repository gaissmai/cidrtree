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
	for i := 1; i <= 48; i++ {
		rtbl.Insert(randPfx4(), nil)
		rtbl.Insert(randPfx6(), nil)
	}
	size, maxDepth, average, deviation := rtbl.statistics(skip6)
	t.Logf("v4:  size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

	size, maxDepth, average, deviation = rtbl.statistics(skip4)
	t.Logf("v6:  size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

	size, maxDepth, average, deviation = rtbl.statistics(nil)
	t.Logf("all: size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

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
			rtbl.Insert(randPfx(), nil)
		}
		size, maxDepth, average, deviation := rtbl.statistics(skip6)
		t.Logf("v4:  size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

		size, maxDepth, average, deviation = rtbl.statistics(skip4)
		t.Logf("v6:  size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

		size, maxDepth, average, deviation = rtbl.statistics(nil)
		t.Logf("all: size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

		t.Log()
	}
}

func TestStatisticsFullTable(t *testing.T) {
	rtbl := new(Table)
	for _, cidr := range fullTable {
		rtbl.Insert(cidr, nil)
	}

	size, maxDepth, average, deviation := rtbl.statistics(skip6)
	t.Logf("FullTableV4: size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

	size, maxDepth, average, deviation = rtbl.statistics(skip4)
	t.Logf("FullTableV6: size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

	size, maxDepth, average, deviation = rtbl.statistics(nil)
	t.Logf("FullTable:   size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)
}

func TestLPMRandom(t *testing.T) {
	var size int
	var depth int
	var maxDepth int
	var average float64
	var lpm netip.Prefix

	for i := 10; i <= 100_000; i *= 10 {
		rtbl := new(Table)
		for c := 0; c <= i; c++ {
			rtbl.Insert(randPfx(), nil)
		}

		addr := randAddr()
		if addr.Is4() {
			lpm, _, _, depth = rtbl.root4.lpmIP(addr, 0)
			size, maxDepth, average, _ = rtbl.statistics(skip6)
		} else {
			lpm, _, _, depth = rtbl.root6.lpmIP(addr, 0)
			size, maxDepth, average, _ = rtbl.statistics(skip4)
		}
		t.Logf("%40v -> %-20v [%2v : %2.0f : %2v] [Depth: match:average:max],  size: %7v", addr, lpm, depth, average, maxDepth, size)
	}
}

func TestLPMFullTableWithDefaultRoutes(t *testing.T) {
	var size int
	var depth int
	var maxDepth int
	var average float64
	var deviation float64
	var addr netip.Addr
	var lpm netip.Prefix

	rtbl := new(Table)
	for _, cidr := range fullTable {
		rtbl.Insert(cidr, nil)
	}
	dg4 := netip.MustParsePrefix("0.0.0.0/0")
	dg6 := netip.MustParsePrefix("::/0")

	rtbl.Insert(dg4, nil)
	rtbl.Insert(dg6, nil)

	size, maxDepth, average, deviation = rtbl.statistics(skip6)
	t.Logf("FullTableV4: size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

	size, maxDepth, average, deviation = rtbl.statistics(skip4)
	t.Logf("FullTableV6: size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

	size, maxDepth, average, deviation = rtbl.statistics(nil)
	t.Logf("FullTable:   size: %10d, maxDepth: %4d, average: %3.2f, deviation: %3.2f", size, maxDepth, average, deviation)

	t.Log()

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

func skip4(pfx netip.Prefix, val any, depth int) bool {
	return pfx.Addr().Is4()
}

func skip6(pfx netip.Prefix, val any, depth int) bool {
	return !pfx.Addr().Is4()
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
