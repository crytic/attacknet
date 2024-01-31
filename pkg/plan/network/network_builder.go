package network

import (
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
)

func clientListsToMaps(execClients, consClients []ClientVersion) (execClientMap, consClientMap map[string]ClientVersion, err error) {
	populateClientMap := func(li []ClientVersion) (map[string]ClientVersion, error) {
		clients := make(map[string]ClientVersion)
		for _, client := range li {
			_, exists := clients[client.Name]
			if exists {
				return nil, stacktrace.NewError("duplicate configuration for client %s", client.Name)
			}
			clients[client.Name] = client
		}
		return clients, nil
	}

	execClientMap, err = populateClientMap(execClients)
	if err != nil {
		return nil, nil, err
	}

	consClientMap, err = populateClientMap(consClients)
	if err != nil {
		return nil, nil, err
	}

	return execClientMap, consClientMap, nil
}

func ComposeNetworkTopology(topology Topology, clientUnderTest string, execClients, consClients []ClientVersion) ([]*Node, error) {
	if clientUnderTest == "all" {
		return nil, stacktrace.NewError("target clientUnderTest 'all' not supported yet")
	}

	isExecutionClient := false
	for _, execClient := range execClients {
		if execClient.Name == clientUnderTest {
			isExecutionClient = true
			break
		}
	}

	execClientMap, consClientMap, err := clientListsToMaps(execClients, consClients)
	if err != nil {
		return nil, err
	}

	var nodes []*Node
	bootnode, err := composeBootnode(topology.BootnodeEL, topology.BootnodeCl, execClientMap, consClientMap)
	if err != nil {
		return nil, err
	}
	nodes = append(nodes, bootnode)

	// assume already checked clientUnderTest is a member of consClients or execClients
	var nodesToTest []*Node
	if isExecutionClient {
		nodesToTest, err = composeExecTesterNetwork(clientUnderTest, execClientMap, consClientMap)
	} else {
		nodesToTest, err = composeConsensusTesterNetwork(clientUnderTest, execClientMap, consClientMap)
	}
	if err != nil {
		return nil, err
	}
	nodes = append(nodes, nodesToTest...)

	// add more nodes to the network to satisfy target percent threshold
	extraNodes, err := composeNodesToSatisfyTargetPercent(
		topology.TargetsAsPercentOfNetwork,
		len(nodes)-1,
		clientUnderTest,
		execClients,
		consClients,
	)
	if err != nil {
		return nil, err
	}
	nodes = append(nodes, extraNodes...)
	return nodes, nil
}

func composeNodesToSatisfyTargetPercent(percentTarget float32, targetedNodeCount int, clientUnderTest string, execClients, consClients []ClientVersion) ([]*Node, error) {
	// percent target is unconfigured
	if percentTarget == 0 {
		return []*Node{}, nil
	}

	nodesToAdd, err := calcNodesNeededToSatisfyTarget(percentTarget, targetedNodeCount)
	if err != nil {
		return nil, err
	}

	startNodeIndex := targetedNodeCount + 1
	nodes, err := pickExtraNodeClients(startNodeIndex, nodesToAdd, clientUnderTest, execClients, consClients)
	return nodes, err
}

func pickExtraNodeClients(startNodeIndex, nodeCount int, clientUnderTest string, execClients, consClients []ClientVersion) ([]*Node, error) {
	var nodes []*Node

	execClientIndex := 0
	consClientIndex := 0
	for i := 0; i < nodeCount; i++ {
		var execClient, consClient ClientVersion
		var err error
		execClient, execClientIndex, err = pickClient(execClientIndex, clientUnderTest, execClients)
		if err != nil {
			return nil, err
		}
		consClient, consClientIndex, err = pickClient(consClientIndex, clientUnderTest, consClients)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, buildNode(startNodeIndex, execClient, consClient))
		startNodeIndex += 1
	}
	return nodes, nil
}

func pickClient(startIndex int, clientUnderTest string, clients []ClientVersion) (ClientVersion, int, error) {
	for i := 0; i < len(clients); i++ {
		c := clients[startIndex]

		startIndex += 1
		if startIndex >= len(clients) {
			startIndex = 0
		}

		if c.Name != clientUnderTest {
			return c, startIndex, nil
		}
	}
	return ClientVersion{}, 0, stacktrace.NewError("Unable to find any clients defined other than %s. Cannot add more nodes.", clientUnderTest)
}

func calcNodesNeededToSatisfyTarget(percentTarget float32, targetedNodeCount int) (int, error) {
	if percentTarget > 1.0 || percentTarget < 0 {
		return 0, stacktrace.NewError("invalid value for targets_as_percent_of_network, must be >=0 and < 1")
	}
	//if percentTarget > 0.9

	networkSize := float32(targetedNodeCount) / percentTarget
	if networkSize-float32(targetedNodeCount) < 1 {
		return 0, stacktrace.NewError("unable to compose a network where targeted nodes are %.2f of the network. The presence of the bootnode prevents this value from exceeding %.2f", percentTarget, float32(targetedNodeCount)/(float32(targetedNodeCount)+1))
	}

	if percentTarget <= 0.30 {
		log.Warnf("The currently configured targets_as_percent_of_network of %.2f will create a network of %d nodes", percentTarget, int(networkSize))
	} else {
		log.Infof("The currently configured targets_as_percent_of_network of %.2f will create a network of %d nodes", percentTarget, int(networkSize))

	}

	nodesToAdd := int(networkSize) - targetedNodeCount - 1
	return nodesToAdd, nil
}
