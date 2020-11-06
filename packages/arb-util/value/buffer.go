/*
 * Copyright 2019, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package value

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/offchainlabs/arbitrum/packages/arb-util/common"
	"github.com/offchainlabs/arbitrum/packages/arb-util/hashing"
)

type BufferHashValue struct {
	hash common.Hash
}

func (iv BufferHashValue) TypeCode() uint8 {
	return TypeCodeBufferHash
}

func NewBufferHash(hashImage common.Hash) BufferHashValue {
	return BufferHashValue{hashImage}
}

func NewBufferHashValueFromReader(rd io.Reader) (BufferHashValue, error) {
	var h common.Hash
	_, err := io.ReadFull(rd, h[:])
	if err != nil {
		return BufferHashValue{}, err
	}
	return NewBufferHash(h), err
}

func (iv BufferHashValue) Clone() Value {
	return BufferHashValue{iv.hash}
}

func (iv BufferHashValue) Equal(val Value) bool {
	other, ok := val.(BufferHashValue)
	if !ok {
		return false
	}
	return iv.hash.Equals(other.hash)
}

func (iv BufferHashValue) Size() int64 {
	return 1
}

func (iv BufferHashValue) Hash() common.Hash {
	return iv.hash
}

func (hp BufferHashValue) Marshal(wr io.Writer) error {
	_, err := wr.Write(hp.hash[:])
	if err != nil {
		return err
	}
	sizeVal := NewInt64Value(hp.Size())
	return sizeVal.Marshal(wr)
}

func (hp BufferHashValue) String() string {
	return fmt.Sprintf("BufferHash(%v)", hp.hash)
}

type BufferValue struct {
	buffer []byte
}

func (iv BufferValue) TypeCode() uint8 {
	return TypeCodeBuffer
}

func NewBuffer(buffer []byte) BufferValue {
	return BufferValue{buffer}
}

func NewBufferValueFromReader(rd io.Reader) (BufferValue, error) {
	intVal, err := NewIntValueFromReader(rd)
	if err != nil {
		return BufferValue{}, err
	}
	h := make([]byte, intVal.BigInt().Int64())
	_, err = io.ReadFull(rd, h[:])
	if err != nil {
		return BufferValue{}, err
	}
	return NewBuffer(h), err
}

func (iv BufferValue) Clone() Value {
	return BufferValue{iv.buffer}
}

func (iv BufferValue) Equal(val Value) bool {
	other, ok := val.(BufferValue)
	if !ok {
		return false
	}
	return bytes.Equal(iv.buffer, other.buffer)
}

func (iv BufferValue) Size() int64 {
	return int64(len(iv.buffer))
}

// TODO: Implement merkle hash once more

var zeroBufferHash = hashing.SoliditySHA3(hashing.Uint256(big.NewInt(0)))

func hashBuffer(buf []byte, pack bool) common.Hash {
	if len(buf) == 0 {
		return zeroBufferHash
	}
	if len(buf) == 32 {
		var arr [32]byte
		copy(arr[:], buf)
		return hashing.SoliditySHA3(hashing.Bytes32(arr))
	}
	len := len(buf)
	h2 := hashBuffer(buf[len/2:len], false)
	if h2 == zeroBufferHash && pack {
		return hashBuffer(buf[0:len/2], true)
	}
	h1 := hashBuffer(buf[0:len/2], false)
	return hashing.SoliditySHA3(hashing.Bytes32(h1), hashing.Bytes32(h2))
}

func (iv BufferValue) Hash() common.Hash {
	return hashBuffer(iv.buffer, true)
}

func (hp BufferValue) Marshal(wr io.Writer) error {
	sizeVal := NewInt64Value(hp.Size())
	err := sizeVal.Marshal(wr)
	if err != nil {
		return err
	}
	_, err = wr.Write(hp.buffer[:])
	return err
}

func (hp BufferValue) String() string {
	return fmt.Sprintf("Buffer(%v)", len(hp.buffer))
}
