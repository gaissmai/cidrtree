package cidrtree_test

import (
	"bufio"
	"compress/gzip"
	"log"
	"math/rand"
	"net/netip"
	"os"
	"runtime"
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

// full internet prefix list, gzipped
var (
	prefixFile    = "testdata/prefixes.txt.gz"
	internetCIDRs = makeInternetPrefixes()
)

func makeInternetPrefixes() []netip.Prefix {
	var cidrs []netip.Prefix

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
		cidrs = append(cidrs, netip.MustParsePrefix(line))
	}
	if err := scanner.Err(); err != nil {
		log.Printf("reading from %v, %v", rgz, err)
	}
	return cidrs
}

func sliceCIDRs(n int) []netip.Prefix {
	if n > len(internetCIDRs) {
		panic("n too big")
	}

	var clone []netip.Prefix
	clone = append(clone, internetCIDRs...)

	rand.Shuffle(len(clone), func(i, j int) {
		clone[i], clone[j] = clone[j], clone[i]
	})
	return clone[:n]
}

func BenchmarkNew(b *testing.B) {
	for k := 1; k <= 1_000_000; k *= 10 {
		cidrs := sliceCIDRs(k)
		b.Run(intMap[k], func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_ = cidrtree.New(cidrs...)
			}
		})
	}
}

func BenchmarkNewCC(b *testing.B) {
	for k := 1; k <= 1_000_000; k *= 10 {
		cidrs := sliceCIDRs(k)
		b.Run(intMap[k], func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_ = cidrtree.NewConcurrent(runtime.NumCPU(), cidrs...)
			}
		})
	}
}

func BenchmarkClone(b *testing.B) {
	for k := 1; k <= 1_000_000; k *= 10 {
		tree := cidrtree.New(sliceCIDRs(k)...)
		name := "Clone" + intMap[k]
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_ = tree.Clone()
			}
		})
	}
}

func BenchmarkInsert(b *testing.B) {
	for n := 1; n <= 1_000_000; n *= 10 {
		tree := cidrtree.New(sliceCIDRs(n)...)
		probe := cidrs[rand.Intn(len(cidrs))]
		name := "Into" + intMap[n]
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_ = tree.Insert(probe)
			}
		})
	}
}

func BenchmarkInsertMutable(b *testing.B) {
	for n := 1; n <= 1_000_000; n *= 10 {
		tree := cidrtree.New(sliceCIDRs(n)...)
		probe := cidrs[rand.Intn(len(cidrs))]
		name := "Into" + intMap[n]

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				tree.InsertMutable(probe)
			}
		})
	}
}

func BenchmarkDelete(b *testing.B) {
	for n := 1; n <= 1_000_000; n *= 10 {
		tree := cidrtree.New(sliceCIDRs(n)...)
		probe := cidrs[rand.Intn(len(cidrs))]
		name := "DeleteFrom" + intMap[n]

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_, _ = tree.Delete(probe)
			}
		})
	}
}

func BenchmarkMutableDelete(b *testing.B) {
	for n := 1; n <= 1_000_000; n *= 10 {
		tree := cidrtree.New(sliceCIDRs(n)...)
		probe := cidrs[rand.Intn(len(cidrs))]
		name := "DeleteFrom" + intMap[n]

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_ = tree.DeleteMutable(probe)
			}
		})
	}
}

func BenchmarkUnionImmutable(b *testing.B) {
	this100_000 := cidrtree.New(sliceCIDRs(100_000)...)
	for n := 10; n <= 100_000; n *= 10 {
		tree := cidrtree.New(sliceCIDRs(n)...)
		name := "size100_000with" + intMap[n]

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_ = this100_000.Union(tree, true)
			}
		})
	}
}

func BenchmarkUnionMutable(b *testing.B) {
	this100_000 := cidrtree.New(sliceCIDRs(100_000)...)
	for n := 10; n <= 100_000; n *= 10 {
		tree := cidrtree.New(sliceCIDRs(n)...)
		name := "size100_000with" + intMap[n]

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_ = this100_000.Union(tree, false)
			}
		})
	}
}

func BenchmarkLookupMatch(b *testing.B) {
	for n := 100; n <= 1_000_000; n *= 10 {
		tree := cidrtree.New(sliceCIDRs(n)...)
		probe := sliceCIDRs(100)[0]
		ip := probe.Addr()
		name := "In" + intMap[n]

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_, _ = tree.Lookup(ip)
			}
		})
	}
}

func BenchmarkLookupMiss(b *testing.B) {
	for n := 100; n <= 1_000_000; n *= 10 {
		tree := cidrtree.New(sliceCIDRs(n)...)
		ip := netip.MustParseAddr("209.46.0.0")
		name := "In" + intMap[n]

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_, _ = tree.Lookup(ip)
			}
		})
	}
}
