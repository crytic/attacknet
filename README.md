# Attacknet

Blockchain networks in the wild are subject to a lot of real life variances that have historically been difficult to capture 
in local or controlled tests. Chaos testing is a disciplined approach to testing a system by proactively simulating and 
identifying failures. Attacknet is a tool that allows you to simulate these real life variances in a controlled environment.
Examples would include adding network latency between nodes, killing nodes at random, or filesystem latency.

The overall architecture of Attacknet relies on Kubernetes to run the workloads, [Kurtosis](https://github.com/kurtosis-tech/kurtosis) to orchestrate a blockchain network and
[Chaos Mesh](https://chaos-mesh.org/) to inject faults into nodes. Attacknet can then be configured to run healthchecks and 
reports back the state of the network at the end of a test. 

![docs/attacknet.svg](docs/attacknet.svg)

### Capabilities

The faults supported by Attacknet include:
- Time based: Clock skew
- Network based: Split networks, Packet loss, corruption, latency, bandwidth throttling
- Container based: Restarting containers, killing containers
- Filesystem based: I/O latency, I/O errors
- Stress based: CPU stress, Memory stress
- (WIP) Kernel based: Kernel faults

Attacknet can be used in the following ways:
- Manually creating specific faults that target nodes matching a criteria
- Genesis devnets of specific topologies using [Kurtosis](https://www.kurtosis.com/), then run faults against them.
- Use the planner to define a matrix of faults and targets, automatically generating the network topology and fault configuration.
- (WIP) Exploratory testing. Dynamically generate various faults/targeting criterion and run faults continuously. 

See [DOCUMENTATION.md](docs/DOCUMENTATION.md) for specific usage examples.

## Getting started
### Installation/Building

1. Install Go 1.21 or newer
2. In the project root, run `go build ./cmd/attacknet`
3. Copy the "attacknet" binary path to your PATH variable or directly invoke it

### Setting up the other bits

1. Set up a containerd k8s cluster. (1.27 or older), ideally without auto-scaling (as high provisioning time leads to timeouts on kurtosis) 
2. Authenticate to the cluster for kubectl
3. Install chaos-mesh
   1. `kubectl create ns chaos-mesh`
   2. `helm repo add chaos-mesh https://charts.chaos-mesh.org`
   3. `helm install chaos-mesh chaos-mesh/chaos-mesh -n=chaos-mesh --version 2.6.1 --set chaosDaemon.runtime=containerd --set chaosDaemon.socketPath=/run/containerd/containerd.sock --set dashboard.securityMode=false --set bpfki.create=true`
   4. To access chaos dashboard, use `kubectl --namespace chaos-mesh port-forward svc/chaos-dashboard 2333`
4. Install [kurtosis locally](https://docs.kurtosis.com/install)
5. Run `kurtosis cluster set cloud`, more information [here](https://docs.kurtosis.com/k8s)
6. If running in digitalocean, edit the kurtosis-config.yml file from `kurtosis config path` and add the following setting under kubernetes-cluster-name: `storage-class: "do-block-storage"`
7. In a separate terminal, run `kurtosis engine start`
8. In a separate terminal, run `kurtosis gateway`. This process needs to stay alive during all attacknet testing and cannot be started via SDK.

## Usage/Configuration

See [DOCUMENTATION.md](docs/DOCUMENTATION.md)

## Contributing
This tool was developed as a collaboration between [Trail of Bits](https://www.trailofbits.com/) and the [Ethereum Foundation](https://github.com/ethereum/).
Thank you for considering helping out with the source code! We welcome contributions from anyone on the internet, and are grateful for even the smallest of fixes!

If you use this tool for finding bugs, please do ensure that the bug is reported to the relevant project maintainers or to the 
[Ethereum foundation Bug bounty program](https://ethereum.org/en/bug-bounty/). Please feel free to reach out to the tool
maintainers on Discord, Email or Twitter for any feature requests. 

If you want to contribute to Attacknet, we recommend running pre-commit before making changes:

1. Install pre-commit
2. Run `pre-commit install`

When making pull requests, **please target the `develop` branch, not main.**

## Changelog

**July 11, 2024 version v1.0.1**
- Updated to Go 1.22.5
- Updated Kurtosis SDK to 1.0
- Modified examples to point at temporary Kurtosis package

**March 18, 2024 version v1.0.0**

First public release!

**New**
- Added two new configuration options in the test planner:
  - target_node_multiplier, which duplicates the number of nodes on the network containing the client under test
  - targets_as_percent_of_network, which adds more non-test nodes to the network to improve client diversity testing
- Added new fault options to the test planner:
  - Network latency faults
  - Network packet loss faults
- Beacon chain clients are now included in health checking.
 
**Fixed**
- Fixed an issue where the test planner's resultant network topology was non-deterministic
- Fixed an issue where a dropped port-forwarding connection to a pod may result in a panic
- Fixed an issue where Chaos Mesh would fail to find targets in networks with more than 10 nodes
- Updated for Kurtosis SDK v0.89.3

**Jan 30, 2024 version v0.3 (internal)**
- Fixed the demo example suite
- Fixed issues with the test planner and pod-restart faults.
- Added bootnode configuration for the test planner.
- Attack sizes in the test planner now refer to size in the context of the entire network. 
  - A supermajority-sized attack will try to target 66%+ nodes in the entire network, not just 66% of the nodes that match the test target criteria.
- Peer scoring is now disabled for all planner-generated network configurations.
- Bootnodes are no longer targetable by planner-generated test suites.

**Jan 11, 2024 version v0.2 (internal)**
- Updated to kurtosis v0.86.1
- Updated to Go 1.21
- Grafana port-forwarding has been temporarily disabled
- Introduces multi-step tests. This allows multiple faults and other actions to be composed into a single test.
- Introduces the suite planner. The suite planner allows the user to define a set of testing criteria/dimensions, which the planner turns into a suite containing multiple tests.
- Successful & failed test suites now emit test artifacts summarizing the results of the test.

**Dec 15, 2023 version v0.1 (internal)**
- Initial internal release
