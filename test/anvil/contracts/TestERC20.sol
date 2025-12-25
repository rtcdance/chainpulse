// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract TestERC20 is ERC20 {
    uint8 private immutable _decimals;

    constructor(
        string memory name,
        string memory symbol,
        uint8 decimals_,
        uint256 initialSupply
    ) ERC20(name, symbol) {
        _decimals = decimals_;
        if (initialSupply > 0) {
            _mint(msg.sender, initialSupply);
        }
    }

    function decimals() public view virtual override returns (uint8) {
        return _decimals;
    }

    function mint(address to, uint256 amount) public {
        _mint(to, amount);
    }

    function burn(address from, uint256 amount) public {
        _burn(from, amount);
    }

    // Function to transfer tokens to multiple addresses at once
    function batchTransfer(address[] calldata recipients, uint256[] calldata amounts) external {
        require(recipients.length == amounts.length, "Recipients and amounts length mismatch");
        
        for (uint256 i = 0; i < recipients.length; i++) {
            transfer(recipients[i], amounts[i]);
        }
    }
}