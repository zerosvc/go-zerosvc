package zerosvc

import (
	"bytes"
	"fmt"
)

// Serialize serializes event into binary blob. Format:
//
//	<sig_length:uint8><signature:bytes><serialized event>
//
// If signing is disabled packet starts with 0x00 and event immediately after.
func (e *Event) Serialize() (out []byte, err error) {
	data, err := e.n.e.Marshal(e)
	if err != nil {
		return
	}
	sigLength := uint8(0)
	signature := []byte{}
	if e.n.Signer != nil {
		signature := e.n.Signer.Sign(out)
		sigLength = uint8(len(signature))
		if sigLength < 8 {
			return out, fmt.Errorf("signing function defined but signature is empty")
		}
	}
	b := bytes.Buffer{}
	b.WriteByte(sigLength)
	b.Write(signature)
	b.Write(data)
	return b.Bytes(), nil
}

func (e *Event) Deserialize(in []byte, node *Node) (ev *Event, err error) {
	if len(in) < 4 {
		return nil, fmt.Errorf("event data too short")
	}
	sigLength := uint8(in[0])
	if len(in) < (4 + int(sigLength)) {
		return nil, fmt.Errorf("event data too short after signature")
	}
	signature := in[1 : sigLength+1]
	data := in[1+sigLength:]
	if sigLength > 0 {
		valid := node.Signer.Verify(signature, data)
		if !valid {
			return nil, ErrSignatureInvalid{}
		}
	}
	err = node.d.Unmarshal(in, ev)
	return ev, err
}
