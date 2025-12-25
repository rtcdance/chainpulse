// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/token/ERC721/extensions/ERC721URIStorage.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Counters.sol";

contract TestERC721 is ERC721, ERC721URIStorage, Ownable {
    using Counters for Counters.Counter;

    Counters.Counter private _tokenIdCounter;
    string private _baseTokenURI;

    constructor(
        string memory name,
        string memory symbol,
        string memory baseTokenURI
    ) ERC721(name, symbol) {
        _baseTokenURI = baseTokenURI;
    }

    function _baseURI() internal view virtual override returns (string memory) {
        return _baseTokenURI;
    }

    function safeMint(address to) public onlyOwner returns (uint256) {
        uint256 tokenId = _tokenIdCounter.current();
        _tokenIdCounter.increment();
        
        _safeMint(to, tokenId);
        _setTokenURI(tokenId, string(abi.encodePacked(_baseTokenURI, Strings.toString(tokenId))));
        
        return tokenId;
    }

    function batchMint(address[] calldata toAddresses) external onlyOwner returns (uint256[] memory) {
        uint256[] memory tokenIds = new uint256[](toAddresses.length);
        
        for (uint256 i = 0; i < toAddresses.length; i++) {
            tokenIds[i] = safeMint(toAddresses[i]);
        }
        
        return tokenIds;
    }

    function tokenURI(uint256 tokenId) public view virtual override(ERC721, ERC721URIStorage) returns (string memory) {
        return super.tokenURI(tokenId);
    }

    function supportsInterface(bytes4 interfaceId) public view virtual override(ERC721, ERC721URIStorage) returns (bool) {
        return super.supportsInterface(interfaceId);
    }

    // Override to allow burning own tokens
    function burn(uint256 tokenId) public {
        require(ownerOf(tokenId) == msg.sender, "Not owner of token");
        _burn(tokenId);
    }

    function withdraw() external onlyOwner {
        uint256 balance = address(this).balance;
        payable(owner()).transfer(balance);
    }
}