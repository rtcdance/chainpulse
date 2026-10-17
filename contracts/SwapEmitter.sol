// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title SwapEmitter
/// @notice Emits the exact Uniswap V3 `Swap` event so a raw log can be captured
///         as a decoding fixture (task T09 / finding F16).
///
/// The event signature is deliberately byte-identical to Uniswap V3's
/// IUniswapV3PoolEvents.Swap, so its topic0 is
/// 0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67.
/// A contract that only *emits* is enough: decoding depends on the log's
/// topics and data, not on the pool's behaviour, and this keeps the fixture
/// generator free of any Uniswap dependency.
contract SwapEmitter {
    event Swap(
        address indexed sender,
        address indexed recipient,
        int256 amount0,
        int256 amount1,
        uint160 sqrtPriceX96,
        uint128 liquidity,
        int24 tick
    );

    /// Emit one Swap with caller-chosen values, so a fixture can pin exact
    /// boundaries (int256 min/max, negative int24, uint128/uint160 maxima).
    function emitSwap(
        address sender,
        address recipient,
        int256 amount0,
        int256 amount1,
        uint160 sqrtPriceX96,
        uint128 liquidity,
        int24 tick
    ) external {
        emit Swap(sender, recipient, amount0, amount1, sqrtPriceX96, liquidity, tick);
    }

    /// A representative "real world" swap: token0 out, token1 in, tick below
    /// zero (price below 1). Exercises the common sign combination.
    function emitTypicalSwap(address sender, address recipient) external {
        emit Swap(
            sender,
            recipient,
            -1234567890123456789, // token0 out
            9876543210987654321, // token1 in
            18446744073709551615, // sqrtPriceX96
            340282366920938463463374607431768211455, // liquidity = uint128 max
            -887272 // tick = int24 min
        );
    }
}
