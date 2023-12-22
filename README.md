# package cidrtree
[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/cidrtree.svg)](https://pkg.go.dev/github.com/gaissmai/cidrtree#section-documentation)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/cidrtree)
[![CI](https://github.com/gaissmai/cidrtree/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/cidrtree/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/cidrtree/badge.svg)](https://coveralls.io/github/gaissmai/cidrtree)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## !!! ATTENTION, API CHANGE AHEAD

The next release v0.2.0 has an API change. See the devel branch with the prepared new API.

## Overview

`package cidrtree` is a datastructure for IP routing tables (IPv4/IPv6) with fast lookup (longest prefix match).

<<<<<<< HEAD
The implementation is based on treaps, which have been augmented here for CIDRs. Treaps are randomized, self-balancing binary search trees. Due to the nature of treaps, the lookups (readers) and updates (writers) can be decoupled without causing delayed rebalancing, which is a perfect fit for a software router or firewall.
||||||| parent of 4eb12b4 (add LookupCIDR method)
Immutability is achieved because insert/delete will return a new tree which will share some nodes with the original tree.
All nodes are read-only after creation, allowing concurrent readers to operate safely with concurrent writers.
=======
The implementation is based on treaps, which have been augmented here for CIDRs. Treaps are randomized, self-balancing binary search trees. Due to the nature of treaps, the lookups (readers) and updates (writers) can be decoupled, which is a perfect fit for a software router or firewall.
>>>>>>> 4eb12b4 (add LookupCIDR method)

This package is a specialization of the more generic [interval package] of the same author,
but explicit for CIDRs. It has a narrow focus with a specialized API for IP routing tables.

[interval package]: https://github.com/gaissmai/interval

## API
```go
  import "github.com/gaissmai/cidrtree"

  type Table struct { // Has unexported fields.  }
    Table is an IPv4 and IPv6 routing table. The zero value is ready to use.

  func New() *Table

  func (t Table) LookupIP(ip netip.Addr) (lpm netip.Prefix, value any, ok bool)
  func (t Table) LookupCIDR(pfx netip.Prefix) (lpm netip.Prefix, value any, ok bool)

  func (t Table) Insert(pfx netip.Prefix, val any) *Table
  func (t Table) Delete(cidr netip.Prefix) (*Table, bool)
  func (t Table) Union(other *Table) *Table
  func (t Table) Clone() *Table

  func (t *Table) InsertMutable(pfx netip.Prefix, val any)
  func (t *Table) DeleteMutable(cidr netip.Prefix) bool
  func (t *Table) UnionMutable(other *Table)

  func (t Table) String() string
  func (t Table) Fprint(w io.Writer) error

  func (t Table) Walk(cb func(pfx netip.Prefix, val any) bool)
```
