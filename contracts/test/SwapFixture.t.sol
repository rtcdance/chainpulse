// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../SwapEmitter.sol";

/// @notice Deterministic fixture generator for the T09 Swap decoding corpus.
///
/// F16 asks for a Swap fixture with a traceable source. Driving Anvil would
/// give a tx hash, but the corpus must be reproducible anywhere, so this
/// generates the raw logs from Foundry's own EVM: same contract, same inputs,
/// same bytes, no network and no external service.
///
/// Run with:
///   forge test --match-path test/SwapFixture.t.sol -vv
///
/// Each emitted log is printed as a single machine-readable line prefixed with
/// FIXTURE so `make swap-fixture` (or a human) can capture it into
/// test/fixtures/uniswap_v3_swap.json.
contract SwapFixtureTest is Test {
    SwapEmitter emitter;

    function setUp() public {
        emitter = new SwapEmitter();
    }

    /// The typical case: negative amount0, positive amount1, negative tick.
    function test_generateTypicalSwapFixture() public {
        vm.recordLogs();

        address sender = address(0x1111111111111111111111111111111111111111);
        address recipient = address(0x2222222222222222222222222222222222222222);
        emitter.emitTypicalSwap(sender, recipient);

        Vm.Log[] memory logs = vm.getRecordedLogs();
        require(logs.length == 1, "expected exactly one log");

        _printFixture("typical", logs[0]);
    }

    /// The boundary case: int256 min/max, int24 min, uint128 max, uint160 max.
    /// These are the values that break naive decoders (sign extension, width
    /// confusion between int24/int256, unsigned overflow).
    function test_generateBoundarySwapFixture() public {
        vm.recordLogs();

        emitter.emitSwap(
            address(0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa),
            address(0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB),
            type(int256).min,
            type(int256).max,
            type(uint160).max,
            type(uint128).max,
            type(int24).min
        );

        Vm.Log[] memory logs = vm.getRecordedLogs();
        require(logs.length == 1, "expected exactly one log");

        _printFixture("boundary", logs[0]);
    }

    /// int24 max, so both int24 extremes are covered.
    function test_generateMaxTickSwapFixture() public {
        vm.recordLogs();

        emitter.emitSwap(
            address(0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC),
            address(0xDDdDddDdDdddDDddDDddDDDDdDdDDdDDdDDDDDDd),
            1,
            -1,
            1,
            0,
            type(int24).max
        );

        Vm.Log[] memory logs = vm.getRecordedLogs();
        require(logs.length == 1, "expected exactly one log");

        _printFixture("max-tick", logs[0]);
    }

    /// topic0 must be the real Uniswap V3 Swap signature; if this ever changes,
    /// the fixture stops representing the event the plan names.
    function test_topic0MatchesUniswapV3Swap() public pure {
        bytes32 topic0 = keccak256("Swap(address,address,int256,int256,uint160,uint128,int24)");
        require(
            topic0 == bytes32(0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67),
            "topic0 is not the Uniswap V3 Swap signature"
        );
    }

    function _printFixture(string memory name, Vm.Log memory log) internal {
        // Indexed params land in topics[1..]; the non-indexed ones are ABI
        // encoded into data. Emitting both verbatim is what makes the fixture
        // a raw corpus rather than a re-encoding of decoded values.
        emit log_named_string(
            "FIXTURE",
            string.concat(
                '{"name":"',
                name,
                '","topic0":"',
                _hex(log.topics[0]),
                '","topic1":"',
                _hex(log.topics[1]),
                '","topic2":"',
                _hex(log.topics[2]),
                '","data":"',
                _hexBytes(log.data),
                '"}'
            )
        );
    }

    function _hex(bytes32 value) internal pure returns (string memory) {
        return _hexBytes(abi.encodePacked(value));
    }

    function _hexBytes(bytes memory data) internal pure returns (string memory) {
        bytes memory alphabet = "0123456789abcdef";
        bytes memory out = new bytes(2 + data.length * 2);
        out[0] = "0";
        out[1] = "x";
        for (uint256 i = 0; i < data.length; i++) {
            out[2 + i * 2] = alphabet[uint8(data[i] >> 4)];
            out[3 + i * 2] = alphabet[uint8(data[i] & 0x0f)];
        }
        return string(out);
    }
}
