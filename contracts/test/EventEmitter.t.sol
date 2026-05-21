// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../EventEmitter.sol";

contract EventEmitterTest is Test {
    EventEmitter emitter;

    function setUp() public {
        emitter = new EventEmitter();
    }

    function test_emitTransfer() public {
        address to = address(0x1234);
        uint256 value = 42;

        vm.expectEmit(true, true, false, true);
        emit EventEmitter.Transfer(address(this), to, value);

        emitter.emitTransfer(to, value);
    }

    function test_emitCustom() public {
        string memory message = "hello chainpulse";

        vm.expectEmit(true, false, false, true);
        emit EventEmitter.CustomEvent(keccak256(abi.encodePacked(uint256(1))), message, block.timestamp);

        emitter.emitCustom(message);
        assertEq(emitter.counter(), 1);
    }

    function test_emitBatch() public {
        uint256 count = 5;

        for (uint256 i = 0; i < count; i++) {
            vm.expectEmit(true, true, false, true);
            emit EventEmitter.Transfer(address(this), address(uint160(i + 1)), i + 1);
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
        emitter.emitBatch(0);
        // Should not revert, just emit no events
    }
}