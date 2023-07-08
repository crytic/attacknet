
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
helm install geth geth --values ./geth/values-multi-leader.yaml --wait && helm install beacon prysm --values ./prysm/values-singlenode-beacon.yaml --wait && helm install validator prysm --values ./prysm/values-singlenode-validator.yaml
```

follower
```
helm install geth-follower geth  --values ./geth/values-multi-follower.yaml --wait

helm install beacon-follower prysm  --values ./prysm/values-multi-follower-beacon.yaml --wait
helm install validator-follower prysm  --values ./prysm/values-multi-follower-validator.yaml 
```

```
helm uninstall geth
helm uninstall beacon
helm uninstall validator

helm uninstall geth-follower
helm uninstall beacon-follower
helm uninstall validator-follower
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