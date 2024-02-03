# package cidrtree
[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/cidrtree.svg)](https://pkg.go.dev/github.com/gaissmai/cidrtree#section-documentation)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/cidrtree)
[![CI](https://github.com/gaissmai/cidrtree/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/cidrtree/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/cidrtree/badge.svg)](https://coveralls.io/github/gaissmai/cidrtree)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## ATTENTION

This package is frozen, please migrate to the even better package [BART - BAlanced Routing Table](https://github.com/gaissmai/bart) 

## Overview

`package cidrtree` is a datastructure for IP routing tables (IPv4/IPv6) with fast lookup (longest prefix match).

The implementation is based on treaps, which have been augmented here for CIDRs. Treaps are randomized, self-balancing binary search trees. Due to the nature of treaps the lookups (readers) and the update (writer) can be easily decoupled. This is the perfect fit for a software router or firewall.

This package is a specialization of the more generic [interval package] of the same author, but explicit for CIDRs. It has a narrow focus with a specialized API for IP routing tables.

[interval package]: https://github.com/gaissmai/interval

## API
```go
  import "github.com/gaissmai/cidrtree"

  type Table[V any] struct { // Has unexported fields.  }
    Table is an IPv4 and IPv6 routing table. The zero value is ready to use.

  func (t Table[V]) Lookup(ip netip.Addr) (lpm netip.Prefix, value V, ok bool)
  func (t Table[V]) LookupPrefix(pfx netip.Prefix) (lpm netip.Prefix, value V, ok bool)

  func (t *Table[V]) Insert(pfx netip.Prefix, value V)
  func (t *Table[V]) Delete(pfx netip.Prefix) bool
  func (t *Table[V]) Union(other Table[V])

  func (t Table[V]) InsertImmutable(pfx netip.Prefix, value V) *Table[V]
  func (t Table[V]) DeleteImmutable(pfx netip.Prefix) (*Table[V], bool)
  func (t Table[V]) UnionImmutable(other Table[V]) *Table[V]
  func (t Table[V]) Clone() *Table[V]

  func (t Table[V]) String() string
  func (t Table[V]) Fprint(w io.Writer) error

  func (t Table[V]) Walk(cb func(pfx netip.Prefix, value V) bool)
```
