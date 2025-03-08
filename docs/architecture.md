# HashPerp: Bitcoin Hash Rate Derivatives Platform

## Executive Summary

HashPerp is a decentralized platform for creating, trading, and settling Bitcoin hash rate derivatives. Built on the Ark layer-2 protocol, HashPerp enables miners and investors to hedge against or speculate on Bitcoin network hash rate fluctuations using perpetual futures contracts. The platform's innovative design features include dynamic joining of contracts, off-chain VTXO swaps, and multiple exit paths to ensure a resilient and user-friendly trading system.

## Core Value Proposition

HashPerp addresses the following needs in the Bitcoin ecosystem:

1. **Hash Rate Risk Management**: Miners can hedge against hash rate fluctuations that directly impact their profitability.
2. **Market Speculation**: Traders can take positions on Bitcoin's network health and security without operating mining hardware.
3. **Capital Efficiency**: The layer-2 solution minimizes on-chain transactions, reducing costs and increasing speed.
4. **Non-custodial Trading**: All contracts are secured by Bitcoin's native security model, eliminating counterparty risk.

## Key Features

### 1. VTXO-based Contract System

The platform leverages Virtual Transaction Outputs (VTXOs) as the core mechanism for managing contract positions. VTXOs represent locked funds in a contract and can be transferred between participants without requiring on-chain transactions until settlement or exit.

### 2. Dynamic Contract Participation

Users can join existing contracts at any point during their lifecycle by:
- Taking over a position from another participant through a VTXO swap
- Creating a new contract that matches their desired parameters

### 3. Off-chain VTXO Swaps

The VTXO swap mechanism allows contract positions to change hands entirely off-chain, enabling:
- Position transfers without blockchain transactions
- Efficient risk management and hedging
- Contract rollovers to maintain market exposure

### 4. Multiple Exit Paths

To protect users from potential issues, HashPerp includes several exit paths:
- Cooperative Settlement: Normal settlement when both parties cooperate
- Non-Cooperative Settlement: Automatic exit after a timeout period
- Pre-signed Exit Transactions: Ready-to-broadcast transactions for emergency exits
- VTXO Sweeping: Recovery mechanism if other exit paths fail

### 5. User-Friendly Interface

The frontend abstracts complex blockchain concepts and presents information in terms that matter to traders:
- "BTC per PetaHash Per Day" as the primary trading metric
- Human-readable contract durations instead of block heights
- Intuitive visualization of market conditions and contract status

## System Architecture

HashPerp consists of the following components:

### 1. Backend Core

- **ContractManager**: Handles contract lifecycle from creation to settlement
- **VTXOManager**: Manages Virtual Transaction Outputs and their state transitions
- **OrderBookManager**: Facilitates order matching and trading
- **SwapOfferManager**: Enables position swapping between participants
- **MarketDataManager**: Tracks Bitcoin hash rate and related metrics
- **TransactionManager**: Records and retrieves all system transactions
- **ScriptGenerator**: Creates Bitcoin scripts for contract operations

### 2. Frontend

- **Contract Management Interface**: UI for creating and managing contracts
- **Order Book Interface**: Trading interface with price visualization
- **Swap Interface**: Position transfer and contract rollover tools
- **Portfolio Visualization**: Track performance and positions

### 3. ASP Modifications

- **Contract Support**: Extensions to the Ark protocol for hash rate derivatives
- **Order Matching**: Mechanism to pair buyers and sellers
- **Swap Mechanism**: Support for trustless position transfers
- **Hash Rate Monitor**: Integration with Bitcoin network data

## Technical Details

### Data Model

The system's core data structures include:

- **Contract**: Represents a hash rate derivative contract with parameters like strike rate, expiry, and participants
- **VTXO**: Represents a virtual output in the system that can be transferred between users
- **Order**: Represents a buy or sell order in the order book
- **SwapOffer**: Represents an offer to swap a contract position
- **Transaction**: Records all system activities for auditing and analysis
- **HashRateData**: Stores Bitcoin network hash rate information

### Settlement Process

Contracts are settled based on a Coinflip transaction structure:

1. **Setup Transaction**: Combines inputs from both parties with specific script conditions
2. **Final Transaction**: Determines the winner based on block height and timestamp conditions
3. **Settlement Transaction**: Finalizes the contract and distributes funds

### VTXO Swap Process

The dynamic joining feature relies on the VTXO swap process:

1. Original participant creates a swap offer
2. New participant accepts the offer
3. System creates a new VTXO for the new participant
4. Original VTXO is marked inactive
5. Contract is updated with the new participant
6. The swap is recorded in the transaction history

## Development Roadmap

The implementation of HashPerp will proceed in phases:

1. **Foundation Phase**:
   - Core data structures and interfaces
   - Basic contract management functionality
   - Repository implementations

2. **Functionality Phase**:
   - Order book and matching engine
   - VTXO swap mechanism
   - Exit path implementation

3. **Integration Phase**:
   - ASP modifications
   - Bitcoin script generation
   - Frontend development

4. **Testing and Deployment Phase**:
   - Security audits
   - Testnet deployment
   - Mainnet launch

## Conclusion

HashPerp represents a significant advancement in decentralized derivatives trading on Bitcoin. By leveraging the Ark layer-2 protocol and innovative VTXO-based contract mechanism, HashPerp enables efficient, secure, and flexible trading of hash rate derivatives. The platform's dynamic joining feature and multiple exit paths ensure a robust trading experience that meets the needs of both miners and investors in the Bitcoin ecosystem.
