package protoannotate

import (
	"fmt"
	"google.golang.org/protobuf/encoding/protowire"
	"io"
	"strings"
	"unicode/utf8"
)

type Encoder struct {
	w io.Writer
	indent int
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

func (enc *Encoder) Encode(b []byte) error {
	for len(b) > 0 {
		n, err := enc.encodeField(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

func (enc *Encoder) encodeField(b []byte) (int, error) {
	if err := enc.writeIndent(); err != nil {
		return 0, err
	}
	num, typ, n := protowire.ConsumeTag(b)
	if n < 0 {
		err := enc.printf("// error: unknown tag: ")
		if err != nil {
			return 0, err
		}
		err = enc.writeBytes(b[:4])
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("unknown tag")
	}
	if err := enc.writeBytes(b[:n]); err != nil {
		return 0, err
	}
	if err := enc.printf(" // field id=%d type=%s\n", num, typeToString(typ)); err != nil {
		return 0, err
	}
	m, err := enc.encodeFieldValue(typ, b[n:])
	if err != nil {
		return 0, err
	}
	return n + m, nil
}

func (enc *Encoder) encodeFieldValue(typ protowire.Type, b []byte) (int, error) {
	if err := enc.writeIndent(); err != nil {
		return 0, err
	}
	var n int
	var val string
	switch typ {
	case protowire.VarintType:
		var v uint64
		v, n = protowire.ConsumeVarint(b)
		if n < 0 {
			return 0, fmt.Errorf("failed to parse varint")
		}
		val = fmt.Sprintf("%d", v)
	case protowire.Fixed32Type:
		var v uint32
		v, n = protowire.ConsumeFixed32(b)
		if n < 0 {
			return 0, fmt.Errorf("failed to parse fixed32")
		}
		val = fmt.Sprintf("%d", v)
	case protowire.Fixed64Type:
		var v uint64
		v, n = protowire.ConsumeFixed64(b)
		if n < 0 {
			return 0, fmt.Errorf("failed to parse fixed64")
		}
		val = fmt.Sprintf("%d", v)
	case protowire.BytesType:
		var v []byte
		v, n = protowire.ConsumeBytes(b)
		if n < 0 {
			return 0, fmt.Errorf("failed to parse bytes")
		}
		if utf8.Valid(v) {
			val = string(v)
		} else {
			val = "<non-utf8 byte array>"
		}
	case protowire.StartGroupType:
		enc.indent += 1
		n = 0
		val = "Start Group"
	case protowire.EndGroupType:
		enc.indent -= 1
		n = 0
		val = "End Group"
	default:
		return 0, fmt.Errorf("invalid type: %d", typ)
	}
	if err := enc.writeBytes(b[:n]); err != nil {
		return 0, err
	}
	if err := enc.printf(" // %s\n", val); err != nil {
		return 0, err
	}
	return n, nil
}

func (enc *Encoder) writeBytes(b []byte) error {
	first := true
	for i := range b {
		if first {
			first = false
			if err := enc.printf("%02x", b[i]); err != nil {
				return err
			}
		} else {
			if err := enc.printf(" %02x", b[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (enc *Encoder) writeIndent() error {
	return enc.printf(strings.Repeat(" ", 2*enc.indent))
}

func (enc *Encoder) printf(format string, args ...interface{}) error {
	_, err := fmt.Fprintf(enc.w, format, args...)
	return err
}

func typeToString(typ protowire.Type) string {
	switch typ {
	case protowire.VarintType:
		return "Varint"
	case protowire.Fixed32Type:
		return "Fixed32"
	case protowire.Fixed64Type:
		return "Fixed64"
	case protowire.BytesType:
		return "Bytes"
	case protowire.StartGroupType:
		return "StartGroup"
	case protowire.EndGroupType:
		return "EndGroup"
	default:
		return "<Unknown>"
	}
}