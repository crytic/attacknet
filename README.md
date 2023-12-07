# Attacknet

## Getting started

### Installation/Building

1. Install Go 1.20 or newer
2. In the project root, run `go build ./cmd/attacknet`
3. Copy the "attacknet" binary to your PATH or directly invoke it.

### Setting up the other bits

1. Set up a containerd k8s cluster. (1.25 or older) (todo: recommended resourcing. Also note that auto-scaling can 
   sometimes be too slow, and kurtosis will time out before the nodes for its workload can be provisioned.)
2. Authenticate to the cluster for kubectl
3. Install chaos-mesh
   1. `kubectl create ns chaos-mesh`
   2. `helm repo add chaos-mesh https://charts.chaos-mesh.org`
   3. `helm install chaos-mesh chaos-mesh/chaos-mesh -n=chaos-mesh --version 2.6.1 --set chaosDaemon.runtime=containerd --set chaosDaemon.socketPath=/run/containerd/containerd.sock --set dashboard.securityMode=false --set bpfki.create=true`
   4. To access chaos dashboard, use `kubectl --namespace chaos-mesh port-forward svc/chaos-dashboard 2333`
4. Install kurtosis locally.
5. Run `kurtosis cluster set cloud`
6. In a separate terminal, run `kurtosis engine start`
7. In a separate terminal, run `kurtosis gateway`. This process needs to stay alive during all attacknet testing and cannot be started via SDK.

## Test Suite Configuration

Attacknet is configured using "test suites". These are yaml files found under `./test-suites` that define everything 
Attacknet needs to genesis a network, test the network, and determine the health of the network.

Test suite configuration is broken into 3 sections:
- Attacknet configuration.
- Harness configuration. This is used to configure the Kurtosis package that will be used to genesis the network. 
- Test configuration. This is used to determine which tests should be run against the devnet and how those tests 
  should be configured. As of right now, only the first test in the array is run before exiting.

Here is an annotated test suite configuration that explains what each bit is for:
```yaml
attacknetConfig:
  grafanaPodName: grafana # the name of the pod that grafana will be deployed to. 
  grafanaPodPort: 3000 # the port grafana is listening to in the pod
  waitBeforeInjectionSeconds: 10 
  # the number of seconds to wait between the genesis of the network and the injection of faults. To wait for finality, use 25 mins (1500 secs)
  reuseDevnetBetweenRuns: true # Whether attacknet should skip enclave deletion after the fault concludes. Defaults to false.
  existingDevnetNamespace: kt-ethereum # If you don't want to genesis a new network, you can specify an existing namespace that contains a Kurtosis enclave and run tests against it instead. I'm expecting this to only be useful for dev/tool testing. Exclude this parameter for normal operation.
  allowPostFaultInspection: true # When set to true, Attacknet will maintain the port-forward connection to Grafana once the fault has concluded to allow the operator to inspect metrics. Default: true

harnessConfig:
  networkPackage: github.com/crytic/ethereum-package # The Kurtosis package to deploy to instrument the devnet.
  networkConfig: default.yaml # The configuration to use for the Kurtosis package. These live in ./network-configs and are referenced by their filename. 


# The list of tests to be run. As of right now, the first test is run and the tool terminates. In the future, we will genesis single-use devnets for each test, run the test, and terminate once all the tests are completed and all the enclaves are cleaned up.
tests:
- testName: packetdrop-1 # Name of the test. Used for logging/artifacts.
  chaosFaultSpec: # The chaosFaultSpec is basically a pass-thru object for Chaos Mesh fault resources. This means we can support every possible fault out-of-the-box, but slightly complicates generating the configuration. To determine the schema for each fault type, check the Chaos Mesh docs: https://chaos-mesh.org/docs/simulate-network-chaos-on-kubernetes/. One issue with this method is that Attacknet can't verify whether your faultSpec is valid until it tries to create the resource in Kubernetes, and that comes after genesis which takes a long time on its own. If you run into schema validation issues, try creating these objects directly in Kubernetes to hasten the debug cycle. 
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

```
## Running the tool

Once you've got your configuration set up, you can run Attacknet:

`attacknet start <suitename>`

If your suite config is located at `./test-suites/suite.yaml`, you would run `attacknet start suite`. This will 
probably be changed.

## Developing (wip)

1. Install pre-commit
   - `brew install pre-commit`
   - `pre-commit install`


## Other stuff

Everything below this line is old and can be ignored. It'll be deleted when I have time to do a pass.


10. Wait for network to hit finality (30ish mins), then run chaos test for clock skew. Skew should be -10m, duration: 30m. Apply it to only one lighthouse EL/CL/Validator. 
11. 10 mins after attack starts, the lighthouse BN should start emitting stale attestations
12. Depending on the proposer schedule, the prysm nodes should start calling replay_blocks and running out a memory a few minutes later, no more than 10.
13. If you actually want the prysm nodes to OOM and crash, set their memory requirement to 1024mb before genesis'ing the network.

### References

https://ethresear.ch/t/cascading-network-effects-on-ethereums-finality/15871

https://docs.prylabs.network/docs/advanced/proof-of-stake-devnet

more focused on execution layer fuzzing:
https://www.usenix.org/system/files/osdi21-yang.pdf
https://github.com/snuspl/fluffy


https://github.com/jepsen-io/tendermint


### Notes (ignore)

prometheus setup
```
kubectl apply --server-side -f manifests/setup
kubectl wait \
	--for condition=Established \
	--all CustomResourceDefinition \
	--namespace=monitoring
kubectl apply -f manifests/


kubectl delete --ignore-not-found=true -f manifests/ -f manifests/setup
```

### Create PVC Inspector pod
```
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: pvc-inspector
spec:
  containers:
  - image: busybox
    name: pvc-inspector
    command: ["tail"]
    args: ["-f", "/dev/null"]
    volumeMounts:
    - mountPath: /pvc
      name: pvc-mount
  volumes:
  - name: pvc-mount
    persistentVolumeClaim:
      claimName: genesis-geth-0
EOF
```

`kubectl exec -it pvc-inspector -- sh`


### Port forwarding

```
kubectl --namespace default port-forward prometheus-operator-6c77ccb5d9-rdlfj 8080
```

```
kubectl --namespace monitoring port-forward svc/prometheus-k8s 9090
```

```
kubectl --namespace monitoring port-forward svc/grafana 3000
```

```
kubectl --namespace chaos-mesh port-forward svc/chaos-dashboard 2333
```


```
kubectl port-forward svc/beacon-prysm 8080
```

### Prom queries

Head slot, beacon nodes
`beacon_head_slot{service=~"beacon-follower-prysm|beacon-follower1-prysm|beacon-follower2-prysm|beacon-follower3-prysm|beacon-follower4-prysm|beacon-follower5-prysm|beacon-follower6-prysm|beacon-prysm"}`

current justified epoch, beacon nodes
`beacon_current_justified_epoch{service=~"beacon-follower-prysm|beacon-follower1-prysm|beacon-follower2-prysm|beacon-follower3-prysm|beacon-follower4-prysm|beacon-follower5-prysm|beacon-follower6-prysm|beacon-prysm"}`

restarts
`kube_pod_container_status_restarts_total{namespace="default"}`


`beacondb_all_deposits{}`
`powchain_valid_deposits_received`
`current_eth1_data_deposit_count`
`beacondb_pending_deposits`
`beacon_processed_deposits_total`


attestation processing rate
`rate(process_attestations_milliseconds_count[5m])`
 passport or identification card (if resident in the UE) number
  + expiry date and country of emission of the identification document
  + date of birth
  + check-in and check-out day (you can omit this if your are staying from October 23th