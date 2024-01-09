# package cidrtree
[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/cidrtree.svg)](https://pkg.go.dev/github.com/gaissmai/cidrtree#section-documentation)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/cidrtree)
[![CI](https://github.com/gaissmai/cidrtree/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/cidrtree/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/cidrtree/badge.svg)](https://coveralls.io/github/gaissmai/cidrtree)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## !!! ATTENTION, API HAS CHANGED

The API has changed from v0.3.0 to v0.4.0.

## Overview

`package cidrtree` is a datastructure for IP routing tables (IPv4/IPv6) with fast lookup (longest prefix match).

The implementation is based on treaps, which have been augmented here for CIDRs. Treaps are randomized, self-balancing binary search trees. Due to the nature of treaps the lookups (readers) and the update (writer) can be easily decoupled. This is the perfect fit for a software router or firewall.

This package is a specialization of the more generic [interval package] of the same author,
but explicit for CIDRs. It has a narrow focus with a specialized API for IP routing tables.

[interval package]: https://github.com/gaissmai/interval

## API
```go
  import "github.com/gaissmai/cidrtree"

  type Table struct { // Has unexported fields.  }
    Table is an IPv4 and IPv6 routing table. The zero value is ready to use.

  func (t Table) Lookup(ip netip.Addr) (lpm netip.Prefix, value any, ok bool)
  func (t Table) LookupPrefix(pfx netip.Prefix) (lpm netip.Prefix, value any, ok bool)

  func (t *Table) Insert(pfx netip.Prefix, val any)
  func (t *Table) Delete(pfx netip.Prefix) bool
  func (t *Table) Union(other Table)

  func (t Table) InsertImmutable(pfx netip.Prefix, val any) *Table
  func (t Table) DeleteImmutable(pfx netip.Prefix) (*Table, bool)
  func (t Table) UnionImmutable(other Table) *Table
  func (t Table) Clone() *Table

  func (t Table) String() string
  func (t Table) Fprint(w io.Writer) error

  func (t Table) Walk(cb func(pfx netip.Prefix, val any) bool)
```
