// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/structs/EnumerableSet.sol";

contract TestGovernance is Ownable {
    using EnumerableSet for EnumerableSet.AddressSet;

    struct Proposal {
        uint256 id;
        string description;
        address proposer;
        uint256 voteCount;
        uint256 endTime;
        bool executed;
        ProposalState state;
        mapping(address => bool) hasVoted;
    }

    enum ProposalState {
        Pending,
        Active,
        Defeated,
        Succeeded,
        Queued,
        Executed
    }

    IERC20 public token;
    uint256 public votingPeriod; // in seconds
    uint256 public proposalThreshold;
    uint256 public quorumPercentage; // percentage of total supply needed for quorum
    uint256 public proposalCount;

    mapping(uint256 => Proposal) public proposals;
    EnumerableSet.AddressSet private proposers;
    mapping(address => uint256) public tokenBalances; // For testing purposes

    event ProposalCreated(
        uint256 indexed id,
        address indexed proposer,
        string description,
        uint256 endTime
    );
    event Voted(uint256 indexed proposalId, address indexed voter, bool support);
    event ProposalExecuted(uint256 indexed proposalId);

    constructor(
        address _token,
        uint256 _votingPeriod,
        uint256 _proposalThreshold,
        uint256 _quorumPercentage
    ) {
        require(_token != address(0), "Invalid token address");
        require(_quorumPercentage <= 100, "Quorum percentage cannot exceed 100");
        
        token = IERC20(_token);
        votingPeriod = _votingPeriod;
        proposalThreshold = _proposalThreshold;
        quorumPercentage = _quorumPercentage;
    }

    function createProposal(string memory description) external returns (uint256) {
        require(
            token.balanceOf(msg.sender) >= proposalThreshold,
            "Insufficient balance to create proposal"
        );

        proposalCount++;
        Proposal storage newProposal = proposals[proposalCount];
        
        newProposal.id = proposalCount;
        newProposal.description = description;
        newProposal.proposer = msg.sender;
        newProposal.endTime = block.timestamp + votingPeriod;
        newProposal.state = ProposalState.Pending;
        
        proposers.add(msg.sender);
        
        // For testing: simulate token lock
        tokenBalances[msg.sender] = token.balanceOf(msg.sender);

        emit ProposalCreated(proposalCount, msg.sender, description, newProposal.endTime);
        
        return proposalCount;
    }

    function vote(uint256 proposalId, bool support) external {
        Proposal storage proposal = proposals[proposalId];
        require(
            proposal.state == ProposalState.Active || 
            (proposal.state == ProposalState.Pending && block.timestamp >= proposal.endTime - votingPeriod/2), // Allow early voting
            "Voting is not active for this proposal"
        );
        require(!proposal.hasVoted[msg.sender], "Already voted");
        
        proposal.hasVoted[msg.sender] = true;
        
        if (support) {
            proposal.voteCount++;
        } else {
            if (proposal.voteCount > 0) {
                proposal.voteCount--;
            }
        }

        emit Voted(proposalId, msg.sender, support);
    }

    function executeProposal(uint256 proposalId) external {
        Proposal storage proposal = proposals[proposalId];
        require(!proposal.executed, "Proposal already executed");
        require(block.timestamp >= proposal.endTime, "Voting period not ended");
        require(proposal.state != ProposalState.Defeated, "Proposal defeated");
        
        // Calculate if quorum was met
        uint256 totalSupply = token.totalSupply();
        uint256 minVotesForQuorum = (totalSupply * quorumPercentage) / 100;
        
        if (proposal.voteCount < minVotesForQuorum) {
            proposal.state = ProposalState.Defeated;
            return;
        }
        
        // Determine if proposal passed (simple majority)
        uint256 againstVotes = getProposalVotes(proposalId, false);
        if (proposal.voteCount > againstVotes) {
            proposal.state = ProposalState.Succeeded;
            proposal.executed = true;
            emit ProposalExecuted(proposalId);
        } else {
            proposal.state = ProposalState.Defeated;
        }
    }

    function getProposalVotes(uint256 proposalId, bool support) public view returns (uint256 voteCount) {
        // This is a simplified implementation for testing
        // In a real governance contract, this would be more complex
        Proposal storage proposal = proposals[proposalId];
        if (support) {
            return proposal.voteCount;
        } else {
            // Count of non-support votes would need to be tracked separately
            // This is a simplified approach for testing
            return 0; // Placeholder
        }
    }

    function queueProposal(uint256 proposalId) external {
        Proposal storage proposal = proposals[proposalId];
        require(
            block.timestamp >= proposal.endTime && 
            proposal.state == ProposalState.Pending,
            "Proposal not ready to be queued"
        );
        
        proposal.state = ProposalState.Queued;
    }

    function setVotingPeriod(uint256 newVotingPeriod) external onlyOwner {
        require(newVotingPeriod > 0, "Voting period must be greater than 0");
        votingPeriod = newVotingPeriod;
    }

    function setProposalThreshold(uint256 newThreshold) external onlyOwner {
        proposalThreshold = newThreshold;
    }

    function setQuorumPercentage(uint256 newQuorumPercentage) external onlyOwner {
        require(newQuorumPercentage <= 100, "Quorum percentage cannot exceed 100");
        quorumPercentage = newQuorumPercentage;
    }

    function getProposalState(uint256 proposalId) external view returns (ProposalState) {
        return proposals[proposalId].state;
    }

    function getTokenBalance(address account) external view returns (uint256) {
        return token.balanceOf(account);
    }

    // Helper function for testing to set balances
    function setTokenBalance(address account, uint256 balance) external onlyOwner {
        tokenBalances[account] = balance;
    }
}