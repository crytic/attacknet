#!/bin/bash

install_boot_node() {
    helm install geth geth --values ./geth/values-multi-bootnode.yaml --wait
    helm install beacon prysm --values ./prysm/values-multi-leader-beacon.yaml --wait
    helm install validator prysm --values ./prysm/values-multi-leader-validator.yaml 
}

install_prysm_node() {
    helm install geth-follower$1 geth --values ./geth/values-multi-follower.yaml --wait
    helm install beacon-follower$1 prysm  --values ./prysm/values-multi-follower-beacon.yaml --values ./prysm/follower/$1-beacon.yaml --wait
    helm install validator-follower$1 prysm  --values ./prysm/values-multi-follower-validator.yaml --values  ./prysm/follower/$1-validator.yaml
}

install_lighthouse_node() {
    helm install geth-follower geth  --values ./geth/values-multi-follower.yaml --wait
    helm install lighthouse-beacon lighthouse  --values ./lighthouse/values-beacon.yaml --wait
    helm install lighthouse-validator lighthouse  --values ./lighthouse/values-validator.yaml 
}

install_boot_node

#install_lighthouse_node &

# install_prysm_node 1 &
#install_prysm_node 2 &
#install_prysm_node 3 &
#install_prysm_node 4 &
#install_prysm_node 5 &
#install_prysm_node 6 &
