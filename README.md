
### Prereq

1. install prysmctl
2. set up cluster
3. auth to cluster


### Run goreli nodes
`helm install geth geth --values ./geth/values-goreli.yaml`
`helm install prysm prysm --values ./prysm/values-goreli.yaml`

### Private Devnet, single node

Start execution client (runs until merge):

`helm install geth geth --values ./geth/values-singlenode-64-validators.yaml --wait`

Start beacon chain client
`helm install beacon prysm --values ./prysm/values-singlenode-beacon.yaml --wait`

Start becon client/validator

`helm install validator prysm --values ./prysm/values-singlenode-validator.yaml`


### Docs

https://ethresear.ch/t/cascading-network-effects-on-ethereums-finality/15871

https://docs.prylabs.network/docs/advanced/proof-of-stake-devnet

more focused on execution layer fuzzing:
https://www.usenix.org/system/files/osdi21-yang.pdf
https://github.com/snuspl/fluffy


https://github.com/jepsen-io/tendermint


### Notes

suspected repro requirements:
1. correct version of the prysm client

Actual live env:
2. enough validators in set (mainnet had 600k at time of incident)
3. deposit queue (30k?) (read: big spike in deposits)
4. byzantine fault is 0.2% 

Potential repro env:
2. enough validators in set
3. deposit queue present w/ spike
4. drop packets from CL -> EL



prometheus setup
```
kubectl apply --server-side -f manifests/setup
kubectl wait \
	--for condition=Established \
	--all CustomResourceDefinition \
	--namespace=monitoring
kubectl apply -f manifests/


kubectl delete --ignore-not-found=true -f manifests/ -f manifests/setup

use for grafana: https://github.com/metanull-operator/eth2-grafana/blob/master/eth2-grafana-dashboard-multiple-sources.json
```


### commands for single-node devnet

`k logs --follow geth-0`
`k logs --follow beacon-prysm-0`
`k logs --follow validator-prysm-0`

```
helm install geth geth --values ./geth/values-singlenode-64-validators.yaml --wait && helm install beacon prysm --values ./prysm/values-singlenode-beacon.yaml --wait && helm install validator prysm --values ./prysm/values-singlenode-validator.yaml
```


```
helm upgrade geth geth --values ./geth/values-singlenode-64-validators.yaml 
helm upgrade beacon prysm --values ./prysm/values-singlenode-beacon.yaml
helm upgrade validator prysm --values ./prysm/values-singlenode-validator.yaml
```


```
helm uninstall geth
helm uninstall beacon
helm uninstall validator

```

```
helm diff upgrade geth geth --values ./geth/values-singlenode-64-validators.yaml
helm diff upgrade beacon prysm --values ./prysm/values-singlenode-beacon.yaml
helm diff upgrade validator prysm --values ./prysm/values-singlenode-validator.yaml
```



### commands for multi-node devnet

leader
```
helm install geth geth --values ./geth/values-multi-leader.yaml --wait
helm install beacon prysm --values ./prysm/values-multi-leader-beacon.yaml --wait
helm install validator prysm --values ./prysm/values-multi-leader-validator.yaml
```

follower
```
helm install geth-follower geth  --values ./geth/values-multi-follower.yaml --wait
helm install beacon-follower prysm  --values ./prysm/values-multi-follower-beacon.yaml --wait
helm install validator-follower prysm  --values ./prysm/values-multi-follower-validator.yaml 
```

followerN
```
helm install geth-follower1 geth --values ./geth/values-multi-follower.yaml --wait
helm install beacon-follower1 prysm  --values ./prysm/values-multi-follower-beacon.yaml --values ./prysm/follower/1-beacon.yaml --wait
helm install validator-follower1 prysm  --values ./prysm/values-multi-follower-validator.yaml --values  ./prysm/follower/1-validator.yaml

```

```
helm uninstall geth
helm uninstall beacon
helm uninstall validator

helm uninstall geth-follower
helm uninstall beacon-follower
helm uninstall validator-follower

helm uninstall geth-follower1
helm uninstall beacon-follower1
helm uninstall validator-follower1
```

Startup configs tested

##### Start 66+% single node, wait 32 slots. Once finalizing, start remainder 33%.
Result: network terminates around slot 682
```
time="2023-07-10 17:52:38" level=error msg="Failed to find peers" error="unable to find requisite number of peers for topic /eth2/ca17b34f/beacon_attestation_5/ssz_snappy - only 0 out of 1 peers were able to be found" prefix=p2p
time="2023-07-10 17:52:38" level=warning msg="Attestation is too old to broadcast, discarding it. Current Slot: 682 , Attestation Slot: 5" prefix=p2p
```
Notably, the follower mode kept submitting attestations and stopped at slot 1200

Start 100% validators on single node.

Next test, we'll set the beacon leader's min sync peers to 1.

##### Same as before, but with leader beacon's minpeers set to 1

Follower beacon chain could not join network, was waiting for suitiable peers that did not show up.

##### Start with genesis 5 mins in future

still terminated eventually. slot 155ish.
```
time="2023-07-10 21:45:15" level=warning msg="voting period before genesis + follow distance, using eth1data from head" genesisTime=1689023139 latestValidTime=1688994467 prefix="rpc/validator"
time="2023-07-10 21:45:18" level=warning msg="Execution client is not syncing" prefix=powchain
time="2023-07-10 21:45:18" level=error msg="Beacon node is not respecting the follow distance" prefix=powchain
time="2023-07-10 21:45:23" level=info msg="Got interrupt, shutting down..." prefix=node
time="2023-07-10 21:45:23" level=info msg="Stopping beacon node" prefix=node
time="2023-07-10 21:45:27" level=error msg="Failed to find peers" error="unable to find requisite number of peers for topic /eth2/ca17b34f/beacon_attestation_0/ssz_snappy - only 0 out of 1 peers were able to be found" prefix=p2p
time="2023-07-10 21:45:27" level=warning msg="Attestation is too old to broadcast, discarding it. Current Slot: 199 , Attestation Slot: 0" prefix=p2p
time="2023-07-10 21:45:27" level=error msg="Could not register validator for topic" error="context canceled" prefix=sync topic="/eth2/ca17b34f/sync_committee_1/ssz_snappy"
```

##### start with genesis 5 mins in future, single, 64 validator node
todo: undo --nodiscover in values-multi-leader
undo peers=0 in values-multi-leader

terminated eventually, both beacon and execution clients waiting on each other

trying again, this time disabling k8s restarts.

trying again, this time upping the per-node ram to 16gb. we saw memory-based evictions atound slot 600 last run.
also undoing the genesis timestamp setting. allowing auto-config.


^ terminated 16 hours later, out of memory again
adding memory request/limits to node

^^ was okay after 4 hrs, but mem still going up.
adding --minimum-peers-per-subnet=0

##### testing with multi-node now
- set validator count to 80
- unset --nodiscover in values-multi-leader
- set follower promethus=true
- re-enable genesis-time

Theories:
- there's a minpeers requirement somewhere that's screwing the node up
- we actually need more nodes on the network for it to work
- genesis time in past is a problem [discard]
- --minimum-peers-per-subnet
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
