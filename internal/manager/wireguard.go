package manager

import (
	"context"
	"net"
	"os/exec"

	"github.com/VAGRAMCHIC/wg-agent/pkg/logger"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Manager struct {
	iface  string
	client *wgctrl.Client
	log    *logger.Logger
}

func New(iface string, log *logger.Logger) (*Manager, error) {

	c, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	return &Manager{
		iface:  iface,
		client: c,
		log:    log,
	}, nil
}

func (m *Manager) EnsureInterface(ctx context.Context, addr string) error {

	if _, err := net.InterfaceByName(m.iface); err == nil {
		m.log.Info(ctx, "wg_interface_exists", map[string]interface{}{
			"iface": m.iface,
		})
		return nil
	}

	m.log.Info(ctx, "creating_wireguard_interface", map[string]interface{}{
		"iface": m.iface,
	})

	cmd := exec.Command("ip", "link", "add", m.iface, "type", "wireguard")
	if err := cmd.Run(); err != nil {
		m.log.Error(ctx, "failed_create_interface", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	cmd = exec.Command("ip", "addr", "add", addr, "dev", m.iface)
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("ip", "link", "set", m.iface, "up")
	return cmd.Run()
}

func (m *Manager) AddPeer(ctx context.Context, pubKey string, allowedIP string) error {

	key, err := wgtypes.ParseKey(pubKey)
	if err != nil {
		return err
	}

	_, ipNet, err := net.ParseCIDR(allowedIP)
	if err != nil {
		return err
	}

	cfg := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:         key,
				ReplaceAllowedIPs: true,
				AllowedIPs:        []net.IPNet{*ipNet},
			},
		},
	}

	err = m.client.ConfigureDevice(m.iface, cfg)

	if err != nil {
		m.log.Error(ctx, "failed_add_peer", map[string]interface{}{
			"public_key": pubKey,
			"ip":         allowedIP,
			"error":      err.Error(),
		})
		return err
	}

	m.log.Info(ctx, "peer_added", map[string]interface{}{
		"public_key": pubKey,
		"ip":         allowedIP,
	})

	return nil
}

func (m *Manager) RemovePeer(ctx context.Context, pubKey string) error {

	key, err := wgtypes.ParseKey(pubKey)
	if err != nil {
		return err
	}

	cfg := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey: key,
				Remove:    true,
			},
		},
	}

	err = m.client.ConfigureDevice(m.iface, cfg)

	if err != nil {
		m.log.Error(ctx, "failed_remove_peer", map[string]interface{}{
			"public_key": pubKey,
			"error":      err.Error(),
		})
		return err
	}

	m.log.Info(ctx, "peer_removed", map[string]interface{}{
		"public_key": pubKey,
	})

	return nil
}

func (m *Manager) ListPeers(ctx context.Context) ([]map[string]interface{}, error) {

	dev, err := m.client.Device(m.iface)
	if err != nil {
		m.log.Error(ctx, "failed_list_peers", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	var peers []map[string]interface{}

	for _, p := range dev.Peers {

		var ips []string

		for _, ip := range p.AllowedIPs {
			ips = append(ips, ip.String())
		}

		peers = append(peers, map[string]interface{}{
			"public_key":  p.PublicKey.String(),
			"allowed_ips": ips,
		})
	}

	return peers, nil
}
