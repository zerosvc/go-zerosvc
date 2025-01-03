package zerosvc

import (
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
	"time"
)

var namespace, _ = uuid.FromString(`63082cd1-0f91-48cd-923a-f1523a26549b`)

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
	// encoder. CBOR will be used if not specified. Tags on builtin structs are only prepared for JSON/CBOR so other encoders might generate a bit longer tags
	Encoder Encoder
	// decoder. CBOR will be used if not specified. Tags on builtin structs are only prepared for JSON/CBOR so other encoders might generate a bit longer tags
	Decoder Decoder
	// what prefix will be added to event path. trailing / not required
	EventRoot         string
	HeartbeatInterval time.Duration
	Logger            *zap.SugaredLogger
}

type Encoder interface {
	Marshal(v any) ([]byte, error)
}
type Decoder interface {
	Unmarshal(data []byte, v any) error
}

type Transport interface {
	Publish(Message) error
	Subscribe(topic string, data chan *Message) error
	// Connect will be called once initially. Transport is the one that should handle reconnections
	Connect(hooks Hooks, willTopic string) error
	HeartbeatMessage(m Message) error
}

type Event struct {
	TraceID   []byte         `cbor:"trace_id" json:"trace_id"`
	SpanID    []byte         `cbor:"span_id" json:"span_id"`
	NodeUUID  string         `cbor:"nuuid" json:"nuuid"`
	NodeName  string         `cbor:"node" json:"node"`
	TS        time.Time      `cbor:"ts" json:"ts"`
	ReplyTo   string         `cbor:"rt" json:"rt"`
	Headers   map[string]any `cbor:"headers" json:"headers"`
	Signature []byte         `cbor:"-" json:"-"`
	Body      []byte         `cbor:"b" json:"b"`
	retain    bool
	n         *Node
}

type Service struct {
	Ok   bool   `json:"ok" cbor:"ok"`
	Info string `json:"info,omitempty" cbor:"info,omitempty"`
	Data any    `json:"data,omitempty" cbor:"data,omitempty"`
}

type ErrSignatureInvalid struct{}

func (e ErrSignatureInvalid) Error() string {
	return "signature invalid"
}
