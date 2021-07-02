package zerosvc

import "fmt"

type Config struct {
	// Node name, preferably in fqdn@service[:instance] format
	NodeName string
	// Node UUID, will be generated automatically if not present
	NodeUUID string
	// New transport. New() will call Connect() on it, do NOT call Connect() before that as some
	// functions need pre-connection setup
	Transport Transport
	// Automatically setup heartbeat with transport-specific config
	AutoHeartbeat bool
	// AutoSigner will setup basic Ed25519 signatures.
	// passed function should:
	//
	// * return currently stored value if called with empty `new` parameter
	// * write whatever is in `new` if not empty
	AutoSigner func(new []byte) (old []byte)
	// function used to sign outgoing packets. XOR with AutoSigner.
	Signer Signer
	//

}

func New(cfg Config) (*Node, error) {
	var n *Node
	if len(cfg.NodeUUID) > 0 {
		n = NewNode(cfg.NodeName, cfg.NodeUUID)
	} else {
		n = NewNode(cfg.NodeName)
	}
	hbPath := n.HeartbeatPath()
	if cfg.AutoHeartbeat {
		cfg.Transport.SetupHeartbeat(hbPath)
	}
	if cfg.Signer != nil && cfg.AutoSigner != nil {
		return nil, fmt.Errorf("either pass Signer or AutoSigner, not both")
	}
	if cfg.AutoSigner != nil {
		key := cfg.AutoSigner([]byte{})
		if len(key) == 0 {
			s, err := NewSignerEd25519()
			if err != nil {
				return nil, fmt.Errorf("error generating key: %s", err)
			}
			cfg.AutoSigner(s.PrivateKey())
			n.Signer = s
		} else {
			s, err1 := NewSignerEd25519(key)
			if err1 != nil {
				s, err2 := NewSignerEd25519()
				if err2 != nil {
					return nil, fmt.Errorf("error retrieving key: %s and generating new one failed: %s", err1, err2)
				} else {
					n.Signer = s
				}
			} else {
				n.Signer = s
			}
		}
	}

	err := cfg.Transport.Connect()
	if err != nil {
		return nil, err
	}
	n.SetTransport(cfg.Transport)
	if cfg.AutoHeartbeat {
		go n.Heartbeater(hbPath)
	}
	return n, nil
}
