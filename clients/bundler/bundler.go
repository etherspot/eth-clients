package bundler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/marioevz/eth-clients/clients"
	"github.com/marioevz/eth-clients/clients/utils"
	"github.com/protolambda/eth2api"
	"github.com/protolambda/eth2api/client/nodeapi"
)

const (
	PortBundlerTCP = 4337
	PortBundlerUDP = 4337
	PortBundlerAPI = 14337
)

type BundlerClientConfig struct {
	ClientIndex    int
	BundlerAPIPort int
	Mempoolnet     string
}

type BundlerClient struct {
	clients.Client
	Logger  utils.Logging
	Config  BundlerClientConfig
	Builder interface{}

	api *eth2api.Eth2HttpClient
}

func (bn *BundlerClient) Logf(format string, values ...interface{}) {
	if l := bn.Logger; l != nil {
		l.Logf(format, values...)
	}
}

func (bn *BundlerClient) Start() error {
	if !bn.IsRunning() {
		if managedClient, ok := bn.Client.(clients.ManagedClient); !ok {
			return fmt.Errorf("attempted to start an unmanaged client")
		} else {
			if err := managedClient.Start(); err != nil {
				return err
			}
		}

	}

	return bn.Init(context.Background())
}

func (bn *BundlerClient) Init(ctx context.Context) error {
	if bn.api == nil {
		port := bn.Config.BundlerAPIPort
		if port == 0 {
			port = PortBundlerAPI
		}
		bn.api = &eth2api.Eth2HttpClient{
			Addr: fmt.Sprintf(
				"http://%s:%d",
				bn.GetIP(),
				port,
			),
			Cli:   &http.Client{},
			Codec: eth2api.JSONCodec{},
		}
	}
	return nil
}

func (bn *BundlerClient) Shutdown() error {
	if managedClient, ok := bn.Client.(clients.ManagedClient); !ok {
		return fmt.Errorf("attempted to shutdown an unmanaged client")
	} else {
		return managedClient.Shutdown()
	}
}

func (bn *BundlerClient) ENR(parentCtx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Second*10)
	defer cancel()
	var out eth2api.NetworkIdentity
	if err := nodeapi.Identity(ctx, bn.api, &out); err != nil {
		return "", err
	}
	return out.ENR, nil
}

func (bn *BundlerClient) P2PAddr(parentCtx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Second*10)
	defer cancel()
	var out eth2api.NetworkIdentity
	if err := nodeapi.Identity(ctx, bn.api, &out); err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"/ip4/%s/tcp/%d/p2p/%s",
		bn.GetIP().String(),
		PortBundlerTCP,
		out.PeerID,
	), nil
}

func (bn *BundlerClient) BeaconAPIURL() (string, error) {
	if bn.api == nil {
		return "", fmt.Errorf("api not initialized")
	}
	return bn.api.Addr, nil
}

func (bn *BundlerClient) EnodeURL() (string, error) {
	return "", errors.New(
		"bundler node does not have an discv4 Enode URL, use ENR or multi-address instead",
	)
}

func (bn *BundlerClient) ClientName() string {
	name := bn.ClientType()
	return name
}

func (bn *BundlerClient) API() *eth2api.Eth2HttpClient {
	return bn.api
}

type BundlerClients []*BundlerClient

// Return subset of clients that are currently running
func (all BundlerClients) Running() BundlerClients {
	res := make(BundlerClients, 0)
	for _, bc := range all {
		if bc.IsRunning() {
			res = append(res, bc)
		}
	}
	return res
}

// Return subset of clients that are part of an specific subnet
func (all BundlerClients) Subnet(subnet string) BundlerClients {
	if subnet == "" {
		return all
	}
	res := make(BundlerClients, 0)
	return res
}

// Returns comma-separated ENRs of all running beacon nodes
func (beacons BundlerClients) ENRs(parentCtx context.Context) (string, error) {
	if len(beacons) == 0 {
		return "", nil
	}
	enrs := make([]string, 0)
	for _, bn := range beacons {
		if bn.IsRunning() {
			enr, err := bn.ENR(parentCtx)
			if err != nil {
				return "", err
			}
			enrs = append(enrs, enr)
		}
	}
	return strings.Join(enrs, ","), nil
}

// Returns comma-separated P2PAddr of all running beacon nodes
func (beacons BundlerClients) P2PAddrs(
	parentCtx context.Context,
) (string, error) {
	if len(beacons) == 0 {
		return "", nil
	}
	staticPeers := make([]string, 0)
	for _, bn := range beacons {
		if bn.IsRunning() {
			p2p, err := bn.P2PAddr(parentCtx)
			if err != nil {
				return "", err
			}
			staticPeers = append(staticPeers, p2p)
		}
	}
	return strings.Join(staticPeers, ","), nil
}
