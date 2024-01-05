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
	1:         "1",
	10:        "10",
	100:       "100",
	1_000:     "1_000",
	10_000:    "10_000",
	100_000:   "100_000",
	1_000_000: "1_000_000",
}

func BenchmarkLookupIP(b *testing.B) {
	for k := 1; k <= 1_000_000; k *= 10 {
		rt := new(cidrtree.Table)
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

func BenchmarkLookupCIDR(b *testing.B) {
	for k := 1; k <= 1_000_000; k *= 10 {
		rt := new(cidrtree.Table)
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

func BenchmarkNew(b *testing.B) {
	for k := 1; k <= 1_000_000; k *= 10 {
		cidrs := shuffleFullTable(k)
		name := fmt.Sprintf("%10s", intMap[k])
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				rt := new(cidrtree.Table)
				for i := range cidrs {
					rt = rt.InsertImmutable(cidrs[i], nil)
				}
			}
		})
	}
}

func BenchmarkClone(b *testing.B) {
	for k := 1; k <= 1_000_000; k *= 10 {
		rt := new(cidrtree.Table)
		for _, cidr := range shuffleFullTable(k) {
			rt = rt.InsertImmutable(cidr, nil)
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
	for k := 1; k <= 1_000_000; k *= 10 {
		rt := new(cidrtree.Table)
		cidrs := shuffleFullTable(k)
		for _, cidr := range cidrs {
			rt = rt.InsertImmutable(cidr, 0)
		}
		probe := routes[mrand.Intn(len(routes))]
		name := fmt.Sprintf("Into%10s", intMap[k])
		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_ = rt.InsertImmutable(probe.cidr, 0)
			}
		})
	}
}

func BenchmarkInsertMutable(b *testing.B) {
	for k := 1; k <= 1_000_000; k *= 10 {
		rt := new(cidrtree.Table)
		cidrs := shuffleFullTable(k)
		for _, cidr := range cidrs {
			rt = rt.InsertImmutable(cidr, 0)
		}
		probe := routes[mrand.Intn(len(routes))]
		name := fmt.Sprintf("Into%10s", intMap[k])
		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				rt.Insert(probe.cidr, 0)
			}
		})
	}
}

func BenchmarkDelete(b *testing.B) {
	for k := 1; k <= 1_000_000; k *= 10 {
		rt := new(cidrtree.Table)
		cidrs := shuffleFullTable(k)
		for _, cidr := range cidrs {
			rt = rt.InsertImmutable(cidr, nil)
		}
		probe := routes[mrand.Intn(len(routes))]
		name := fmt.Sprintf("From%10s", intMap[k])

		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_, _ = rt.DeleteImmutable(probe.cidr)
			}
		})
	}
}

func BenchmarkDeleteMutable(b *testing.B) {
	for k := 1; k <= 1_000_000; k *= 10 {
		rt := new(cidrtree.Table)
		cidrs := shuffleFullTable(k)
		for _, cidr := range cidrs {
			rt = rt.InsertImmutable(cidr, nil)
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

func BenchmarkWalk(b *testing.B) {
	for k := 1; k <= 1_000_000; k *= 10 {
		rt := new(cidrtree.Table)
		cidrs := shuffleFullTable(k)
		for _, cidr := range cidrs {
			rt.Insert(cidr, nil)
		}
		name := fmt.Sprintf("Walk%10s", intMap[k])

		c := 0
		cb := func(netip.Prefix, any) bool {
			c++
			return true
		}

		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				rt.Walk(cb)
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
