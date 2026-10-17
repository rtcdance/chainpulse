// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../EventEmitter.sol";

/// @notice Unit tests for EventEmitter.
/// @dev Solidity cannot `emit` another contract's event through a qualified
/// name, so the events under test are re-declared locally for `vm.expectEmit`.
/// Signatures must stay identical to EventEmitter's declarations.
contract EventEmitterTest is Test {
    event Transfer(address indexed from, address indexed to, uint256 value);
    event CustomEvent(bytes32 indexed id, string message, uint256 timestamp);

    EventEmitter emitter;

    function setUp() public {
        emitter = new EventEmitter();
    }

    function test_emitTransfer() public {
        address to = address(0x1234);
        uint256 value = 42;

        vm.expectEmit(true, true, false, true);
        emit Transfer(address(this), to, value);

        emitter.emitTransfer(to, value);
    }

    function test_emitCustom() public {
        string memory message = "hello chainpulse";

        vm.expectEmit(true, false, false, true);
        emit CustomEvent(
            keccak256(abi.encodePacked(uint256(1))),
            message,
            block.timestamp
        );

        emitter.emitCustom(message);
        assertEq(emitter.counter(), 1);
    }

    function test_emitBatch() public {
        uint256 count = 5;

        for (uint256 i = 0; i < count; i++) {
            vm.expectEmit(true, true, false, true);
            emit Transfer(address(this), address(uint160(i + 1)), i + 1);
        }

        emitter.emitBatch(count);
    }

    function test_counterIncrements() public {
        assertEq(emitter.counter(), 0);

        emitter.emitCustom("first");
        assertEq(emitter.counter(), 1);

        emitter.emitCustom("second");
        assertEq(emitter.counter(), 2);
    }

    function test_emitBatchRevertOnZero() public {
        // Should not revert, just emit no events.
        emitter.emitBatch(0);
    }
}
