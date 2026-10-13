package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type txInput struct {
	PreviousTxID string `json:"previous_txid"`
	Vout         uint32 `json:"vout"`
	Sequence     uint32 `json:"sequence"`
	ScriptSig    string `json:"script_sig_hex"`
}
type txOutput struct {
	ValueSats int64  `json:"value_sats"`
	ScriptHex string `json:"script_pub_key_hex"`
}
type decodedTx struct {
	TxID        string     `json:"txid"`
	Version     int32      `json:"version"`
	LockTime    uint32     `json:"lock_time"`
	SegWit      bool       `json:"segwit"`
	InputCount  int        `json:"input_count"`
	OutputCount int        `json:"output_count"`
	Inputs      []txInput  `json:"inputs"`
	Outputs     []txOutput `json:"outputs"`
	Note        string     `json:"note"`
}

type txReader struct {
	raw []byte
	pos int
}

func (r *txReader) take(n int) ([]byte, error) {
	if n < 0 || n > len(r.raw)-r.pos {
		return nil, io.ErrUnexpectedEOF
	}
	b := r.raw[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}
func (r *txReader) u32() (uint32, error) {
	b, e := r.take(4)
	if e != nil {
		return 0, e
	}
	return binary.LittleEndian.Uint32(b), nil
}
func (r *txReader) u64() (uint64, error) {
	b, e := r.take(8)
	if e != nil {
		return 0, e
	}
	return binary.LittleEndian.Uint64(b), nil
}
func (r *txReader) varInt() (uint64, error) {
	b, e := r.take(1)
	if e != nil {
		return 0, e
	}
	switch b[0] {
	case 0xfd:
		v, e := r.take(2)
		if e != nil {
			return 0, e
		}
		n := uint64(binary.LittleEndian.Uint16(v))
		if n < 0xfd {
			return 0, fmt.Errorf("non-canonical varint")
		}
		return n, nil
	case 0xfe:
		v, e := r.u32()
		if e != nil {
			return 0, e
		}
		if v <= 0xffff {
			return 0, fmt.Errorf("non-canonical varint")
		}
		return uint64(v), nil
	case 0xff:
		v, e := r.u64()
		if e != nil {
			return 0, e
		}
		if v <= 0xffffffff {
			return 0, fmt.Errorf("non-canonical varint")
		}
		return v, nil
	default:
		return uint64(b[0]), nil
	}
}
func (r *txReader) varBytes() ([]byte, error) {
	n, e := r.varInt()
	if e != nil {
		return nil, e
	}
	if n > 4*1024*1024 {
		return nil, fmt.Errorf("script or witness item exceeds 4 MiB limit")
	}
	return r.take(int(n))
}
func reverseHex(b []byte) string {
	c := append([]byte(nil), b...)
	for i, j := 0, len(c)-1; i < j; i, j = i+1, j-1 {
		c[i], c[j] = c[j], c[i]
	}
	return hex.EncodeToString(c)
}
func doubleSHA256(b []byte) []byte {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:]
}

func DecodeTransaction(rawHex string, out io.Writer) error {
	rawHex = strings.TrimSpace(rawHex)
	if rawHex == "" {
		return fmt.Errorf("transaction hex is required")
	}
	if len(rawHex) > 8*1024*1024 {
		return fmt.Errorf("transaction hex exceeds 8 MiB safety limit")
	}
	raw, err := hex.DecodeString(rawHex)
	if err != nil {
		return fmt.Errorf("invalid transaction hex: %w", err)
	}
	if len(raw) < 10 {
		return fmt.Errorf("transaction is too short")
	}
	r := &txReader{raw: raw}
	versionBytes, err := r.take(4)
	if err != nil {
		return fmt.Errorf("read version: %w", err)
	}
	version := int32(binary.LittleEndian.Uint32(versionBytes))
	segwit := false
	if len(raw) >= 6 && raw[r.pos] == 0 && raw[r.pos+1] != 0 {
		_, _ = r.take(2)
		segwit = true
	}
	inputStart := r.pos
	inputCount, err := r.varInt()
	if err != nil {
		return fmt.Errorf("read input count: %w", err)
	}
	if inputCount == 0 || inputCount > 100000 {
		return fmt.Errorf("invalid input count %d", inputCount)
	}
	inputs := make([]txInput, 0, int(inputCount))
	for i := uint64(0); i < inputCount; i++ {
		prev, err := r.take(32)
		if err != nil {
			return fmt.Errorf("input %d previous txid: %w", i, err)
		}
		vout, err := r.u32()
		if err != nil {
			return fmt.Errorf("input %d vout: %w", i, err)
		}
		script, err := r.varBytes()
		if err != nil {
			return fmt.Errorf("input %d script: %w", i, err)
		}
		seq, err := r.u32()
		if err != nil {
			return fmt.Errorf("input %d sequence: %w", i, err)
		}
		inputs = append(inputs, txInput{PreviousTxID: reverseHex(prev), Vout: vout, Sequence: seq, ScriptSig: hex.EncodeToString(script)})
	}
	outputCount, err := r.varInt()
	if err != nil {
		return fmt.Errorf("read output count: %w", err)
	}
	if outputCount > 100000 {
		return fmt.Errorf("invalid output count %d", outputCount)
	}
	outputs := make([]txOutput, 0, int(outputCount))
	for i := uint64(0); i < outputCount; i++ {
		value, err := r.u64()
		if err != nil {
			return fmt.Errorf("output %d value: %w", i, err)
		}
		if value > 21000000*100000000 {
			return fmt.Errorf("output %d exceeds Bitcoin's maximum monetary supply", i)
		}
		script, err := r.varBytes()
		if err != nil {
			return fmt.Errorf("output %d script: %w", i, err)
		}
		outputs = append(outputs, txOutput{ValueSats: int64(value), ScriptHex: hex.EncodeToString(script)})
	}
	outputsEnd := r.pos
	if segwit {
		for i := uint64(0); i < inputCount; i++ {
			items, err := r.varInt()
			if err != nil {
				return fmt.Errorf("input %d witness count: %w", i, err)
			}
			if items > 100000 {
				return fmt.Errorf("input %d has too many witness items", i)
			}
			for j := uint64(0); j < items; j++ {
				if _, err := r.varBytes(); err != nil {
					return fmt.Errorf("input %d witness item %d: %w", i, j, err)
				}
			}
		}
	}
	lockTime, err := r.u32()
	if err != nil {
		return fmt.Errorf("read lock time: %w", err)
	}
	if r.pos != len(raw) {
		return fmt.Errorf("transaction has %d trailing bytes", len(raw)-r.pos)
	}
	// txid hashes the serialization without the SegWit marker, flag, and witness data.
	stripped := bytes.NewBuffer(nil)
	stripped.Write(versionBytes)
	stripped.Write(raw[inputStart:outputsEnd])
	var lockBytes [4]byte
	binary.LittleEndian.PutUint32(lockBytes[:], lockTime)
	stripped.Write(lockBytes[:])
	hash := doubleSHA256(stripped.Bytes())
	for i, j := 0, len(hash)-1; i < j; i, j = i+1, j-1 {
		hash[i], hash[j] = hash[j], hash[i]
	}
	result := decodedTx{TxID: hex.EncodeToString(hash), Version: version, LockTime: lockTime, SegWit: segwit, InputCount: len(inputs), OutputCount: len(outputs), Inputs: inputs, Outputs: outputs, Note: "Decoded locally. Scripts are shown as hex; addresses and transaction fees are not inferred."}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
