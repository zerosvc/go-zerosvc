package zerosvc

import (
	"github.com/fxamacker/cbor/v2"
	uuid "github.com/satori/go.uuid"
)

func NewNode(config Config, transport Transport) (*Node, error) {
	n := Node{
		Name:      config.NodeName,
		UUID:      config.NodeUUID,
		Services:  map[string]Service{},
		e:         config.Encoder,
		d:         config.Decoder,
		autoTrace: true,
	}
	if len(config.NodeUUID) == 0 {
		n.UUID = uuid.NewV5(namespace, n.Name).String()
	}
	if config.Encoder == nil {
		n.e, _ = cbor.CanonicalEncOptions().EncMode()
	}
	if config.Decoder == nil {
		n.d, _ = cbor.DecOptions{}.DecMode()
	}

	return &n, nil
}
