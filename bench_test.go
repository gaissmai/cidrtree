package cidrtree_test

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	mrand "math/rand"
	"net/netip"
	"os"
	"strings"
	"testing"

	"github.com/gaissmai/cidrtree"
)

var intMap = map[int]string{
	1:       "1",
	10:      "10",
	100:     "100",
	1_000:   "1_000",
	10_000:  "10_000",
	100_000: "100_000",
}

func BenchmarkLookup(b *testing.B) {
	for k := 1; k <= 100_000; k *= 10 {
		rt := new(cidrtree.Table[any])
		cidrs := shuffleFullTable(k)
		for _, cidr := range cidrs {
			rt.Insert(cidr, nil)
		}
		probe := cidrs[mrand.Intn(k)]
		ip := probe.Addr()
		name := fmt.Sprintf("In%10s", intMap[k])

		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_, _, _ = rt.Lookup(ip)
			}
		})
	}
}

func BenchmarkLookupPrefix(b *testing.B) {
	for k := 1; k <= 100_000; k *= 10 {
		rt := new(cidrtree.Table[any])
		cidrs := shuffleFullTable(k)
		for _, cidr := range cidrs {
			rt.Insert(cidr, nil)
		}
		probe := cidrs[mrand.Intn(k)]
		name := fmt.Sprintf("In%10s", intMap[k])

		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_, _, _ = rt.LookupPrefix(probe)
			}
		})
	}
}

func BenchmarkClone(b *testing.B) {
	for k := 1; k <= 100_000; k *= 10 {
		rt := new(cidrtree.Table[any])
		for _, cidr := range shuffleFullTable(k) {
			rt.Insert(cidr, nil)
		}
		name := fmt.Sprintf("%10s", intMap[k])
		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_ = rt.Clone()
			}
		})
	}
}

func BenchmarkInsert(b *testing.B) {
	for k := 1; k <= 100_000; k *= 10 {
		rt := new(cidrtree.Table[any])
		cidrs := shuffleFullTable(k)
		for _, cidr := range cidrs {
			rt.Insert(cidr, nil)
		}
		cidr := routes[mrand.Intn(len(routes))].cidr
		name := fmt.Sprintf("Into%10s", intMap[k])

		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				rt.Insert(cidr, nil)
			}
		})
	}
}

func BenchmarkDelete(b *testing.B) {
	for k := 1; k <= 100_000; k *= 10 {
		rt := new(cidrtree.Table[any])
		cidrs := shuffleFullTable(k)
		for _, cidr := range cidrs {
			rt.Insert(cidr, nil)
		}
		probe := routes[mrand.Intn(len(routes))]
		name := fmt.Sprintf("From%10s", intMap[k])

		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_ = rt.Delete(probe.cidr)
			}
		})
	}
}

// #####################################################
// helpers
// #####################################################

// full internet prefix list, gzipped
var (
	prefixFile = "testdata/prefixes.txt.gz"
	fullTable  = loadFullTable()
)

func shuffleFullTable(n int) []netip.Prefix {
	if n > len(fullTable) {
		panic("n too big")
	}

	var clone []netip.Prefix
	clone = append(clone, fullTable...)

	mrand.Shuffle(len(clone), func(i, j int) {
		clone[i], clone[j] = clone[j], clone[i]
	})
	return clone[:n]
}

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
