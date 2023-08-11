# Run 1

Run 1000ms delay for 5m on lighthouse's EL node (just to see what happens)

Results:
lighthouse CL stall, stalled beacon head slot.
Unsure if this is the condition we need to trigger replay_blocks. 
Next run, run it for longer so we can capture an entire epoch.

Note: we cannot run these tests on node 1 since it's the bootnode. DUH!


# Run 2

btw: genesis to finality is 25m

second node is now lighthouse. we'll test it's EL instead of node 1 (bootnode)
1000ms, 10m on lighthouse-2 EL node

it's taking a bit longer for the CL to stall.
it seems like it's not stalling at all a few mins into the attack. restarting.

wash

# Run 3

same as above, but using 5000ms instead of 1000.

we're seeing cl failures now.
lighthouse BN now falling behind.

the validator client says it's publishing attestations. are these attestations targetting the old block? 

BN is now changed to a syncing state.

recovery looks clean, ugh. no replay blocks.

# Run 4

going to do this the same time as a create_deposit.sh run and bump the attack time to 15m
it seems like once the BN switches to syncing mode, there's a big spike in the lighthouse node's attestation processing time

looks like the network recovered successfully, no replay blocks.

I suspect deposits arent actually being processed until a certain number of blocks have passed.

# Run 5
Hail mary, let's try a packet loss attack. 75% for 10m.

EL client is starting to lag, peers dropping.

I don't think we're seeing any BN lag yet. EL client is lagged by 30ish blocks.
It just successfully advanced to the next epoch despite the EL being lagged.

finally seems to be lagging out the beacon head slot now that we're like 30 blocks behind for 5 mins.

the head slot is slowly advancing still. is this how we get it to attest to an old epoch w/ new block?
at the end of hte attack, all of the BN peers dropped. weird. had to recover using `kubectl get pod <pod_name> -n <namespace> -o yaml | kubectl replace --force -f -`

after recovering the BN pod, they're back to "okay its fine", but the EL is still syncing very slowly.  40ish block delay.

despite this delay, the finalized head is still advancing.
the new BN pod is staying at a single peer. this implies peer scoring for BN nodes is enabled and only the bootnode will tolerate it.

# Run 6

let's try 200ms latency/50ms jitter on all EL nodes in the cluster with the eception of the bootnode, 15 mins.

nothing happened, network proceeded as normal. Going to turn this up to 2000ms next run to see if chaos mesh is working

# Run 7

2000ms latency, 50ms jitter, all EL nodes except bootnode.

the EL/CL node from run 5 was still lagging and almost immediately started to fall behind the head. 

the other nodes took a bit, but soon started to fall behind as well. 

it would be interesting to see how this goes with more jitter

# Run 8

1000ms latency, 1000ms jitter, all EL nodes except bootnode, 15m, 2:03p
