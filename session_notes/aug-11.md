# Run 1

Injecting -10m clock skew on lighthouse-geth pods for 30m. We'll also start adding to the deposit queue to crank up the heat.



Start around 9:10a.

replay_blocks happens around 9:22, 60 block replay per node suggests the number of blocks that need to be replayed is directly correlated to the initial clock skew.

cl-4 prysm node stack mem spiked to about 1.6gi, but only that node. the others appears to have not been impacted, oddly enough. Something to keep an eye on.


# Run 2

We'll presaturate the deposit queue with at least 10-15k deposits and run the deposit daemon during the entire test.

We'll also turn off peer scoring to allow us to re-use the network after the impacted node gets down scored.

For this run, I'd like to try -40m clock skew. I think the easiest way to do this is to wait until the network is at least 40m old, restart the lighthouse node & instantly start a chaos test to skew its clock. Alternately, maybe we should adjust the block rate down to a few seconds.

Start @ 10:03am

Replay blocks rate hit 1.25, which is kinda in line with the previous -10m test. This suggests there may be limitations as to how well this attack works.

todo: we should set min peers to 1 for lighthouse