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

The implementation is based on treaps, which have been augmented here for CIDRs. Treaps are randomized, self-balancing binary search trees. Due to the nature of treaps, the lookups (readers) and updates (writers) can be decoupled without causing delayed rebalancing, which is a perfect fit for a software router or firewall.

This package is a specialization of the more generic [interval package] of the same author,
but explicit for CIDRs. It has a narrow focus with a specialized API for IP routing tables.

[interval package]: https://github.com/gaissmai/interval

## API
```go
  import "github.com/gaissmai/cidrtree"

  type Route struct{
      CIDR  netip.Prefix   // route
      Value any            // payload, e.g. next hop(s)
  }

  type Tree struct{ /* has unexported fields */ }

  func New(routes ...Route) Tree
  func NewConcurrent(jobs int, routes ...Route) Tree

  func (t Tree) Lookup(ip netip.Addr) (cidr netip.Prefix, value any, ok bool)

  func (t Tree) Insert(cidrs ...netip.Prefix) Tree
  func (t Tree) Delete(cidr netip.Prefix) (Tree, bool)

  func (t *Tree) InsertMutable(cidrs ...netip.Prefix)
  func (t *Tree) DeleteMutable(cidr netip.Prefix) bool

  func (t Tree) Union(other Tree, immutable bool) Tree
  func (t Tree) Clone() Tree

  func (t Tree) String() string
  func (t Tree) Fprint(w io.Writer) error
```
