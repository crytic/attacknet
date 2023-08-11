# Run 1

Injecting -10m clock skew on lighthose-geth pods for 5m.

Beacon node and validator nodes fall on their face, they seem to be aware there's a time discrepancy. Whether this is because of how chaos mesh injects clock skew, or they see their clock time is lower than the time indicated by their beacon db is unknown.

# Run 2

Injecting -30s clock skew on lighthouse-geth pods for 5m.

Immediate errors, just like the -10m clock skew. 
Once 30s passed, the errors disappeared. This might be the business.
It seems like the node is still publishing attestations, but it's not clear if it's publishing them "late" because the delay is only 30s. 
It does appear that it's following late by several blocks.

# Run 3

Injecting -5m clock skew on lighthouse-geth pods for 20m.

It's definitely publishing attestations for old blocks after 5m elapses.

cl-6-prysm-geth pod terminated from OOM 9 minutes into the attack, 4 minutes after attestations started being broadcast
cl-3-prysm-geth terminated from oom a few mins later



# Run 4

Injecting -5m clock skew on lighthouse-geth pods for 45m.
start @ 2:41pm

Successful, we had two replay_blocks spikes from the prysm clients. 30 blocks for each client. (rate: 0.6) (2:54, 13ish minutes after start.)

Next, we should try with a longer delay. 30 blocks is roughly 5 minutes, so I think the longer the delay, the more blocks that will be forced through replay.

# Run 5

Injecting -10m clock skew on lighthouse-geth pods for 30m. We'll also start adding to the deposit queue to crank up the heat.

start: 4:28pm


