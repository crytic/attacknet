
Running on commit 000d78a2ac50a34871fe2291d13c08696edbc71f
16384 validators across 8 nodes.

Genesis at 9:12a
Start create-deposit to fill deposit indexes 16384-20000 at 9:27.

While running create-deposit, noticed cast is not giving the successive transactions the correct nonce, so only 1 deposit makes it into every block.

Fixed cast send to do correct nonce, re-launching createdeposit at 9:33.
I suspect blocks still aren't being filled though, it's been half an hour and we're only around 650 deposits.

pausing deposit creation at 10:11. modifying script to tell us new val offset and log what's taking so long.

I think it's the time to connect to the rpc.
re-launch at 10:13

finished at 10:20

start new deposit thing, index 20k to 30k.

10:53, 30k to 60k.


## new run

2pmish, same config as above

running deposits right now, but will start testing before deposits are done.

#### Attack 1
start 20 minute 2000ms latency attack against follower-0's geth instance.
geth is between 1-2 peers.
beacon chain still full peers.

prysm reporting: time="2023-07-27 21:08:14" level=info msg="Fallen behind peers; reverting to initial sync to catch up" currentEpoch=32 peersEpoch=32 prefix=sync syncedEpoch=30
and time="2023-07-27 21:09:23" level=warning msg="Block is not processed" error="could not process block: could not validate new payload: timeout from http.Client: received an undefined execution engine error" prefix=initial-sync

geth crashed due to unhealthy liveness/readiness probe.

replay blocks sum went up a bit along with various cache misses for the single impacted node.
after turning off the attack, geth still has issues with peers. might be a kubernetes config issue.

#### Attack 2

we'll reduce the latency to 500ms, same node. start at 2:13pm

5 mins later and I don't think the performance of the node is materially impacted.

it appears there's a finalization delay between 21:15 and 21:16. This may correspond to the impact node's recovery causing issues on the network.

Let's keep running with the 500ms delay for 10 mins to get everything into a steady state, then disable the attack and see if there's an impact.

2:27:
seeing another finalization lag around 2:22, which is fully recovered from a few mins later. 8 mins since last fault. could it be that this one node being slow to attest is causing finalization delay depending on what committee it's in?

2:28, attack stopped. wait 10 mins.

2:33 not seeing anything so far. I wonder if the latency delay is impacting prometheus in such a way that could cause this.

2:37 starting a new attack, 1000ms, same target. 15mins
not sure if it's working this time, hard to test.
beacon chain reported a late blog reorg attempt:
time="2023-07-27 21:39:08" level=info msg="Attempted late block reorg aborted due to attestations at 10 seconds" prefix=blockchain root=0x1c3ee5eabcf4bbe22d4a75035752a3f5c6ffd4fb32e90519db3a3d1efbff5857 weight=75552000000000

definitely starting to notice some geth timeouts.
beacon: Received nil payload ID on VALID engine response
geth: INFO [07-27|21:41:50.015] Ignoring beacon update to old head       number=344 hash=608a7f..51f88e age=16s   have=345
validator finally reports: Failed to request block from beacon node" blockSlot=360 error="rpc error: code = Internal desc = Could not set execution data: failed to get execution payload: could not get cached payload from execution client: timeout from http.Client" prefix=validator pubKey=0x9353f1b0b4fb
2:44, there's now a sync gap of about 20 blocks between exec and consensus nodes.
2:45 geth stops.

2:46, disabling attack.