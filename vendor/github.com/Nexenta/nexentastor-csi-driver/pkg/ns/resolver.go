package ns

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// Resolver - NexentaStor cluster API provider
type Resolver struct {
	Nodes []ProviderInterface
	Log   *logrus.Entry
}

// Resolve - get one NS from the list of NSs by provided pool/dataset/fs path
func (nsr *Resolver) Resolve(path string) (resolvedNS ProviderInterface, lastError error) {
	l := nsr.Log.WithField("func", "Resolve()")

	if path == "" {
		return nil, fmt.Errorf("Resolved was called with empty pool/dataset path")
	}

	//TODO do non-block requests to all NSs in the list, select first one responded
	for _, ns := range nsr.Nodes {
		_, err := ns.GetFilesystem(path)
		if err != nil {
			lastError = err
		} else {
			resolvedNS = ns
			break
		}
	}

	if resolvedNS != nil {
		l.Debugf("resolve '%s' to '%s'", path, resolvedNS)
		return resolvedNS, nil
	}

	message := fmt.Sprintf("No NexentaStor(s) found with pool/dataset: '%s'", path)
	if lastError != nil {
		return nil, fmt.Errorf("%s, last error: %s", message, lastError)
	}
	return nil, fmt.Errorf(message)
}

// IsCluster - check if nodes is a NS cluster
// For now it simple checks if all nodes return at least one similar cluster name
func (nsr *Resolver) IsCluster() (bool, error) {
	l := nsr.Log.WithField("func", "IsCluster()")

	if len(nsr.Nodes) < 2 {
		return false, nil
	}

	names := map[string]int{
		// ClusterName: FindOnNodeCount
	}

	for _, node := range nsr.Nodes {
		// get RSF cluster from each node
		clusters, err := node.GetRSFClusters()
		if err != nil {
			return false, err
		}
		for _, cluster := range clusters {
			if v, ok := names[cluster.Name]; ok {
				names[cluster.Name] = v + 1
			} else {
				names[cluster.Name] = 1
			}
		}
	}

	for clusterName, findOnNodeCount := range names {
		if findOnNodeCount == len(nsr.Nodes) {
			l.Infof("all nodes belong to '%s' cluster", clusterName)
			return true, nil
		}
	}

	return false, nil
}

// ResolverArgs - params to create resolver instance from config
type ResolverArgs struct {
	Address  string
	Username string
	Password string
	Log      *logrus.Entry
}

// NewResolver - create NexentaStor resolver instance based on configuration
func NewResolver(args ResolverArgs) (nsr *Resolver, err error) {
	if len(args.Address) == 0 {
		return nil, fmt.Errorf("NexentaStor address not specified: %s", args.Address)
	}

	l := args.Log.WithFields(logrus.Fields{
		"cmp": "NSResolver",
		"ns":  args.Address,
	})

	l.Debugf("created for %s", args.Address)

	var nodes []ProviderInterface
	addressList := strings.Split(args.Address, ",")
	for _, address := range addressList {
		nsProvider, err := NewProvider(ProviderArgs{
			Address:  address,
			Username: args.Username,
			Password: args.Password,
			Log:      l,
		})
		if err != nil {
			return nil, fmt.Errorf("Cannot create provider for %s NexentaStor: %s", address, err)
		}
		nodes = append(nodes, nsProvider)
	}

	nsr = &Resolver{
		Nodes: nodes,
		Log:   l,
	}

	return nsr, nil
}
