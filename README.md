# Filecoin (go-filecoin)

[![CircleCI](https://circleci.com/gh/filecoin-project/go-filecoin.svg?style=svg&circle-token=5a9d1cb48788b41d98bdfbc8b15298816ec71fea)](https://circleci.com/gh/filecoin-project/go-filecoin)

> Filecoin implementation in Go, turning the world’s unused storage into an algorithmic market.

**Table of Contents**
<!--
    TOC generated by https://github.com/thlorenz/doctoc
    Install with `npm install -g doctoc`.
    Regenerate with `doctoc README.md`.
    It's ok to edit manually if you don't have/want doctoc.
 -->
<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [What is Filecoin?](#what-is-filecoin)
- [Install](#install)
  - [System Requirements](#system-requirements)
<!--  - [Install from Release Binary](#install-from-release-binary) -->
  - [Install from Source](#install-from-source)
    - [Install Go and Rust](#install-go-and-rust)
    - [Install Dependencies](#install-dependencies)
    - [Build, Run Tests, and Install](#build-run-tests-and-install)
    - [Manage Submodules Manually (optional)](#manage-submodules-manually-optional)
- [Usage](#usage)
  - [Advanced usage](#advanced-usage)
    - [Run Multiple Nodes with IPTB](#run-multiple-nodes-with-iptb)
- [Contributing](#contributing)
- [Community](#community)
- [Developer Networks](#developer-networks)
- [License](#license)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## What is Filecoin?
Filecoin is a decentralized storage network that turns the world’s unused storage into an algorithmic market, creating a permanent, decentralized future for the web. **Miners** earn the native protocol token (also called “filecoin”) by providing data storage and/or retrieval. **Clients** pay miners to store or distribute data and to retrieve it. Check out [How Filecoin Works](https://github.com/filecoin-project/go-filecoin/wiki/How-Filecoin-Works) for more.

**WARNING**: `go-filecoin` is a work in progress and is not ready for production use.
See [KNOWN_ISSUES](https://github.com/filecoin-project/go-filecoin/blob/master/KNOWN_ISSUES.md) for an outline of known vulnerabilities.


## Install

👋 Welcome to Go Filecoin!

<!--
- To **run** `go-filecoin` for mining, storing or other exploring, jump straight to
  [detailed setup instructions](https://github.com/filecoin-project/go-filecoin/wiki/Getting-Started).
- To **build** `go-filecoin` from source for development, keep following this README.
-->


### System Requirements

Filecoin can build and run on most Linux and MacOS systems. Windows is not yet supported.

<!--
### Install from Release Binary

- We host prebuilt binaries for Linux and OSX at [Releases](https://github.com/filecoin-project/go-filecoin/releases/). Log in with Github.
- Follow the remaining steps in [Start running Filecoin](https://github.com/filecoin-project/go-filecoin/wiki/Getting-Started#start-running-filecoin)
-->

### Install from Source

Clone the git repository:

```sh
mkdir -p ${GOPATH}/src/github.com/filecoin-project
git clone https://github.com/filecoin-project/go-filecoin.git ${GOPATH}/src/github.com/filecoin-project/go-filecoin
```

Now install the tools and dependencies listed below. If you have **any problems building go-filecoin**, see the [Troubleshooting & FAQ](https://github.com/filecoin-project/go-filecoin/wiki/Troubleshooting-&-FAQ) Wiki page.

#### Install Go and Rust

The build process for go-filecoin requires:

- [Go](https://golang.org/doc/install) >= v1.11.2.
  - Installing Go for the first time? We recommend [this tutorial](https://www.ardanlabs.com/blog/2016/05/installing-go-and-your-workspace.html) which includes environment setup.
- [Rust](https://www.rust-lang.org/) >= v1.31.0 and `cargo`
- `pkg-config` - used by go-filecoin to handle generating linker flags
  - Mac OS devs can install through brew `brew install pkg-config`

Due to our use of `cgo`, you'll need a C compiler to build go-filecoin whether you're using a prebuilt libfilecoin_proofs (our cgo-compatible rust-fil-proofs library) or building it yourself from source. If you want to use `gcc` (e.g. `export CC=gcc`) when building go-filecoin, you will need to use v7.4.0 or higher.
  - You must have libclang on you linker search path in order to build
    rust-fil-proofs from source. You can satisfy this requirement in most
    environments by installing Clang using your favorite package manager.

#### Install Dependencies

`go-filecoin` depends on some proofs code written in Rust, housed in the
[rust-fil-proofs](https://github.com/filecoin-project/rust-fil-proofs) repo and consumed as a submodule. You will need to have `cargo` and `jq` installed.

go-filecoin's dependencies are managed by [gx][2]; this project is not "go gettable." To install gx, gometalinter, and
other build and test dependencies (with precompiled proofs, recommended), run:

```sh
cd ${GOPATH}/src/github.com/filecoin-project/go-filecoin
FILECOIN_USE_PRECOMPILED_RUST_PROOFS=true go run ./build/*.go deps
```

Note: The first time you run `deps` can be **slow** as a ~1.6GB parameter file is either downloaded or generated locally in `/tmp/filecoin-proof-parameters`.
Have patience; future runs will be faster.

#### Build, Run Tests, and Install

```sh
# First, build the binary
go run ./build/*.go build

# Install go-filecoin to ${GOPATH}/bin (necessary for tests)
go run ./build/*.go install

# Then, run the tests.
go run ./build/*.go test

# Build and test can be combined!
go run ./build/*.go best
```

Other handy build commands include:

```sh
# Check the code for style and correctness issues
go run ./build/*.go lint

# Test with a coverage report
go run ./build/*.go test -cover

# Test with Go's race-condition instrumentation and warnings (see https://blog.golang.org/race-detector)
go run ./build/*.go test -race

# Deps, Lint, Build, Test (any args will be passed to `test`)
go run ./build/*.go all
```

Note: Any flag passed to `go run ./build/*.go test` (e.g. `-cover`) will be passed on to `go test`.

If you have **problems with the build**, please see the [Troubleshooting & FAQ](https://github.com/filecoin-project/go-filecoin/wiki/Troubleshooting-&-FAQ) Wiki page.


#### Manage Submodules Manually (optional)

If you're editing `rust-fil-proofs`, you need to manage the submodule manually. If you're *not* editing `rust-fil-proofs` you can relax:
`deps` build (above) will do it for you. You may need to run `deps` again after pulling master if the submodule is
updated by someone else (it will appear modified in `git status`).

To initialize the submodule:

```sh
cd ${GOPATH}/src/github.com/filecoin-project/go-filecoin
git submodule update --init
```

Later, when the head of the `rust-fil-proofs` `master` branch changes, you may want to update `go-filecoin` to use these changes:

```sh
git submodule update --remote
```

Note that updating the `rust-fil-proofs` submodule in this way will require a commit to `go-filecoin` (changing the submodule hash).

## Usage

The [Getting Started](https://github.com/filecoin-project/go-filecoin/wiki/Getting-Started) wiki page contains
a simple sequence to get your Filecoin node up and running and connected to a devnet.

The [Commands](https://github.com/filecoin-project/go-filecoin/wiki/Commands) page contains further detail about
specific commands and environment variables, as well as scripts for for setting up a miner and making a deal.

To see a full list of commands, run `go-filecoin --help`.

### Advanced usage

#### Run Multiple Nodes with IPTB

The [`localfilecoin` IPTB plugin](https://github.com/filecoin-project/go-filecoin/tree/master/tools/iptb-plugins) provides an automation layer that makes it easy to run multiple filecoin nodes. For example, it enables you to easily start up 10 mining nodes locally on your machine.

## Contributing

We ❤️ all our contributors; this project wouldn’t be what it is without you! If you want to help out, please see [CONTRIBUTING.md](CONTRIBUTING.md).

Check out the [Go-Filecoin code overview](CODEWALK.md) for a brief tour of the code.

## Community

Here are a few places to get help and connect with the Filecoin community:
- [Documentation Wiki](https://github.com/filecoin-project/go-filecoin/wiki) — for tutorials, troubleshooting, and FAQs
- #fil-dev on [Filecoin Project Slack](https://filecoinproject.slack.com/messages/CEHHJNJS3/) or [Matrix/Riot](https://riot.im/app/#/room/#fil-dev:matrix.org) - for live help and some dev discussions
- [Filecoin Community Forum](https://discuss.filecoin.io) - for talking about design decisions, use cases, implementation advice, and longer-running conversations
- [GitHub issues](https://github.com/filecoin-project/go-filecoin/issues) - for now, use only to report bugs, and view or contribute to ongoing development. PRs welcome! Please see [our contributing guidelines](CONTRIBUTING.md).

Looking for even more? See the full rundown at [filecoin-project/community](https://github.com/filecoin-project/community).

## Developer Networks

There are currently 3 developer networks (aka devnets) available for development and testing. These are subject to _**frequent downtimes and breaking changes**_. See [Devnets](https://github.com/filecoin-project/go-filecoin/wiki/Devnets) in the wiki for a description of
these developer networks and instructions for connecting your nodes to them.

## License

The Filecoin Project is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](https://github.com/filecoin-project/go-filecoin/blob/master/LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](https://github.com/filecoin-project/go-filecoin/blob/master/LICENSE-MIT) or http://opensource.org/licenses/MIT)


[1]: https://golang.org/dl/
[2]: https://github.com/whyrusleeping/gx
[3]: https://github.com/RichardLitt/standard-readme
[4]: https://golang.org/doc/install
[5]: https://www.rust-lang.org/en-US/install.html
