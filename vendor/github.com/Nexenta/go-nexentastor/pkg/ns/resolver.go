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

// Resolve returns one NS from the list of NSs by provided pool/dataset/fs path
func (r *Resolver) Resolve(path string) (ProviderInterface, error) {
	l := r.Log.WithField("func", "Resolve()")

	if path == "" {
		return nil, fmt.Errorf("Resolved was called with empty pool/dataset path")
	}

	//TODO do non-block requests to all NSs in the list, select first one responded
	var nefError error
	var resolvedNS ProviderInterface
	for _, ns := range r.Nodes {
		_, err := ns.GetFilesystem(path)
		if err != nil {
			nefError = err
		} else {
			resolvedNS = ns
			break
		}
	}

	if resolvedNS != nil {
		l.Debugf("resolve '%s' to '%s'", path, resolvedNS)
		return resolvedNS, nil
	}

	if nefError != nil {
		l.Debugf("error while resolving '%s' to '%s': %s", path, resolvedNS, nefError)
		return nil, nefError
	}

	l.Debugf("no NexentaStor(s) found with pool/dataset: '%s'", path)
	return nil, nil
}

// IsCluster checks if nodes is a NS cluster
// For now it simple checks if all nodes return at least one similar cluster name
func (r *Resolver) IsCluster() (bool, error) {
	l := r.Log.WithField("func", "IsCluster()")

	if len(r.Nodes) < 2 {
		return false, nil
	}

	names := map[string]int{
		// "ClusterName": "FindOnNodeCount"
	}

	for _, node := range r.Nodes {
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
		if findOnNodeCount == len(r.Nodes) {
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

	// InsecureSkipVerify controls whether a client verifies the server's certificate chain and host name.
	InsecureSkipVerify bool
}

// NewResolver creates NexentaStor resolver instance based on configuration
func NewResolver(args ResolverArgs) (*Resolver, error) {
	l := args.Log.WithFields(logrus.Fields{
		"cmp": "NSResolver",
		"ns":  args.Address,
	})

	if args.Address == "" {
		return nil, fmt.Errorf("NexentaStor address not specified: %s", args.Address)
	}

	var nodes []ProviderInterface
	addressList := strings.Split(args.Address, ",")
	for _, address := range addressList {
		nsProvider, err := NewProvider(ProviderArgs{
			Address:            address,
			Username:           args.Username,
			Password:           args.Password,
			Log:                l,
			InsecureSkipVerify: args.InsecureSkipVerify,
		})
		if err != nil {
			return nil, fmt.Errorf("Cannot create provider for %s NexentaStor: %s", address, err)
		}
		nodes = append(nodes, nsProvider)
	}

	l.Debugf("created for '%s'", args.Address)
	return &Resolver{
		Nodes: nodes,
		Log:   l,
	}, nil
}
