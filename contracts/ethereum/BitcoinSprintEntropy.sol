// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title BitcoinSprintEntropy
 * @dev Smart contract for providing cryptographically secure entropy as a service
 * @author BitcoinCab.inc
 * @notice This contract enables on-chain entropy generation with payment processing
 */
contract BitcoinSprintEntropy {
    address public owner;
    address public treasury;
    uint256 public requestCount;
    uint256 public constant MIN_PAYMENT = 0.001 ether;
    uint256 public fulfillmentTimeout = 300; // 5 minutes
    
    struct EntropyRequest {
        address requester;
        uint256 timestamp;
        bytes32 entropyHash;
        uint256 payment;
        bool fulfilled;
        bool refunded;
        uint8 qualityTier; // 1=basic, 2=pro, 3=enterprise
    }
    
    mapping(uint256 => EntropyRequest) public requests;
    mapping(address => uint256[]) public userRequests;
    mapping(address => bool) public authorizedFulfillers;
    
    // Events
    event EntropyRequested(
        uint256 indexed requestId, 
        address indexed requester, 
        uint256 payment,
        uint8 qualityTier
    );
    event EntropyFulfilled(
        uint256 indexed requestId, 
        bytes32 indexed entropyHash,
        uint256 qualityScore
    );
    event RefundIssued(uint256 indexed requestId, address requester, uint256 amount);
    event TreasuryUpdated(address oldTreasury, address newTreasury);
    
    // Modifiers
    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }
    
    modifier onlyAuthorizedFulfiller() {
        require(authorizedFulfillers[msg.sender] || msg.sender == owner, "Not authorized");
        _;
    }
    
    constructor(address _treasury) {
        owner = msg.sender;
        treasury = _treasury;
        authorizedFulfillers[msg.sender] = true;
    }
    
    /**
     * @dev Request entropy with payment
     * @param qualityTier 1=basic (0.001 ETH), 2=pro (0.005 ETH), 3=enterprise (0.01 ETH)
     * @return requestId Unique identifier for the entropy request
     */
    function requestEntropy(uint8 qualityTier) external payable returns (uint256) {
        require(qualityTier >= 1 && qualityTier <= 3, "Invalid quality tier");
        
        uint256 requiredPayment = getRequiredPayment(qualityTier);
        require(msg.value >= requiredPayment, "Insufficient payment");
        
        uint256 requestId = ++requestCount;
        requests[requestId] = EntropyRequest({
            requester: msg.sender,
            timestamp: block.timestamp,
            entropyHash: bytes32(0),
            payment: msg.value,
            fulfilled: false,
            refunded: false,
            qualityTier: qualityTier
        });
        
        userRequests[msg.sender].push(requestId);
        
        // Transfer payment to treasury
        payable(treasury).transfer(msg.value);
        
        emit EntropyRequested(requestId, msg.sender, msg.value, qualityTier);
        return requestId;
    }
    
    /**
     * @dev Fulfill entropy request (only authorized fulfillers)
     * @param requestId Request to fulfill
     * @param entropy Generated entropy value
     * @param qualityScore Quality assessment (0-10000, representing 0.00-100.00%)
     */
    function fulfillEntropy(
        uint256 requestId, 
        bytes32 entropy,
        uint256 qualityScore
    ) external onlyAuthorizedFulfiller {
        require(requestId > 0 && requestId <= requestCount, "Invalid request ID");
        require(!requests[requestId].fulfilled, "Already fulfilled");
        require(!requests[requestId].refunded, "Already refunded");
        require(entropy != bytes32(0), "Invalid entropy");
        require(qualityScore <= 10000, "Invalid quality score");
        
        requests[requestId].entropyHash = entropy;
        requests[requestId].fulfilled = true;
        
        emit EntropyFulfilled(requestId, entropy, qualityScore);
    }
    
    /**
     * @dev Get entropy for fulfilled request
     * @param requestId Request identifier
     * @return entropy The generated entropy value
     */
    function getEntropy(uint256 requestId) external view returns (bytes32) {
        require(requestId > 0 && requestId <= requestCount, "Invalid request ID");
        require(requests[requestId].fulfilled, "Not fulfilled");
        require(
            requests[requestId].requester == msg.sender || msg.sender == owner, 
            "Not authorized"
        );
        
        return requests[requestId].entropyHash;
    }
    
    /**
     * @dev Request refund for unfulfilled entropy (after timeout)
     * @param requestId Request to refund
     */
    function requestRefund(uint256 requestId) external {
        require(requestId > 0 && requestId <= requestCount, "Invalid request ID");
        require(requests[requestId].requester == msg.sender, "Not your request");
        require(!requests[requestId].fulfilled, "Already fulfilled");
        require(!requests[requestId].refunded, "Already refunded");
        require(
            block.timestamp >= requests[requestId].timestamp + fulfillmentTimeout,
            "Timeout not reached"
        );
        
        requests[requestId].refunded = true;
        uint256 refundAmount = requests[requestId].payment;
        
        payable(msg.sender).transfer(refundAmount);
        
        emit RefundIssued(requestId, msg.sender, refundAmount);
    }
    
    /**
     * @dev Get required payment for quality tier
     * @param qualityTier Quality tier (1-3)
     * @return Required payment amount in wei
     */
    function getRequiredPayment(uint8 qualityTier) public pure returns (uint256) {
        if (qualityTier == 1) return 0.001 ether; // Basic
        if (qualityTier == 2) return 0.005 ether; // Pro
        if (qualityTier == 3) return 0.01 ether;  // Enterprise
        revert("Invalid quality tier");
    }
    
    /**
     * @dev Get user's request history
     * @param user User address
     * @return Array of request IDs
     */
    function getUserRequests(address user) external view returns (uint256[] memory) {
        return userRequests[user];
    }
    
    /**
     * @dev Check if request can be refunded
     * @param requestId Request to check
     * @return Whether refund is available
     */
    function canRefund(uint256 requestId) external view returns (bool) {
        if (requestId == 0 || requestId > requestCount) return false;
        if (requests[requestId].fulfilled || requests[requestId].refunded) return false;
        return block.timestamp >= requests[requestId].timestamp + fulfillmentTimeout;
    }
    
    // Admin functions
    function addAuthorizedFulfiller(address fulfiller) external onlyOwner {
        authorizedFulfillers[fulfiller] = true;
    }
    
    function removeAuthorizedFulfiller(address fulfiller) external onlyOwner {
        authorizedFulfillers[fulfiller] = false;
    }
    
    function updateTreasury(address newTreasury) external onlyOwner {
        require(newTreasury != address(0), "Invalid treasury");
        address oldTreasury = treasury;
        treasury = newTreasury;
        emit TreasuryUpdated(oldTreasury, newTreasury);
    }
    
    function updateFulfillmentTimeout(uint256 newTimeout) external onlyOwner {
        require(newTimeout >= 60 && newTimeout <= 3600, "Invalid timeout"); // 1 min to 1 hour
        fulfillmentTimeout = newTimeout;
    }
    
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Invalid owner");
        owner = newOwner;
    }
    
    // Emergency functions
    function emergencyWithdraw() external onlyOwner {
        payable(owner).transfer(address(this).balance);
    }
    
    function pause() external onlyOwner {
        // Implementation for emergency pause functionality
        // This would be implemented with a paused state variable
    }
    
    // View functions
    function getContractStats() external view returns (
        uint256 totalRequests,
        uint256 totalFulfilled,
        uint256 contractBalance
    ) {
        uint256 fulfilled = 0;
        for (uint256 i = 1; i <= requestCount; i++) {
            if (requests[i].fulfilled) fulfilled++;
        }
        
        return (requestCount, fulfilled, address(this).balance);
    }
}
