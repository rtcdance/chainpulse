// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title EventEmitter
/// @notice Minimal contract for learning ChainPulse's event indexing flow.
/// Deploy to Anvil, call emitTransfer() / emitCustom(), and watch the events
/// appear in ChainPulse's debugger.
contract EventEmitter {
    event Transfer(address indexed from, address indexed to, uint256 value);
    event CustomEvent(bytes32 indexed id, string message, uint256 timestamp);

    uint256 public counter;

    /// Emit a standard Transfer event (like ERC-20).
    function emitTransfer(address to, uint256 value) external {
        emit Transfer(msg.sender, to, value);
    }

    /// Emit a custom event with a string message.
    function emitCustom(string calldata message) external {
        counter++;
        emit CustomEvent(keccak256(abi.encodePacked(counter)), message, block.timestamp);
    }

    /// Emit multiple events in one transaction for batch testing.
    function emitBatch(uint256 count) external {
        for (uint256 i = 0; i < count; i++) {
            emit Transfer(msg.sender, address(uint160(i + 1)), i + 1);
        }
    }
}
