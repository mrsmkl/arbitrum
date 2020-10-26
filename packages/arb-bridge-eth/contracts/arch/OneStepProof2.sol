// SPDX-License-Identifier: Apache-2.0

/*
 * Copyright 2019-2020, Offchain Labs, Inc.
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

pragma solidity ^0.5.11;

import "./IOneStepProof.sol";
import "./OneStepProofCommon.sol";
import "./Value.sol";
import "./Machine.sol";
import "../inbox/Messages.sol";
import "../libraries/Precompiles.sol";

// Originally forked from https://github.com/leapdao/solEVM-enforcer/tree/master

contract OneStepProof2 is IOneStepProof2, OneStepProofCommon {

    function executeStep(
        bytes32 inboxAcc,
        bytes32 messagesAcc,
        bytes32 logsAcc,
        bytes calldata proof,
        bytes calldata bproof
    ) external view returns (uint64 gas, bytes32[5] memory fields) {
        AssertionContext memory context = initializeExecutionContext(
            inboxAcc,
            messagesAcc,
            logsAcc,
            proof,
            bproof
        );

        executeOp(context);

        return returnContext(context);
    }

    /* solhint-disable no-inline-assembly */

    function executeErrorInsn(AssertionContext memory context) internal pure {
        handleOpcodeError(context);
    }

    function executeNewBuffer(AssertionContext memory context) internal pure {
        Value.Data memory val1 = popVal(context.stack);
        if (!val1.isInt()) {
            handleOpcodeError(context);
            return;
        }
        pushVal(context.stack, Value.newBuffer(keccak256(abi.encodePacked(bytes32(0)))));
    }

    
    function makeZeros() internal pure returns (bytes32[] memory) {
        bytes32[] memory zeros = new bytes32[](64);
        zeros[0] = keccak1(0);
        for (uint i = 1; i < 64; i++) {
            zeros[i] = keccak2(zeros[i-1], zeros[i-1]);
        }
        return zeros;
    }

    function keccak1(bytes32 b) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(b));
    }

    function keccak2(bytes32 a, bytes32 b) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(a, b));
    }

    // hashes are normalized
    function get(bytes32 buf, uint loc, bytes32[] memory proof) internal pure returns (bytes32) {
        // empty tree is full of zeros
        if (proof.length == 0) {
            require(buf == keccak1(bytes32(0)), "expected empty buffer");
            return 0;
        }
        bytes32 acc = keccak1(proof[0]);
        for (uint i = 1; i < proof.length; i++) {
            if (loc & 1 == 1) acc = keccak2(proof[i], acc);
            else acc = keccak2(acc, proof[i]);
            loc = loc >> 1;
        }
        require(acc == buf, "expected correct root");
        // maybe it is a zero outside the actual tree
        if (loc > 0) return 0;
        return proof[0];
    }

    function calcHeight(uint loc) internal pure returns (uint) {
        if (loc == 0) return 1;
        else return 1+calcHeight(loc>>1);
    }

    function set(bytes32 buf, uint loc, bytes32 v, bytes32[] memory proof, uint nh, bytes32 normal1, bytes32 normal2) internal pure returns (bytes32) {
        // three possibilities, the tree depth stays same, it becomes lower or it's extended
        bytes32 acc = keccak1(v);
        // check that the proof matches original
        get(buf, loc, proof);
        bytes32[] memory zeros = makeZeros();
        // extended
        if (loc >= (1 << (proof.length-1))) {
            if (v == 0) return buf;
            uint height = calcHeight(loc);
            // build the left branch
            for (uint i = proof.length; i < height-1; i++) {
                buf = keccak2(buf, zeros[i-1]);
            }
            for (uint i = 1; i < height-1; i++) {
                if (loc & 1 == 1) acc = keccak2(zeros[i-1], acc);
                else acc = keccak2(acc, zeros[i-1]);
                loc = loc >> 1;
            }
            return keccak2(buf, acc);
        }
        for (uint i = 1; i < proof.length; i++) {
            bytes32 a = loc & 1 == 1 ? proof[i] : acc;
            bytes32 b = loc & 1 == 1 ? acc : proof[i];
            acc = keccak2(a, b);
            loc = loc >> 1;
        }
        if (v != bytes32(0)) return acc;
        require(normal2 != zeros[nh] || nh == 0, "right subtree cannot be zero");
        bytes32 res = nh == 0 ? normal1 : keccak2(normal1, normal2);
        bytes32 acc2 = res;
        for (uint i = nh; i < proof.length-1; i++) {
            acc2 = keccak2(acc2, zeros[i]);
        }
        require(acc2 == acc, "expected match");
        return res;
    }

    function getByte(bytes32 word, uint256 num) internal pure returns (uint256) {
        return (uint256(word) >> ((31-num)*8)) & 0xff;
    }

    function setByte(bytes32 word, uint256 num, uint256 b) internal pure returns (bytes32) {
        bytes memory arr = bytes32ToArray(word);
        arr[num] = bytes1(uint8(b));
        return bytes32(bytes32FromArray(arr));
    }

    function setByte(bytes32 word, uint256 num, bytes1 b) internal pure returns (bytes32) {
        bytes memory arr = bytes32ToArray(word);
        arr[num] = b;
        return bytes32(bytes32FromArray(arr));
    }

    function decode(bytes memory arr, bytes1 _start, bytes1 _end) internal pure returns (bytes32[] memory) {
        uint len = uint(uint8(_end)-uint8(_start));
        uint start = uint(uint8(_start));
        bytes32[] memory res = new bytes32[](len);
        for (uint i = 0; i < len; i++) {
            res[i] = bytes32(bytes32FromArray(arr, (start+i)*32));
        }
        return res;
    }

    struct BufferProof {
        bytes32[] proof1;
        bytes32[] nproof1;
        bytes32[] proof2;
        bytes32[] nproof2;
    }

    function decodeProof(bytes memory proof) internal pure returns (BufferProof memory) {
        bytes32[] memory proof1 = decode(proof, proof[0], proof[1]);
        bytes32[] memory nproof1 = decode(proof, proof[1], proof[2]);
        bytes32[] memory proof2 = decode(proof, proof[2], proof[3]);
        bytes32[] memory nproof2 = decode(proof, proof[3], proof[4]);
        return BufferProof(proof1, nproof1, proof2, nproof2);
    }

    function bytes32FromArray(bytes memory arr) internal pure returns (uint256) {
        uint256 res = 0;
        for (uint i = 0; i < arr.length; i++) {
            res = res << 8;
            res = res | uint256(uint8(arr[i]));
        }
        return res;
    }

    function bytes32FromArray(bytes memory arr, uint offset) internal pure returns (uint256) {
        uint256 res = 0;
        for (uint i = 0; i < 32; i++) {
            res = res << 8;
            res = res | uint256(uint8(arr[offset+i]));
        }
        return res;
    }

    function bytes32ToArray(bytes32 b) internal pure returns (bytes memory) {
        uint256 acc = uint256(b);
        bytes memory res = new bytes(32);
        for (uint i = 0; i < 32; i++) {
            res[31-i] = bytes1(uint8(acc));
            acc = acc >> 8;
        }
        return res;
    }

    function getBuffer8(bytes32 buf, uint256 offset, BufferProof memory proof) internal pure returns (uint256) {
        return getByte(get(buf, offset/32, proof.proof1), offset%32);
    }

    function getBuffer64(bytes32 buf, uint256 offset, BufferProof memory proof) internal pure returns (uint256) {
        bytes memory res = new bytes(8);
        bytes32 word = get(buf, offset/32, proof.proof1); 
        if (offset%32 + 8 >= 32) {
            bytes32 word2 = get(buf, offset/32 + 1, proof.proof2);
            for (uint i = 0; i < 8 - (offset%32 + 8 - 32); i++) {
                res[i] = bytes1(uint8(getByte(word, offset%32 + i)));
            }
            for (uint i = 8 - (offset%32 + 8 - 32); i < 8; i++) {
                res[i] = bytes1(uint8(getByte(word2, (offset + i) % 32)));
            }
        } else {
            for (uint i = 0; i < 8; i++) {
                res[i] = bytes1(uint8(getByte(word, offset%32 + i)));
            }
        }
        return bytes32FromArray(res);
    }

    function getBuffer256(bytes32 buf, uint256 offset, BufferProof memory proof) internal pure returns (uint256) {
        bytes memory res = new bytes(32);
        bytes32 word = get(buf, offset/32, proof.proof1); 
        if (offset%32 + 32 >= 32) {
            bytes32 word2 = get(buf, offset/32 + 1, proof.proof2);
            for (uint i = 0; i < 32 - (offset%32 + 32 - 32); i++) {
                res[i] = bytes1(uint8(getByte(word, offset%32 + i)));
            }
            for (uint i = 8 - (offset%32 + 32 - 32); i < 32; i++) {
                res[i] = bytes1(uint8(getByte(word2, (offset + i) % 32)));
            }
        } else {
            for (uint i = 0; i < 32; i++) {
                res[i] = bytes1(uint8(getByte(word, offset%32 + i)));
            }
        }
        return bytes32FromArray(res);
    }

    function set(bytes32 buf, uint loc, bytes32 v, bytes32[] memory proof, bytes32[] memory nproof) internal pure returns (bytes32) {
        require(nproof.length == 3, "normalization proof has wrong size");
        return set(buf, loc, v, proof, uint256(nproof[0]), nproof[1], nproof[2]);
    }

    function setBuffer8(bytes32 buf, uint256 offset, uint256 b, BufferProof memory proof) internal pure returns (bytes32) {
        bytes32 word = get(buf, offset/32, proof.proof1);
        bytes32 nword = setByte(word, offset%32, b);
        bytes32 res = set(buf, offset/32, nword, proof.proof1, proof.nproof1);
        return res;
    }

    function setBuffer64(bytes32 buf, uint256 offset, uint256 val, BufferProof memory proof) internal pure returns (bytes32) {
        bytes memory arr = bytes32ToArray(bytes32(val));
        bytes32 nword = get(buf, offset/32, proof.proof1);
        if (offset%32 + 8 >= 32) {
            for (uint i = 0; i < 8 - (offset%32 + 8 - 32); i++) {
                nword = setByte(nword, (offset+i)%32, arr[i+24]);
            }
            buf = set(buf, offset/32, nword, proof.proof1, proof.nproof1);
            bytes32 nword2 = get(buf, offset/32 + 1, proof.proof2); 
            for (uint i = 8 - (offset%32 + 8 - 32); i < 8; i++) {
                nword2 = setByte(nword2, (offset+i)%32, arr[i+24]);
            }
            buf = set(buf, offset/32 + 1, nword2, proof.proof2, proof.nproof2);
        } else {
            for (uint i = 0; i < 8; i++) {
                nword = setByte(nword, offset%32 + i, arr[i+24]);
            }
            buf = set(buf, offset/32, nword, proof.proof1, proof.nproof1);
        }
        return buf;
    }

    function setBuffer256(bytes32 buf, uint256 offset, uint256 val, BufferProof memory proof) internal pure returns (bytes32) {
        bytes memory arr = bytes32ToArray(bytes32(val));
        bytes32 nword = get(buf, offset/32, proof.proof1);
        if (offset%32 + 32 >= 32) {
            for (uint i = 0; i < 32 - (offset%32 + 32 - 32); i++) {
                nword = setByte(nword, offset%32 + i, arr[i]);
            }
            buf = set(buf, offset/32, nword, proof.proof1, proof.nproof1);
            bytes32 nword2 = get(buf, offset/32 + 1, proof.proof2); 
            for (uint i = 32 - (offset%32 + 32 - 32); i < 32; i++) {
                nword2 = setByte(nword2, (offset+i)%32, arr[i]);
            }
            buf = set(buf, offset/32 + 1, nword2, proof.proof2, proof.nproof2);
        } else {
            for (uint i = 0; i < 32; i++) {
                nword = setByte(nword, offset%32 + i, arr[i]);
            }
            buf = set(buf, offset/32, nword, proof.proof1, proof.nproof1);
        }
        return buf;
    }

    function executeGetBuffer8(AssertionContext memory context) internal pure {
        Value.Data memory val1 = popVal(context.stack);
        Value.Data memory val2 = popVal(context.stack);
        if (!val2.isInt64() || !val1.isBuffer()) {
            handleOpcodeError(context);
            return;
        }
        uint256 res = getBuffer8(val1.bufferHash, val2.intVal, decodeProof(context.bufProof));
        pushVal(context.stack, Value.newInt(res));
    }

    function executeGetBuffer64(AssertionContext memory context) internal pure {
        Value.Data memory val1 = popVal(context.stack);
        Value.Data memory val2 = popVal(context.stack);
        if (!val2.isInt64() || !val1.isBuffer()) {
            handleOpcodeError(context);
            return;
        }
        require(val1.intVal < (1 << 64), "buffer index must be 64-bit");
        uint256 res = getBuffer64(val1.bufferHash, val2.intVal, decodeProof(context.bufProof));
        pushVal(context.stack, Value.newInt(res));
    }

    function executeGetBuffer256(AssertionContext memory context) internal pure {
        Value.Data memory val1 = popVal(context.stack);
        Value.Data memory val2 = popVal(context.stack);
        if (!val2.isInt64() || !val1.isBuffer()) {
            handleOpcodeError(context);
            return;
        }
        require(val1.intVal < (1 << 64), "buffer index must be 64-bit");
        uint256 res = getBuffer256(val1.bufferHash, val2.intVal, decodeProof(context.bufProof));
        pushVal(context.stack, Value.newInt(res));
    }

    function executeSetBuffer8(AssertionContext memory context) internal pure {
        Value.Data memory val1 = popVal(context.stack);
        Value.Data memory val2 = popVal(context.stack);
        Value.Data memory val3 = popVal(context.stack);
        if (!val2.isInt64() || !val3.isInt() || !val1.isBuffer()) {
            handleOpcodeError(context);
            return;
        }
        bytes32 res = setBuffer8(val1.bufferHash, val2.intVal, val3.intVal, decodeProof(context.bufProof));
        pushVal(context.stack, Value.newBuffer(res));
    }

    function executeSetBuffer64(AssertionContext memory context) internal pure {
        Value.Data memory val1 = popVal(context.stack);
        Value.Data memory val2 = popVal(context.stack);
        Value.Data memory val3 = popVal(context.stack);
        if (!val2.isInt64() || !val3.isInt() || !val1.isBuffer()) {
            handleOpcodeError(context);
            return;
        }
        bytes32 res = setBuffer64(val1.bufferHash, val2.intVal, val3.intVal, decodeProof(context.bufProof));
        pushVal(context.stack, Value.newBuffer(res));
    }

    function executeSetBuffer256(AssertionContext memory context) internal pure {
        Value.Data memory val1 = popVal(context.stack);
        Value.Data memory val2 = popVal(context.stack);
        Value.Data memory val3 = popVal(context.stack);
        if (!val2.isInt64() || !val3.isInt() || !val1.isBuffer()) {
            handleOpcodeError(context);
            return;
        }
        bytes32 res = setBuffer256(val1.bufferHash, val2.intVal, val3.intVal, decodeProof(context.bufProof));
        pushVal(context.stack, Value.newBuffer(res));
    }

    function opInfo(uint256 opCode)
        internal
        pure
        returns (
            uint256, // stack pops
            uint256, // auxstack pops
            uint64, // gas used
            function(AssertionContext memory) internal view // impl
        )
    {
       
        if (opCode == OP_NEWBUFFER) {
            return (1, 0, 1, executeNewBuffer);
        } else if (opCode == OP_GETBUFFER8) {
            return (2, 0, 10, executeGetBuffer8);
        } else if (opCode == OP_GETBUFFER64) {
            return (2, 0, 10, executeGetBuffer64);
        } else if (opCode == OP_GETBUFFER256) {
            return (2, 0, 10, executeGetBuffer256);
        } else if (opCode == OP_SETBUFFER8) {
            return (3, 0, 10, executeSetBuffer8);
        } else if (opCode == OP_SETBUFFER64) {
            return (3, 0, 10, executeSetBuffer64);
        } else if (opCode == OP_SETBUFFER256) {
            return (3, 0, 10, executeSetBuffer256);
        } else {
            return (0, 0, 0, executeErrorInsn);
        }
    }
}