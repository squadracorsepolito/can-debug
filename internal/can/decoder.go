package can

import (
	"context"

	"github.com/squadracorsepolito/acmelib"
)

type Decoder struct {
	m map[uint32]func([]byte) []*acmelib.SignalDecoding
}

func NewDecoder(messages []*acmelib.Message) *Decoder {
	m := make(map[uint32]func([]byte) []*acmelib.SignalDecoding)

	for _, msg := range messages {
		m[uint32(msg.GetCANID())] = msg.SignalLayout().Decode
	}

	return &Decoder{
		m: m,
	}
}

func (d *Decoder) Decode(ctx context.Context, canID uint32, data []byte) []*acmelib.SignalDecoding {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}

	fn, ok := d.m[canID]
	if !ok {
		return nil
	}
	return fn(data)
}
