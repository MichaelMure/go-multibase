package multibase

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"unicode/utf8"

	b32 "github.com/multiformats/go-base32"
)

// Encoder is a multibase encoding that is verified to be supported and
// supports an Encode method that does not return an error
type Encoder struct {
	enc Encoding
}

// NewEncoder create a new Encoder from an Encoding
func NewEncoder(base Encoding) (Encoder, error) {
	_, ok := EncodingToStr[base]
	if !ok {
		return Encoder{-1}, fmt.Errorf("unsupported multibase encoding: %d", base)
	}
	return Encoder{base}, nil
}

// MustNewEncoder is like NewEncoder but will panic if the encoding is
// invalid.
func MustNewEncoder(base Encoding) Encoder {
	_, ok := EncodingToStr[base]
	if !ok {
		panic("Unsupported multibase encoding")
	}
	return Encoder{base}
}

// EncoderByName creates an encoder from a string, the string can
// either be the multibase name or single character multibase prefix
func EncoderByName(str string) (Encoder, error) {
	var base Encoding
	var ok bool
	if len(str) == 0 {
		return Encoder{-1}, fmt.Errorf("empty multibase encoding")
	} else if utf8.RuneCountInString(str) == 1 {
		r, _ := utf8.DecodeRuneInString(str)
		base = Encoding(r)
		_, ok = EncodingToStr[base]
	} else {
		base, ok = Encodings[str]
	}
	if !ok {
		return Encoder{-1}, fmt.Errorf("unsupported multibase encoding: %s", str)
	}
	return Encoder{base}, nil
}

func (p Encoder) Encoding() Encoding {
	return p.enc
}

// Encode encodes the multibase using the given Encoder.
func (p Encoder) Encode(data []byte) string {
	str, err := Encode(p.enc, data)
	if err != nil {
		// should not happen
		panic(err)
	}
	return str
}

func (p Encoder) Writer(w io.Writer) io.WriteCloser {
	return newEncoderWriter(p.enc, w)
}

var _ io.WriteCloser = &encoderWriter{}

// encoderWriter is a multibase encoder that wraps and outputs to an io.Writer.
type encoderWriter struct {
	enc           Encoding
	out           io.Writer
	headerWritten bool
	processor     io.Writer
}

func newEncoderWriter(enc Encoding, out io.Writer) *encoderWriter {
	ew := &encoderWriter{
		enc:           enc,
		out:           out,
		headerWritten: false,
	}

	switch enc {
	case Identity:
		ew.processor = out
	case Base16:
		ew.processor = hex.NewEncoder(out)
	case Base32:
		ew.processor = b32.NewEncoder(base32StdLowerNoPad, out)
	case Base32Upper:
		ew.processor = b32.NewEncoder(base32StdUpperNoPad, out)
	case Base32hex:
		ew.processor = b32.NewEncoder(base32HexLowerNoPad, out)
	case Base32hexUpper:
		ew.processor = b32.NewEncoder(base32HexUpperNoPad, out)
	case Base32pad:
		ew.processor = b32.NewEncoder(base32StdLowerPad, out)
	case Base32padUpper:
		ew.processor = b32.NewEncoder(base32StdUpperPad, out)
	case Base32hexPad:
		ew.processor = b32.NewEncoder(base32HexLowerPad, out)
	case Base32hexPadUpper:
		ew.processor = b32.NewEncoder(base32HexUpperPad, out)
	case Base64pad:
		ew.processor = base64.NewEncoder(base64.StdEncoding, out)
	case Base64urlPad:
		ew.processor = base64.NewEncoder(base64.URLEncoding, out)
	case Base64url:
		ew.processor = base64.NewEncoder(base64.RawURLEncoding, out)
	case Base64:
		ew.processor = base64.NewEncoder(base64.RawStdEncoding, out)

	case Base2, Base16Upper, Base36, Base36Upper, Base58BTC, Base58Flickr, Base256Emoji:
		// unsupported
	}

	if ew.processor == nil {
		// implement fallback? or in Write() ?
	}

	return ew
}

func (ew encoderWriter) Write(p []byte) (n int, err error) {
	if ew.processor == nil {
		return -1, fmt.Errorf("unsupported encoding as writer")
	}
	if !ew.headerWritten {
		_, err = ew.out.Write([]byte{byte(ew.enc)})
		if err != nil {
			return 0, err
		}
		ew.headerWritten = true
	}
	return ew.processor.Write(p)
}

func (ew encoderWriter) Close() error {
	if closer, ok := ew.processor.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
