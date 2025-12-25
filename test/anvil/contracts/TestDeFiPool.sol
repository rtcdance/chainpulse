// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";

contract TestDeFiPool is Ownable, ReentrancyGuard {
    using SafeERC20 for IERC20;

    IERC20 public immutable lpToken;
    IERC20 public immutable rewardToken;

    uint256 public constant PRECISION = 1e18;
    uint256 public rewardRate; // Amount of rewardToken distributed per second
    uint256 public lastUpdateTime;
    uint256 public rewardPerTokenStored;
    uint256 public totalStaked;

    mapping(address => uint256) public userRewardPerTokenPaid;
    mapping(address => uint256) public rewards;
    mapping(address => uint256) public stakedAmounts;

    event Staked(address indexed user, uint256 amount);
    event Withdrawn(address indexed user, uint256 amount);
    event RewardPaid(address indexed user, uint256 reward);
    event RewardRateUpdated(uint256 newRate);

    constructor(
        address _lpToken,
        address _rewardToken,
        uint256 _initialRewardRate
    ) {
        require(_lpToken != address(0), "Invalid LP token address");
        require(_rewardToken != address(0), "Invalid reward token address");
        
        lpToken = IERC20(_lpToken);
        rewardToken = IERC20(_rewardToken);
        rewardRate = _initialRewardRate;
        lastUpdateTime = block.timestamp;
    }

    modifier updateReward(address account) {
        rewardPerTokenStored = rewardPerToken();
        lastUpdateTime = lastTimeRewardApplicable();

        if (account != address(0)) {
            rewards[account] = earned(account);
            userRewardPerTokenPaid[account] = rewardPerTokenStored;
        }
        _;
    }

    function lastTimeRewardApplicable() public view returns (uint256) {
        return block.timestamp < rewardRate * PRECISION ? block.timestamp : rewardRate * PRECISION;
    }

    function rewardPerToken() public view returns (uint256) {
        if (totalStaked == 0) {
            return rewardPerTokenStored;
        }
        return
            rewardPerTokenStored +
            (((lastTimeRewardApplicable() - lastUpdateTime) * rewardRate * PRECISION) / totalStaked);
    }

    function earned(address account) public view returns (uint256) {
        return
            ((stakedAmounts[account] * (rewardPerToken() - userRewardPerTokenPaid[account])) / PRECISION) +
            rewards[account];
    }

    function stake(uint256 amount) external nonReentrant updateReward(msg.sender) {
        require(amount > 0, "Amount must be greater than 0");
        
        totalStaked += amount;
        stakedAmounts[msg.sender] += amount;
        
        lpToken.safeTransferFrom(msg.sender, address(this), amount);
        
        emit Staked(msg.sender, amount);
    }

    function withdraw(uint256 amount) external nonReentrant updateReward(msg.sender) {
        require(amount > 0, "Amount must be greater than 0");
        require(stakedAmounts[msg.sender] >= amount, "Insufficient balance");
        
        totalStaked -= amount;
        stakedAmounts[msg.sender] -= amount;
        
        lpToken.safeTransfer(msg.sender, amount);
        
        emit Withdrawn(msg.sender, amount);
    }

    function getReward() external nonReentrant updateReward(msg.sender) {
        uint256 reward = rewards[msg.sender];
        if (reward > 0) {
            rewards[msg.sender] = 0;
            rewardToken.safeTransfer(msg.sender, reward);
            emit RewardPaid(msg.sender, reward);
        }
    }

    function exit() external {
        withdraw(stakedAmounts[msg.sender]);
        getReward();
    }

    function notifyRewardAmount(uint256 reward) external onlyOwner updateReward(address(0)) {
        if (block.timestamp >= rewardRate * PRECISION) {
            rewardPerTokenStored = 0;
        } else {
            uint256 remaining = (rewardRate * PRECISION - lastUpdateTime);
            uint256 leftover = remaining * rewardRate;
            rewardRate = (reward + leftover) / PRECISION;
        }

        lastUpdateTime = lastTimeRewardApplicable();

        emit RewardRateUpdated(rewardRate);
    }

    function setRewardRate(uint256 newRate) external onlyOwner {
        require(newRate > 0, "Reward rate must be greater than 0");
        rewardRate = newRate;
        emit RewardRateUpdated(newRate);
    }

    function recoverERC20(address tokenAddress, uint256 tokenAmount) external onlyOwner {
        require(tokenAddress != address(lpToken) && tokenAddress != address(rewardToken), "Cannot withdraw reward or lp tokens");
        IERC20(tokenAddress).safeTransfer(owner(), tokenAmount);
    }
}