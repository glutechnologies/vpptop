# VPPTop

**VPPTop** is a Go implementation of real-time data viewer for VPP interfaces and metrics displayed in dynamic terminal user interface.

## Preview

Below is a short demo preview of **VPPTop** in action:

[![preview][preview-svg]][preview]

## Features

VPPTop currently supports following metrics:

* **Interfaces** - shows full list of interfaces with associated data like VPP interface index, MTU, real-time Rx/Tx counters, dropped packets and so on. 
* **Node stats** - information about VPP runtime including node name, state, clocks, vectors, calls, suspends...      
* **Error counters** - number of errors with associated node and reason.
* **Memory usage** - data about free and used memory per thread.
* **Thread info** - displays data about thread ID and name, PID, number of cores, etc.

## VPP Requirements

[VPP][wiki-vpp] version supported is:

- **v25.10-release**

VPPTop uses the binary API bindings bundled with GovPP. The VPP and GovPP
versions must therefore use compatible binary API schemas.

### Install VPP

To install VPP from Packagecloud, run the following commands using the `2510`
repository:

```
curl -s https://packagecloud.io/install/repositories/fdio/2510/script.deb.sh | sudo bash
sudo apt-get install -y vpp vpp-dev vpp-plugin-core
```

For more information about how to install VPP from packages, see the [FD.io wiki page][vpp-install]. 

### Configure VPP

The VPPTop uses VPP stats API for retrieving statistics. Older VPP versions may have the stats API disabled by default. If this is the case, use following guide  [`statseg` section][stats-guide] to learn how to modify VPP config and enable it, or use the following in the VPP config:

```
# enable stats socket (default at /run/vpp/stats.sock)
statseg {
    default
}
```

## Install & Run VPPTop

VPPTop requires [Go][go-download] **1.26** (or newer) to install and run.

### Install

To install VPPTop run:

```shell
# install latest release version of vpptop
go install github.com/glutechnologies/vpptop@latest
# install master branch version of vpptop
go install github.com/glutechnologies/vpptop@master
```

### Run

To start VPPTop run:

```shell
# sudo might be required here, because of the permissions to stats socket file
sudo -E vpptop
```

### Remote nodes

Connect to a GovPP proxy on a remote node by passing its IP address:

```shell
vpptop node 192.0.2.10
```

The proxy must listen on TCP port `7878`. Kubernetes discovery has been
removed: vpptop no longer reads kubeconfig files or resolves Kubernetes node
names, and the `--kubeconfig` option is no longer available. IPv4 and IPv6
addresses are accepted.

In case you have cloned the repository, use can use `make` to build or install binaries:
```shell
make build
# or
make install
```

The command builds a single VPPTop binary using GovPP directly.

VPPTop also supports a light terminal theme. To use darker colors which have better visibility on light background set `VPPTOP_THEME_LIGHT` environment variable.

**Note:** VPPTop expects VPP be running during the startup. Delayed start is currently not available.

### Keybindings

1. Keyboard arrows ``Up, Down, Left, Right`` to switch tabs, scroll.
2. ``Crtl-Space`` open/close menu for sort by a column for the active table.
3. ``/`` to filter the active table, `Enter` to keep the filter.
4. ``Esc`` to cancel the previous operation.
5. ``PgDn PgUp`` to skip pages in the active table.
6. ``Ctrl-C`` to clear counters for the active table.
7. ``q`` to quit from the application

[go-download]: https://golang.org/dl/
[preview]: https://asciinema.org/a/NHODZM2ebcwWFPEEPcja8X19R
[preview-svg]: https://asciinema.org/a/NHODZM2ebcwWFPEEPcja8X19R.svg
[stats-guide]: https://wiki.fd.io/view/VPP/Command-line_Arguments#statseg_.7B_..._.7D
[vpp-install]: https://wiki.fd.io/view/VPP/Installing_VPP_binaries_from_packages
[wiki-tui]: https://en.wikipedia.org/wiki/Text-based_user_interface
[wiki-vpp]: https://wiki.fd.io/view/VPP
