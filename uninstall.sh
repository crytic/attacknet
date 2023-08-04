#!/bin/bash

helm uninstall geth
helm uninstall beacon
helm uninstall validator

helm uninstall geth-follower
helm uninstall lighthouse-beacon
helm uninstall lighthouse-validator

helm uninstall geth-follower1
helm uninstall beacon-follower1
helm uninstall validator-follower1

helm uninstall geth-follower2
helm uninstall beacon-follower2
helm uninstall validator-follower2

helm uninstall geth-follower3
helm uninstall beacon-follower3
helm uninstall validator-follower3

helm uninstall geth-follower4
helm uninstall beacon-follower4
helm uninstall validator-follower4

helm uninstall geth-follower5
helm uninstall beacon-follower5
helm uninstall validator-follower5

helm uninstall geth-follower6
helm uninstall beacon-follower6
helm uninstall validator-follower6