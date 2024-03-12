# Attacknet

Blockchain networks in the wild are subject to a lot of real life variances that have historically been difficult to capture 
in local or controlled tests. Chaos testing is a disciplined approach to testing a system by proactively simulating and 
identifying failures. Attacknet is a tool that allows you to simulate these real life variances in a controlled environment.
Examples would include adding network latency between nodes, killing nodes at random, filesystem errors being returned.

The overall architecture of Attacknet relies on Kubernetes to run the workloads, [Kurtosis](https://github.com/kurtosis-tech/kurtosis) to orchestrate a blockchain network and
[Chaos Mesh](https://chaos-mesh.org/) to inject faults into it. Attacknet can then be configured to run healthchecks and 
reports back the state of the network at the end of a test. 

![docs/attacknet.svg](docs/attacknet.svg)

### TLDR; Capabilities
Attacknet can be used in the following ways:
- Manually creating test suites/network configs
- Manually running single tests against a network
- Using the planner feature to define a matrix of faults and targets to auto generate test files
- Running the test suites
- (WIP) Exploratory testing

The faults supported by Attacknet include:
- Time based: Clock skew
- Network based: Split networks, Packet loss, corruption, latency, bandwidth throttling
- Container based: Restarting containers, killing containers
- Filesystem based: I/O latency, I/O errors
- Stress based: CPU stress, Memory stress
- (WIP) Kernel based: Kernel faults

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

## Usage guides
## Manually creating/configuring test suites

Attacknet is configured using "test suites". These are yaml files found under `./test-suites` that define everything 
Attacknet needs to genesis a network, test the network, and determine the health of the network. You may have to manually add/remove
targeting criteria from these configs depending on the network being tested.

Test suite configuration is broken into 3 sections:
- Attacknet configuration.
- Harness configuration. This is used to configure the Kurtosis package that will be used to genesis the network. 
- Test configuration. This is used to determine which tests should be run against the devnet and how those tests 
  should be configured.

Here is an annotated test suite configuration that explains what each bit is for:
```yaml
attacknetConfig:
  grafanaPodName: grafana # the name of the pod that grafana will be deployed to. 
  grafanaPodPort: 3000 # the port grafana is listening to in the pod
  waitBeforeInjectionSeconds: 10 
  # the number of seconds to wait between the genesis of the network and the injection of faults. To wait for finality, use 25 mins (1500 secs)
  reuseDevnetBetweenRuns: true # Whether attacknet should skip enclave deletion after the fault concludes. Defaults to false.
  existingDevnetNamespace: kt-ethereum # Omit field for random namespace geneartion. If you want to reuse a running network, you can specify an existing namespace that contains a Kurtosis enclave and run tests against it.
  allowPostFaultInspection: true # When set to true, Attacknet will maintain the port-forward connection to Grafana once the fault has concluded to allow the operator to inspect metrics. Default: true

harnessConfig:
  networkPackage: github.com/crytic/ethereum-package # The Kurtosis package to deploy to instrument the devnet.
  networkConfig: default.yaml # The configuration to use for the Kurtosis package. These live in ./network-configs and are referenced by their filename. 
  networkType: ethereum # no touchy

# The list of tests to be run before termination
testConfig:
   tests:
   - testName: packetdrop-1 # Name of the test. Used for logging/artifacts.
     health:
        enableChecks: true # whether health checks should be run after the test concludes
        gracePeriod: 2m0s # how long the health checks will attempt to pass before marking the test a failure
     planSteps: # the list of steps to facilitate the test, executed in order
      - stepType: injectFault # this step injects a fault, the continues to the next step without waiting for the fault to terminate
        description: "inject fault"
        chaosFaultSpec: # The chaosFaultSpec is basically a pass-thru object for Chaos Mesh fault resources. This means we can support every possible fault out-of-the-box. To determine the schema for each fault type, check the Chaos Mesh docs: https://chaos-mesh.org/docs/simulate-network-chaos-on-kubernetes/. One issue with this method is that Attacknet can't verify whether your faultSpec is valid until it tries to create the resource in Kubernetes, and that comes after genesis which takes a long time on its own. If you run into schema validation issues, try creating these objects directly in Kubernetes to hasten the debug cycle. 
          kind: NetworkChaos
          apiVersion: chaos-mesh.org/v1alpha1
          spec:
            selector:
              labelSelectors:
                kurtosistech.com/id: cl-1-lighthouse-geth-validator
            mode: all
            action: loss
            duration: 1m
            loss:
              loss: '10'
              correlation: '0'
            direction: to
      - stepType: waitForFaultCompletion # this step waits for all previous running faults to complete before continuing
        description: wait for faults to terminate
```

Over the long term, expect manual fault configuration to be deprecated in favor of the fault planner and other automatic test
generation tools.

## Automatically creating test suites/network configs using the planner

Attacknet can automatically create test suites based off a pre-defined test plan. This can be used to create large, comprehensive test suites that test against a variety of different client combos. This feature is highly experimental at this time.

An example test plan can be found in the `planner-configs/` directory
Here's an annotated version:

```yaml
execution: # list of execution clients that will be used in the network topology
  - name: geth
    image: ethereum/client-go:latest
  - name: reth
    image: ghcr.io/paradigmxyz/reth:latest
consensus: # list of consensus clients that will be used in the network topology
  - name: lighthouse
    image: sigp/lighthouse:latest
    has_sidecar: true
  - name: prysm
    image: prysmaticlabs/prysm-beacon-chain:latest,prysmaticlabs/prysm-validator:latest
    has_sidecar: true
network_params:
  num_validator_keys_per_node: 32 # required. 
kurtosis_package: "github.com/kurtosis-tech/ethereum-package"
kubernetes_namespace: kt-ethereum
topology:
  bootnode_el: geth  # self explanatory
  bootnode_cl: prysm
  targets_as_percent_of_network: 0.25 # [optional] defines what percentage of the network contains the target client. 0.25 means only 25% of nodes will contain the client defined in the fault spec. Warning: low percentages may lead to massive networks.
  target_node_multiplier: 2 # optional, default:1. Adds duplicate el/cl combinations based on the multiplier. Useful for testing weird edge cases in consensus
fault_config:
  fault_type: ClockSkew  # which fault to use. A list of faults currently supported by the planner can be found in pkg/plan/suite/types.go in FaultTypeEnum
  target_client: reth # which client to test. this can be an exec client or a consensus client. must show up in the client definitions above.
  wait_before_first_test: 300s # how long to wait before running the first test. Set this to 25 minutes to test against a finalized network.
  fault_config_dimensions: # the different fault configurations to use when creating tests. At least one config dimension is required.
    - skew: -2m # these configs differ for each fault
      duration: 1m
      grace_period: 1800s # how long to wait for health checks to pass before marking the test as failed
    - skew: 2m
      duration: 1m
      grace_period: 1800s
  fault_targeting_dimensions: # Defines how we want to impact the targets. We can inject faults into the client and only the client, or we can inject faults into the node (injects into cl, el, validator)
    - MatchingNode
    - MatchingClient
  fault_attack_size_dimensions: # Defines how many of the matching targets we actually want to attack. 
    - AttackOneMatching # attacks only one matching target
    - AttackMinorityMatching # attacks <33% 
    - AttackSuperminorityMatching # attacks >33% but <50%
    - AttackMajorityMatching # attacks >50% but <66%
    - AttackSupermajorityMatching # attacks >66%
    - AttackAllMatching # attacks all
```

The total number of tests generated by a plan is equal to `len(fault_config_dimensions) * len(fault_targeting_dimensions) * len(fault_attack_size_dimensions)`

You can create a test plan by invoking `attacknet plan <suitename> <planner config path>`

The suite plan will be written to `./test-suites/plan/<suitename>.yaml`

The network config will be written to `./network-configs/plan/<suitename>.yaml`

and can be executed by attacknet using `attacknet start plan/suitename`

### Faults supported by planner

#### ClockSkew
Config:
```yaml
    - skew: -2m # how far to skew the clock. can be positive or negative
      duration: 1m # how long to skew the clock for
      grace_period: 1800s # how long to wait for health checks to pass before marking the test as failed
```

#### RestartContainers
Config:
```yaml
    - grace_period: 1800s # how long to wait for health checks to pass before marking the test as failed
```

#### IOLatency
Config:
```yaml
    - grace_period: 1800s # how long to wait for health checks to pass before marking the test as failed
      delay: 1000ms # how long the i/o delay should be
      duration: 1m # how long the fault should last
      percent: 50 # the percentage of i/o requests impacted.
```


## Running test suites

Once you've got your configuration set up, you can run Attacknet:

`attacknet start <suitename>`

If your suite config is located at `./test-suites/suite.yaml`, you would run `attacknet start suite`. This will 
probably be changed.

Depending on the state of the Kurtosis package and tons of other variables, a lot of the example test suites/networks might not work out of the box.
If you're just trying to test things out, use `attacknet start suite`. This refers to a demo test suite that was tested on Jan 30.

## Contribution
This tool was developed as a collaboration between [Trail of Bits](https://www.trailofbits.com/) and the [Ethereum Foundation](https://github.com/ethereum/).
Thank you for considering helping out with the source code! We welcome contributions from anyone on the internet, and are grateful for even the smallest of fixes!

If this tool was used for finding bugs, please do ensure that the bug is reported to the relevant project maintainers or to the 
[Ethereum foundation Bug bounty program](https://ethereum.org/en/bug-bounty/). Please feel free to reach out to the tool
maintainers on Discord, Email or Twitter for any feature requests. 

## Changelog

**TBD**

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

## Developing (wip)

1. Install pre-commit
   - `brew install pre-commit`
   - `pre-commit install`

When making pull requests, target the `develop` branch, not main.
