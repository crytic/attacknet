package network

import (
	"github.com/kurtosis-tech/stacktrace"
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

func ComposeNetworkTopology(bootEl, bootCl, client string, execClients, consClients []ClientVersion) ([]*Node, error) {
	if client == "all" {
		return nil, stacktrace.NewError("target client 'all' not supported yet")
	}

	isExecutionClient := false
	for _, execClient := range execClients {
		if execClient.Name == client {
			isExecutionClient = true
			break
		}
	}
	// assume already checked client is a member of consClients or execClients
	if isExecutionClient {
		return composeExecTesterNetwork(bootEl, bootCl, client, execClients, consClients)
	} else {
		return composeConsensusTesterNetwork(bootEl, bootCl, client, execClients, consClients)
	}
}
