# Attacknet

## Getting started

1. Set up a containerd k8s cluster.
2. Authenticate to the cluster for kubectl
3. Install chaos-mesh
   1. `kubectl create ns chaos-mesh`
   2. `helm repo add chaos-mesh https://charts.chaos-mesh.org`
   3. `helm install chaos-mesh chaos-mesh/chaos-mesh -n=chaos-mesh --version 2.6.1 --set chaosDaemon.runtime=containerd --set chaosDaemon.socketPath=/run/containerd/containerd.sock --set dashboard.securityMode=false`
   4. To access chaos dashboard, use `kubectl --namespace chaos-mesh port-forward svc/chaos-dashboard 2333`
4. Install kurtosis locally.
5. Run `kurtosis cluster set cloud`
6. In a separate terminal, run `kurtosis gateway`
7. `cd kurtosis`
8. Run `make genesis` to start the network
9. Add chaos using the gui for now.




Ignore everything below this line, it's for the old helm deploy



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
