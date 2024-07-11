# Documentation

## Overview

Fundamentally, Attacknet is an orchestration tool for performing Chaos Testing. It can be used for multiple different testing workflows ranging from simply injecting faults into a Kubernetes pod to orchestrating a barrage of tests against a dynamically defined network topology. 

To understand its capabilities and how to use it, we've documented a set of "testing workflows", ranging from simple to more complicated, along with accompanying docs on how to use the workflow and what kind of testing it should be used for. 

**NOTE**: at this time, the canonical Kurtosis package for deploying Ethereum devnets is located here: https://github.com/ethpandaops/ethereum-package. However, there's been some compatibility issues that are preventing the latest version of the package from working with Attacknet on Kubernetes. 

Until it gets fixed, we recommend using the following Kurtosis package, however this package may not work for the latest testnet configurations: `github.com/crytic/ethereum-package@0ed559c2661989b44cd2a44eca013acd64786f7f`

### Workflow #1, Inject fault into an existing pod, then quit

For this workflow, we want Attacknet to inject a fault into a pre-existing pod in a pre-existing namespace, then quit.
We need to specify a [Test Suite configuration](#test-suites) file that can provide Attacknet with the information needed to run the workflow.

Note that if there isn't a Kurtosis enclave running in the `pre-existing-namespace` namespace below, Attacknet will try to genesis a network in the namespace.

If the config file below is located at `test-suites/suite.yaml`, you would run it using Attacknet by invoking `attacknet start suite`
```yaml
attacknetConfig:
    grafanaPodName: grafana
    grafanaPodPort: "3000"
    allowPostFaultInspection: true
    waitBeforeInjectionSeconds: 0
    reuseDevnetBetweenRuns: true
    existingDevnetNamespace: pre-existing-namespace
    
harnessConfig: # even though we're not creating a network, these still need to be defined. just a quirk that will be fixed eventually
    networkType: ethereum
    networkPackage: github.com/kurtosis-tech/ethereum-package 
    networkConfig: plan/default.yaml
testConfig:
    tests:
        - testName: Restart geth/lighthouse node
          planSteps:
            - stepType: injectFault
              description: 'Restart geth/lighthouse node'
              chaosFaultSpec:
                apiVersion: chaos-mesh.org/v1alpha1
                kind: PodChaos
                spec:
                    action: pod-failure
                    duration: 5s
                    mode: all
                    selector:
                        expressionSelectors:
                            - key: kurtosistech.com/id
                              operator: In
                              values:
                                - el-1-geth-lighthouse
          health:
            enableChecks: false
```

### Workflow #2, Genesis a network, inject fault into a node, ensure the node recovers, then quit

This workflow requires two config files,a [test suite configuration](#test-suites) and a [network configuration](#network-configs). The test suite config can be found below, and the network config that it points to in [../network-configs/default.yaml](../network-configs/default.yaml)

We've several changes to the test suite from workflow 1, each is annotated with a comment.
```yaml
attacknetConfig:
  grafanaPodName: grafana
  grafanaPodPort: "3000"
  allowPostFaultInspection: true
  waitBeforeInjectionSeconds: 600 # we want to wait until all the nodes are synced and emitting attestations
  reuseDevnetBetweenRuns: true
  existingDevnetNamespace: non-existing-namespace # we use a new namespace to ensure that the chaos is applied to a fresh network

harnessConfig:
  networkType: ethereum
  networkPackage: github.com/kurtosis-tech/ethereum-package
  networkConfig: plan/default.yaml
testConfig:
  tests:
    - testName: Restart geth/lighthouse node
      planSteps:
        - stepType: injectFault
          description: 'Restart geth/lighthouse node'
          chaosFaultSpec:
            apiVersion: chaos-mesh.org/v1alpha1
            kind: PodChaos
            spec:
              action: pod-failure
              duration: 5s
              mode: all
              selector:
                expressionSelectors:
                  - key: kurtosistech.com/id
                    operator: In
                    values:
                      - el-1-geth-lighthouse
      health:
        enableChecks: true # we want attacknet to run health checks against EL/CL clients
        gracePeriod: 5m0s # How long Attacknet should wait for the network to stabilize before considering the test a failure.
```

Since health checks are enabled now, Attacknet will emit a health check artifact once the test concludes (successful or not). These health artifacts can be found in the `./artifacts` directory.

Note: when Attacknet is run using `start suite`, it's going to check whether a network is already running in the `existingDevnetNamespace` namespace. If no network is running, it will genesis a network using the specified network config.

### Workflow #3, use the planner to build a test suite for exhaustively testing a single client, then run the test suite

This workflow is useful for exhaustively testing a specific EL or CL client against a specific fault with various intensities/client combinations. This workflow consumes a [planner config file](#planner-configs) and emits a network config and test suite config that can be run by Attacknet.

Using the example planner config in the [planner configs docs](#planner-configs), we can use `attacknet plan <suitename> <planner config path>` to generate a test suite/network config. The suite plan will be written to `./test-suites/plan/<suitename>.yaml`, and the network config will be written to `./network-configs/plan/<suitename>.yaml`.

The test suite can then be run using `attacknet run plan/<suitename>`.

It should be noted that the number of tests generated will be equal to `len(fault_config_dimensions) * len(fault_targeting_dimensions) * len(fault_attack_size_dimensions)`, so budget your testing dimensions accordingly.

Note that not all faults are supported in the test planner at this time, see the planner config docs for more info.

## Configuration Files
### Test Suites
Test suites are configuration files that tell Attacknet:
1. Which namespace faults/tests should be injected into in Kubernetes
2. Whether a network should be genesis'ed using Kurtosis, and if so, what that network topology should be.
3. The lifecycle rules for the devnet & whether it should be terminated after the suite concludes.
4. The actual tests to run against the devnet. 

These config files are stored as yaml and are found under `./test-suites`. You can manually create new test suites, or use the planner to generate test suites.

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
  reuseDevnetBetweenRuns: true # Whether attacknet should skip enclave deletion after the fault concludes. Defaults to true.
  existingDevnetNamespace: kt-ethereum # If you want to reuse a running network, you can specify an existing namespace that contains a Kurtosis enclave and run tests against it. If this field is defined and no Kurtosis enclave is present, the network defined in the harness configuration will be deployed to it.
  allowPostFaultInspection: true # When set to true, Attacknet will maintain the port-forward connection to Grafana once the fault has concluded to allow the operator to inspect metrics. Default: true

harnessConfig:
  networkPackage: github.com/kurtosis/ethereum-package # The Kurtosis package to deploy to instrument the devnet.
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

#### Plan Steps

In the above example, we use two planSteps, `injectFault` and `waitForFaultCompletion`.

The `injectFault` planStep provides a pass-through to Chaos Mesh, where the manifest under `chaosFaultSpec` is directly written to Kubernetes as a manifest. When Attacknet runs an `injectFault` planStep, it waits until Chaos Mesh has confirmed the fault to be injected into the target pod, then proceeds to the next step. Information on how to configure different kinds of faults can be found in the [Chaos Mesh documentation](https://chaos-mesh.org/docs/simulate-pod-chaos-on-kubernetes/). Some examples can be found in the `test-suites/` directory as well. 

The `waitForFaultCompletion` planStep does exactly what it says. Attacknet determines when currently running faults are expected to terminate by checking their manifest's `duration` field, then holds up the test suite execution for the longest expected `duration`. Once the `duration` has elapsed, it checks all outstanding fault manifests and verifies Chaos Mesh was able to turn off the fault properly.

The `waitForDuration` planStep isn't in the above suite, but it exists. See [pkg/test_executor/types.go](../pkg/test_executor/types.go) for how to configure it.

### Network Configs
These files define the network topology and configuration of a network to be deployed by Kurtosis. You can create them manually or using the planner tool.

They are stored under the `network-configs` directory, and are directly passed through to the Kurtosis package when deploying a devnet. 
When referencing a network config in a test suite, you don't have to include `network-configs` in the path. 

Since these files are entirely passthrough to the Kurtosis package, see the [Ethereum Kurtosis package](https://github.com/kurtosis-tech/ethereum-package) for further documentation. 

### Planner Configs
These files are used by the test planner feature to generate network configs and test suites. They are found in the `planner-configs/` directory.

Here's an annotated test plan:

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
    image: prysmaticlabs/prysm-beacon-chain:latest
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

#### Faults supported by planner

##### ClockSkew
Config:
```yaml
    - skew: -2m # how far to skew the clock. can be positive or negative
      duration: 1m # how long to skew the clock for
      grace_period: 1800s # how long to wait for health checks to pass before marking the test as failed
```

##### RestartContainers
Config:
```yaml
    - grace_period: 1800s # how long to wait for health checks to pass before marking the test as failed
```

##### IOLatency
Config:
```yaml
    - grace_period: 1800s # how long to wait for health checks to pass before marking the test as failed
      delay: 1000ms # how long the i/o delay should be
      duration: 1m # how long the fault should last
      percent: 50 # the percentage of i/o requests impacted.
```

##### Network Latency
Config:
```yaml
  - grace_period: 1800s # how long to wait for health checks to pass before marking the test as failed
    delay: 500ms # how long the latency delay should be, on average
    jitter: 50ms # the amount of jitter
    duration: 5m # how long the fault should last
    correlation: 50 # 0 - 100 
```

##### Packet Loss
Config:
```yaml
  - grace_period: 1800s # how long to wait for health checks to pass before marking the test as failed
    loss_percent: 75% # the pct of packets to drop
    direction: to # may be to, from, or both 
    duration: 5m # how long the fault should last
```


