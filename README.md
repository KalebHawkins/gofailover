
# Failover Automation Tool

This tool is used to perform scheduled failover procedures for various applications utilizing pacemaker as the clustering software.

- [Failover Automation Tool](#failover-automation-tool)
- [Schedule](#schedule)
  - [Building the Binary](#building-the-binary)
    - [Go Compiler Installation](#go-compiler-installation)
    - [GoReleaser Installation](#goreleaser-installation)
    - [Build](#build)


# Schedule

Failover should be performed every 1st Sunday of the month. This should be when the primary master node is failed over to run on the secondary node.
All subsequent Sundays it should be verified that the primary master node is running it's resources. 
If it is not a failover is performed to correct which node the resources are running on. 

The above logic can be broken down as follows:

| Day              | Function                                                                         | 
|------------------|----------------------------------------------------------------------------------|
| 1st Sunday       | Failover from primary to backup node                                             |
| 2nd Sunday       | Failover from backup to primary node                                             |
| 3rd & 4th Sunday | If primary is not running resources perform failover from backup to primary node |

> See the [`docs`](docs/) folder for more information on the tool.

## Building the Binary

To build a binary you will need the `go compiler (v1.17+)` installed and `GoReleaser (v1.7.0+)`. 

> You will also need `git` installed. This documentation assumes that knowledge on how to download and use basic git commands. That process will not be covered here.

### Go Compiler Installation

To install the `go comiler` use the following commands.

```bash
curl -LO https://go.dev/dl/go1.18.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.18.linux-amd64.tar.gz
```

Verify the installation using the `go version` command.

### GoReleaser Installation

To install `GoReleaser` run the following commands.

```bash
curl -L https://github.com/goreleaser/goreleaser/releases/download/v1.7.0/goreleaser_Linux_x86_64.tar.gz | tar zxv -C /usr/bin/
export PATH=$PATH:/usr/local/go/bin
```

Verify the installation using the `goreleaser --version` command.

### Build

To build a binary you need to clone the repository. Move into the repository and run the build command. See below.

```bash
git clone https://hnapxlscmgit01.amerhonda.com/vfc01813/gofailover.git
cd gofailover
goreleaser build --rm-dist
```

This will start the build process creating a `dist` directory where your binary will be available. 