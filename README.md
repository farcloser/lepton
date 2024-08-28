# lepton

> A lightweight, fast, multi-platform containers cli

## Why another cli

Besides Docker, there is a handful of good clis already - most notably ctr, crictl, and nerdctl.

They all have their strengths and use-cases, though nerdctl is definitely the best
choice for most users, with a wide range of features and compatibility with docker.
If you are not sure what you are looking for, or if you want a docker drop-in replacement,
we strongly recommend [nerdctl](https://github.com/containerd/nerdctl), as it is best-in-class in its category.

Although, nerdctl's features depth and breadth, large number of use-cases, maturity, along 
with some of it architectural choices, make it a big, complex, hard to re-architecture project.

Specifically some of these choices make it prone to recurring issues with concurrency, 
handling of hosts.toml and authentication, and overall requirements and installation complexity.

## What is different with lepton

Note that lepton started as a friendly fork of nerdctl, at commit 1a55c720256c629371ed092278acf9e1d5bd83c0.

To the extent that it is possible, lepton will get rebased regularly, and conversely,
we will submit PRs to nerdctl where it makes sense for both projects to benefit.

In terms of changeset, lepton:
1. runs on macOS, linux and windows (nerdctl does not run on macOS)
1. compose, build, and general container operations can be used individually in different binaries
1. significantly reduced complexity, codebase size, and footprint
1. filesystem storage for state and volumes is much more rational:
   1. no limitations on identifiers
   2. safe to use concurrently
1. does not rely on shelling out to third-party binaries

To achieve this, some choices have been made:
1. reduced feature-set
   1. alternative snapshotters are not supported (except Soci) of Soci
   2. systemd is not a target
   3. only the latest LTS version of OS and containerd are supported
1. behavior compatibility with docker and nerdctl is still desired, but on a best-effort basis 
, and not a strong guarantee: output format may change in places, and some features may not be supported, 
or behave slightly differently. This will be especially true when docker UX does not make sense.

The overall purpose is to have a _lightweight_, _fast_, _concurrency safe_ cli that does less
with higher quality.

lepton architecture also makes it very easy to roll your own cli, using it as a library.

## Installation

## Build from source

## 


## History

Dropped:
- snapshotters:
  - nydus
  - stargz
  - overlaybd
  - ipfs
- platforms:
  - freebsd
- compat:
  - schema1
- flags:
  - --debug-full - --debug will now do what debug-full was doing
  - --insecure-registry - use hosts.toml instead

Incompatible changes:
- moved storage to new implementation

## TODO

- kill: pkg/labels vs. pkg/annotations
- make the binary name / prefix configurable instead of const