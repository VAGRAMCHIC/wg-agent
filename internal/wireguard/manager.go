package wireguard

import (
	"fmt"
	"net"
	"sync"

	bolt "go.etcd.io/bbolt"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type IPPool struct {
	Network *net.IPNet
	Used    map[string]bool
	mu      sync.Mutex
	db      *bolt.DB
}

func NewIPPool(cidr string, dbFile string) (*IPPool, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, err
	}

	pool := &IPPool{
		Network: network,
		Used:    make(map[string]bool),
		db:      db,
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("ips"))
		return err
	})
	if err != nil {
		return nil, err
	}

	// загружаем уже занятые IP
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("ips"))
		return b.ForEach(func(k, v []byte) error {
			pool.Used[string(k)] = true
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func (p *IPPool) Allocate() (net.IP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ip := p.Network.IP.Mask(p.Network.Mask)
	for {
		ip = nextIP(ip)
		if !p.Network.Contains(ip) {
			break
		}
		if !p.Used[ip.String()] {
			p.Used[ip.String()] = true
			err := p.db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("ips"))
				return b.Put([]byte(ip.String()), []byte("allocated"))
			})
			if err != nil {
				return nil, err
			}
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no free IPs")
}

func (p *IPPool) Release(ip net.IP) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.Used, ip.String())
	p.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("ips"))
		return b.Delete([]byte(ip.String()))
	})
}

func nextIP(ip net.IP) net.IP {
	ip = ip.To4()
	next := make(net.IP, len(ip))
	copy(next, ip)
	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	return next
}

type Manager struct {
	iface  string
	client *wgctrl.Client
	ipPool *IPPool
}

func New(iface string) (*Manager, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	pool, err := NewIPPool("10.0.1.0/24", "bolt.db")
	if err != nil {
		return nil, err
	}

	return &Manager{
		iface:  iface,
		client: client,
		ipPool: pool,
	}, nil
}

func (m *Manager) ServerPublicKey() string {
	dev, _ := m.client.Device(m.iface)
	return dev.PublicKey.String()
}

func (m *Manager) Endpoint() string {
	return "vpn.example.com:51820" // можно брать из ENV
}

func (m *Manager) AddPeer(publicKey string, allowedIP string) error {
	key, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return err
	}

	_, ipnet, err := net.ParseCIDR(allowedIP)
	if err != nil {
		return err
	}

	cfg := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:  key,
				AllowedIPs: []net.IPNet{*ipnet},
			},
		},
	}

	return m.client.ConfigureDevice(m.iface, cfg)
}

func (m *Manager) RemovePeer(publicKey string) error {
	key, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return err
	}

	dev, _ := m.client.Device(m.iface)
	for _, p := range dev.Peers {
		if p.PublicKey.String() == publicKey && len(p.AllowedIPs) > 0 {
			m.ipPool.Release(p.AllowedIPs[0].IP)
		}
	}

	cfg := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey: key,
				Remove:    true,
			},
		},
	}

	return m.client.ConfigureDevice(m.iface, cfg)
}

func (m *Manager) ListPeers() ([]wgtypes.Peer, error) {
	dev, err := m.client.Device(m.iface)
	if err != nil {
		return nil, err
	}
	return dev.Peers, nil
}

func (m *Manager) AddPeerAuto(publicKey string) (net.IP, error) {
	ip, err := m.ipPool.Allocate()
	if err != nil {
		return nil, err
	}

	err = m.AddPeer(publicKey, ip.String()+"/32")
	if err != nil {
		m.ipPool.Release(ip)
		return nil, err
	}

	return ip, nil
}