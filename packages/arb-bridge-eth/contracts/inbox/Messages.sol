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

import "../arch/Value.sol";
import "../arch/Marshaling.sol";
import "../libraries/BytesLib.sol";

library Messages {
    using BytesLib for bytes;

    function messageHash(
        uint8 kind,
        address sender,
        uint256 blockNumber,
        uint256 timestamp,
        uint256 inboxSeqNum,
        bytes32 messageDataHash
    ) internal pure returns (bytes32) {
        return
            keccak256(
                abi.encodePacked(kind, sender, blockNumber, timestamp, inboxSeqNum, messageDataHash)
            );
    }

    function keccak1(bytes32 b) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(b));
    }

    function keccak2(bytes32 a, bytes32 b) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(a, b));
    }

    function bytes32FromArray(bytes memory arr, uint256 offset) internal pure returns (uint256) {
        uint256 res = 0;
        for (uint256 i = 0; i < 32; i++) {
            res = res << 8;
            res = res | uint256(uint8(arr[offset + i]));
        }
        return res;
    }

    function merkleRoot(bytes memory data, uint256 startOffset, uint256 dataLength, bool pack) internal pure returns (bytes32) {
        if (dataLength == 32) {
            if (startOffset >= data.length) {
                return keccak1(bytes32(0));
            }
            return keccak1(bytes32(bytes32FromArray(data, startOffset)));
        }
        bytes32 h2 = merkleRoot(data, startOffset + dataLength / 2, dataLength/2, false);
        if (h2 == keccak1(bytes32(0)) && pack) {
            return merkleRoot(data, startOffset, dataLength / 2, true);
        }
        bytes32 h1 = merkleRoot(data, startOffset, dataLength / 2, false);
        return keccak2(h1, h2);
    }

    function messageDataHash(bytes memory data) internal pure returns (bytes32) {
        return merkleRoot(data, 0, data.length, true);
    }

    function messageValue(
        uint8 kind,
        uint256 blockNumber,
        uint256 timestamp,
        address sender,
        uint256 inboxSeqNum,
        bytes memory messageData
    ) internal pure returns (Value.Data memory) {
        Value.Data[] memory tupData = new Value.Data[](6);
        tupData[0] = Value.newInt(uint256(kind));
        tupData[1] = Value.newInt(blockNumber);
        tupData[2] = Value.newInt(timestamp);
        tupData[3] = Value.newInt(uint256(sender));
        tupData[4] = Value.newInt(inboxSeqNum);
        tupData[5] = Marshaling.bytesToBuffer(messageData, 0, messageData.length);
        return Value.newTuple(tupData);
    }

    function addMessageToInbox(bytes32 inbox, bytes32 message) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(inbox, message));
    }

    struct OutgoingMessage {
        uint8 kind;
        address sender;
        bytes data;
    }

    struct EthMessage {
        address dest;
        uint256 value;
    }

    struct ERC20Message {
        address token;
        address dest;
        uint256 value;
    }

    struct ERC721Message {
        address token;
        address dest;
        uint256 id;
    }

    uint256 private constant ETH_MESSAGE_LENGTH = 20 + 32;
    uint256 private constant ERC20_MESSAGE_LENGTH = 20 + 20 + 32;
    uint256 private constant ERC721_MESSAGE_LENGTH = 20 + 20 + 32;

    function unmarshalOutgoingMessage(bytes memory data, uint256 startOffset)
        internal
        pure
        returns (
            bool valid,
            uint256 offset,
            OutgoingMessage memory message
        )
    {
        offset = startOffset;
        uint8 valType = uint8(data[offset]);
        offset++;

        if (valType != Value.tupleTypeCode() + 3) {
            return (false, startOffset, message);
        }

        uint256 rawKind;
        (valid, offset, rawKind) = Marshaling.deserializeCheckedInt(data, offset);
        if (!valid) {
            return (false, startOffset, message);
        }
        message.kind = uint8(rawKind);

        uint256 senderRaw;
        (valid, offset, senderRaw) = Marshaling.deserializeCheckedInt(data, offset);
        if (!valid) {
            return (false, startOffset, message);
        }

        message.sender = address(uint160((senderRaw)));
        (valid, offset, message.data) = Marshaling.bufferToBytes(data, offset);
        if (!valid) {
            return (false, startOffset, message);
        }

        return (true, offset, message);
    }

    function parseEthMessage(bytes memory data)
        internal
        pure
        returns (bool valid, Messages.EthMessage memory message)
    {
        if (data.length < ETH_MESSAGE_LENGTH) {
            return (false, message);
        }
        uint256 offset = 0;
        offset += 12;
        message.dest = data.toAddress(offset);
        offset += 20;
        message.value = data.toUint(offset);
        return (true, message);
    }

    function parseERC20Message(bytes memory data)
        internal
        pure
        returns (bool valid, Messages.ERC20Message memory message)
    {
        if (data.length < ERC20_MESSAGE_LENGTH) {
            return (false, message);
        }
        uint256 offset = 0;
        offset += 12;
        message.token = data.toAddress(offset);
        offset += 20;
        offset += 12;
        message.dest = data.toAddress(offset);
        offset += 20;
        message.value = data.toUint(offset);
        return (true, message);
    }

    function parseERC721Message(bytes memory data)
        internal
        pure
        returns (bool valid, Messages.ERC721Message memory message)
    {
        if (data.length < ERC721_MESSAGE_LENGTH) {
            return (false, message);
        }
        uint256 offset = 0;
        offset += 12;
        message.token = data.toAddress(offset);
        offset += 20;
        offset += 12;
        message.dest = data.toAddress(offset);
        offset += 20;
        message.id = data.toUint(offset);
        return (true, message);
    }
}
