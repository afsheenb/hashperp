# HashPerp: A Decentralized Bitcoin Hash Rate Derivatives Platform

## Overview

HashPerp is a decentralized platform enabling the creation, trading, and settlement of Bitcoin hash rate derivatives. By leveraging the Ark layer-2 protocol, the platform offers non-custodial, cryptographically secure contracts that allow users to trade contracts tied to the performance of the Bitcoin network’s hash rate. The platform operates using a sophisticated transaction structure that ensures trustless settlement, enables **dynamic joining of contracts**, and offers **off-chain VTXO swaps** for efficient risk management. 

## Key Components

### 1. **VTXOs** and **Connector Outputs**: Trustless Contract Mechanism

HashPerp leverages **VTXOs** (Virtual Transaction Outputs) as the core mechanism for managing contract positions. These VTXOs represent locked funds in a contract, and they can be transferred between participants without requiring on-chain transactions until it is time to settle or exit the contract. **Connector Outputs** provide the flexibility to link different participants into the contract, forming a trustless, non-custodial system for managing Bitcoin hash rate derivatives.

- **VTXO Swap**: A user (Alice) can **swap her VTXO** with another participant (Carol) off-chain, effectively transferring her claim to the contract. This swap does not trigger any on-chain transactions but allows Alice to hedge, adjust positions, or exit without immediate blockchain interaction. When Alice or Carol chooses to **settle** or **exit** the contract, an **on-chain transaction** is required to finalize the contract and settle the funds.

- **Connector Outputs**: These outputs link participants (e.g., Alice and Carol) to the contract. When a contract position changes hands, the output can be updated to reflect the new participant, ensuring that the blockchain always reflects the most current position in the contract.

### 2. **Exit Paths**: Safeguarding Against Griefing and Non-Cooperation

HashPerp includes several **exit paths** to safeguard participants from potential issues such as non-cooperation, griefing, or technical failures. These exit paths are structured with specific on-chain transactions that ensure users can always recover their funds in case a participant becomes unresponsive.

#### Exit Path Mechanisms:
1. **Cooperative Settlement**: 
   - The contract’s outcome is deterministically calculated based on **block height** and **timestamp** conditions. The **settlement transaction** is cooperatively signed by both parties.
   - If both parties cooperate, the contract is settled in a timely manner.

2. **Non-Cooperative Settlement**: 
   - If one party refuses to cooperate, after a **timeout period**, the contract enters an automatic exit phase, where the remaining party can proceed with settlement using their side of the contract.
   - **OP_CHECKSEQUENCEVERIFY (CSV)** locks the contract into a **timeout** condition, ensuring that if one party fails to sign, the contract can still be resolved after a set period.

3. **Pre-signed Exit Transactions**: 
   - Participants can create **pre-signed exit transactions** during contract setup. These transactions are stored securely and can be broadcast if necessary, allowing users to exit the contract if the other party becomes unresponsive or refuses to sign.
   - **VTXO sweeping** ensures that if there is any failure in settlement, users can recover their funds using their pre-signed transactions.

4. **ASP Failure Recovery**: 
   - The **ASP** (Application Service Provider) acts as a facilitator of the contract, but it cannot block the settlement process. If the ASP fails, the contract’s exit paths (including **VTXO sweeping** and pre-signed transactions) allow participants to exit without needing the ASP’s cooperation.

### 3. **Contract Rollovers and Dynamic Joining**

The **contract rollover** feature allows participants to seamlessly continue trading derivatives contracts based on updated conditions, without needing to manually exit and re-enter the system.

- **Dynamic Joining**: 
   - The system supports **dynamic joining** of new participants at any time during the life of a contract. If Alice (the option buyer) or Bob (the option seller) wants to exit the contract, they may **swap** their position with another participant (e.g., Carol), by transferring their VTXO.
   - The new participant (Carol) would then take over Alice’s or Bob’s position in the contract. This transfer happens **off-chain** without triggering an on-chain transaction. Only when Carol chooses to settle or exit the contract will the on-chain transaction be triggered.

- **Contract Rollover**: 
   - If a contract reaches its expiration or a participant decides to exit or hedge, the contract can be **rolled over** to a new participant or an updated contract. This allows participants to maintain exposure to the same hash rate derivative market without incurring unnecessary on-chain costs.
   - The **order book** facilitates matching new participants to contracts that are nearing expiration, ensuring liquidity and continuous market activity. 

### 4. **Settlement Mechanisms: Coinflip Transaction Structure**

Contracts are settled based on a **Coinflip transaction structure**. This structure ensures deterministic outcomes based on the **target block height** and **timestamp**.

#### Coinflip Transaction Flow:
1. **Setup Transaction**: 
   - Inputs: VTXOs from both Party A (option buyer) and Party B (option seller).
   - Output: A script path that forces Party A to reveal their secret, with a fallback path for Party B to claim the funds after a timeout.

2. **Final Transaction**:
   - Inputs: The output from the Setup transaction.
   - Outputs: Conditional script paths for both Party A and Party B based on the revealed secrets. If the target block height is reached before the timestamp, Party A’s script path is activated; if the timestamp is reached before the block height, Party B’s script path is activated.

3. **Settlement Transaction**:
   - Input: The Final transaction output.
   - Signed by the winner, this transaction settles the contract and sends funds to the winner's address.

This structure ensures that:
- Neither party can cheat by seeing the other’s move first.
- The correct contract outcome is enforced based on blockchain data.
- The settlement process is automated and tamper-proof.

### 5. **Contract Swap Mechanism: Dynamic Participants**

In cases where a participant wishes to exit the contract or hedge their position, they can **swap** their position with a new participant using the **order book** or through a direct agreement. 

- **Order Book Matching**: 
   - The system’s order book matches **buyers and sellers** for contract positions, facilitating seamless contract swaps.
   - Once a swap is initiated, the new participant (e.g., Carol) takes the VTXO from the exiting participant (e.g., Alice), updating the contract to reflect the new participant’s involvement.

- **Direct Swaps**: 
   - Alice can directly swap her VTXO with Carol, transferring her position to Carol without needing to interact with the order book. This off-chain swap only requires an on-chain settlement transaction when Carol chooses to settle the contract.

### 6. **Safeguards Against Cheating and Griefing**

The system is designed to prevent any form of **cheating** or **griefing** by ensuring that contracts remain **trustless** and **non-custodial**.

#### Prevention Mechanisms:
1. **User Attempts to Cheat Another User**:
   - The **2-of-2 multisig** setup requires both signatures to settle the contract in the cooperative case.
   - The contract outcome is determined by blockchain data (hash rate, block height, timestamp).
   - Neither participant can alter the contract terms or outcome after agreement.

2. **User Attempts to Grief by Refusing to Cooperate**:
   - If a user refuses to cooperate, the **timeout** mechanism (via **OP_CHECKSEQUENCEVERIFY (CSV)**) automatically triggers the exit path, allowing the cooperative party to settle.
   - The **timeout path** becomes available after the specified period, ensuring settlement despite one party’s refusal to sign.

3. **ASP Attempts to Cheat or Grief Users**:
   - The **ASP** cannot block the settlement process, as it is not a required party for settlement. The **pre-signed exit transactions** and **VTXO sweeping** allow users to exit independently.
   - If the ASP fails, the **users can broadcast their pre-signed exit transactions** and recover their funds directly.

---

## Conclusion

The **HashPerp** system, based on the Ark protocol, offers a robust and **non-custodial** framework for trading Bitcoin hash rate derivatives. By using **VTXOs** and **connector outputs**, participants can freely manage their contract positions, hedge, and swap without requiring on-chain transactions unless settlement is needed. The platform’s **dynamic joining**, **contract rollover**, and **exit path mechanisms** provide flexibility and security, ensuring that users can always recover their funds in case of disputes or non-cooperation.


### **Project Plan for Implementing HashPerp: Bitcoin Hash Rate Derivatives on Layer 2**

---

### **Project Overview:**
HashPerp is a decentralized financial platform designed to allow the creation, trading, and settlement of perpetual futures contracts based on Bitcoin's hash rate. The platform provides miners and speculators with a new method to hedge against fluctuations in Bitcoin’s network hash rate by offering customized, dynamic contracts that operate via decentralized, trustless mechanisms. The platform is built upon Bitcoin’s Taproot capabilities, utilizing VTXOs (Virtual Transaction Outputs) for efficient and secure contract handling. HashPerp introduces innovative contract dynamics, such as exit paths, contract rollovers, and VTXO swaps, to optimize performance and provide flexibility for participants.

The platform will be built in three primary components:
1. **Backend** – Manages the core logic for contract creation, order matching, and contract lifecycle management.
2. **Frontend** – Provides a user interface for contract creation, management, and market interaction.
3. **Ark Service Provider (ASP) Modifications** – Adjustments to the ASP for decentralized transaction handling and contract validation.

---

### **1. Backend Development:**

The backend is responsible for managing the entire contract lifecycle, order matching, and the integration with Bitcoin’s blockchain and the Ark Protocol.

#### **1.1 Database Design:**
- **Contracts Table:**
  - Stores contract details such as strike hash rate, expiration block, contract type (CALL or PUT), parties involved, and any contract-specific metadata.
- **Order Book Table:**
  - Records buy and sell orders, tracking contract parameters and order status (open, matched, canceled).
- **VTXOs Table:**
  - Tracks the virtual transaction outputs (VTXOs) specific to contracts, including signatures, state transitions, and contract-specific data.
- **Transaction Table:**
  - Logs all contract-related transactions, including creation, settlement, swaps, and rollovers.
- **Market Data Table:**
  - Stores real-time Bitcoin hash rate data for use in contract pricing and settlement.

#### **1.2 Core Backend Features:**
- **Contract Creation and Management:**
  - Implement the logic for perpetual futures contracts with parameters like strike hash rate, expiration, contract type, and participants.
  - Enable dynamic joining of contracts by allowing users to opt into existing contracts with specific parameters, ensuring contract flexibility and market liquidity.
  - Generate Taproot script paths that reflect different contract states (high hash rate, low hash rate, dispute resolution).
  - Implement contract rollover functionality, allowing contracts to seamlessly transition at expiration into new, renewed positions.
  
- **VTXO Swap Mechanism:**
  - Support secure, trustless VTXO swaps to transfer contract ownership or adjust participant roles.
  - Implement a signature-based mechanism for updating contract VTXOs, ensuring all parties are aligned before swapping.
  - Manage contract rollovers, where the original contract may transition into a new contract of similar terms or customized parameters.

- **Exit Paths:**
  - Design contract exit paths that allow users to exit contracts before expiration, such as via buyout options or negotiated terms between parties. 
  - Implement exit strategies for both the initiating and counterparty, offering flexible pathways to liquidate contracts early.

#### **1.3 Dynamic Contract Joining:**
- **Joining Existing Contracts:**
  - Allow users to dynamically join open positions in perpetual futures contracts, either by agreeing to the current contract terms or negotiating specific modifications (such as strike price or expiration).
  - Enable participants to buy or sell positions without needing to create a new contract from scratch, enhancing liquidity and market efficiency.

#### **1.4 API Extensions:**
- **Contract Lifecycle APIs:**
  - Endpoints for creating, managing, settling, and swapping contracts, including dynamic joining and contract rollover.
- **Order Book APIs:**
  - Endpoints for placing, modifying, canceling, and viewing orders, including real-time order book status.
- **Market Data APIs:**
  - Endpoints to fetch Bitcoin hash rate data and contract pricing.
- **Event Streams:**
  - Real-time updates via WebSockets for contract lifecycle events, including swaps, rollovers, and settlements.

---

### **2. Frontend Development:**

The frontend provides a user-friendly interface for interacting with HashPerp, creating contracts, managing positions, and tracking market activity.

#### **2.1 Contract Management Interface:**
- **Contract Creation Wizard:**
  - A guided, step-by-step process for users to create new perpetual futures contracts, select parameters (e.g., strike hash rate, expiration block, type), and determine contract size.
- **Dynamic Joining Interface:**
  - A user interface to view existing open positions and join contracts with similar terms. Users can dynamically join contracts through a simple negotiation and approval process.
- **Contract Rollover Interface:**
  - A dashboard displaying contracts nearing expiration, with options to roll over into new positions based on the existing contract terms or modified parameters.
- **Portfolio View:**
  - A portfolio section showing active contracts, their status, current value, and performance over time.
  
#### **2.2 Order Book Interface:**
- **Real-time Order Book Visualization:**
  - Display live buy and sell orders, with real-time updates on matching, execution, and contract details.
  - Include depth charts, price movements, and order status.
- **Order Management Tools:**
  - Tools to place, modify, and cancel orders, including partial fills and expirations.
- **Matched Orders History:**
  - A history of all past matched orders, showing contract details, pricing, and settlement outcomes.

#### **2.3 Market Data Visualization:**
- **Hash Rate Charts:**
  - Visual representation of Bitcoin hash rate trends, allowing users to track performance over time and forecast potential contract outcomes.
- **Contract Price History:**
  - Display historical contract prices based on Bitcoin hash rate movements, providing users insight into market trends and potential opportunities.

#### **2.4 Swap and Exit Path Interfaces:**
- **Swap Interface:**
  - A feature that allows users to swap contract positions with others, displaying potential counterparties and negotiated terms for swaps.
  - Users can propose and accept swap offers via a bidding/negotiation interface.
- **Exit Path Management:**
  - An intuitive interface for managing contract exits, whether by early buyout or agreement with another counterparty.

---

### **3. ASP Modifications:**

Modifications to the Ark Service Provider (ASP) will ensure secure, decentralized contract validation, VTXO management, and transaction enforcement.

#### **3.1 Contract Support:**
- **Contract Data Structure:**
  - Extend the ASP to manage hash rate contract data structures, including dynamic joining, VTXO management, contract rollovers, and swaps.
  - Update VTXO handling to ensure contract-specific VTXOs are tracked and validated during lifecycle events, including swaps and rollovers.

- **Contract Validation:**
  - Ensure that all contract terms and parameters are validated before settlement, ensuring they comply with defined conditions and signatures.
  
- **Hash Rate Monitoring:**
  - Integrate with Bitcoin node data for real-time hash rate tracking to facilitate contract settlement and ensure accurate pricing.

#### **3.2 Order Matching and Swap Mechanism:**
- **Order Matching:**
  - Extend ASP logic to handle matching of buy and sell orders based on contract parameters.
  - Automatically generate contracts when orders match.
- **Swap Transaction Logic:**
  - Implement logic to securely swap contract positions, ensuring signatures are valid and contract VTXOs are updated accordingly.
  - Enable atomic swaps for VTXOs to ensure consistency and security during swaps.

- **Contract Rollovers:**
  - Implement logic for seamlessly transitioning expired contracts into new contracts, carrying forward key terms and adjusting parameters as necessary.

#### **3.3 API Extensions:**
- **Lifecycle Management APIs:**
  - APIs to manage all aspects of contract lifecycle, including creation, settlement, swaps, dynamic joining, and rollovers.
- **Order Book APIs:**
  - Provide endpoints for interacting with the order book, including order placement, modification, cancellation, and viewing of status.
- **Real-Time Event Streams:**
  - Extend event streams to notify users of contract status, order matching, VTXO swaps, and other lifecycle events in real-time.

---

### **Conclusion:**

HashPerp will provide a flexible, decentralized platform for Bitcoin hash rate derivatives, using perpetual futures contracts as the primary financial instrument. The platform’s dynamic contract features, such as exit paths, contract rollovers, and VTXO swaps, will enable more fluid market participation, while the backend and ASP modifications will ensure secure, trustless transaction handling. By leveraging Bitcoin’s Taproot and VTXO capabilities, HashPerp will empower users to hedge against hash rate volatility through customizable, decentralized financial contracts.

This project will provide a robust foundation for a secure, scalable, and innovative platform that meets the needs of miners and speculators in the Bitcoin ecosystem.


## **What We Are Trading**
#
To make the user experience smoother and more intuitive, the frontend interface should abstract away the complexity of block times and block heights, and focus on the metric that users care about: "BTC per PetaHash Per Day." Here’s a proposed approach:

### **1. Contract Creation Interface:**
- **Specifying Strike Rate:**
  - **Metric:** Users should input the strike rate in terms of "BTC per PetaHash Per Day" (e.g., 0.05 BTC/pH/day).
  - **Graphical Representation:** A slider or input field where users can specify their strike rate. A chart can show historical trends of BTC per PetaHash per day to give context.
  
- **Contract Duration:**
  - Instead of block height or block time, provide a time window (e.g., 7 days, 30 days) and show when the contract will expire in terms of the target date and time.
  - A visual countdown or timeline could help users understand the life of the contract.

- **Order Book Visualization:**
  - When users place orders, they should see a live order book with bid and ask prices in BTC/pH/day, making it clear what others are willing to pay or accept.
  - For each contract, indicate the expected settlement price (the BTC per PetaHash per day) and the number of orders available for a specific price.

### **2. Contract Details & Trading Interface:**
- **Show Current Market Price:**
  - The interface should display the live BTC/pH/day rate for contracts that are actively being traded, helping users understand the current market price for the underlying metric.
  - **Historical Data Graph:** Provide users with charts showing how the BTC/pH/day rate has evolved over time, allowing them to visualize trends and make informed decisions.

- **Market Segmentation:**
  - **Active Contracts:** Show users all contracts in play, with their strike rate, expiry date, and the current BTC/pH/day rate.
  - **Position Overview:** Users should be able to view their position(s) in active contracts with metrics like the strike rate and their unrealized profit or loss based on the current market rate.

- **Trade History & Settlement:**
  - Provide a section where users can view their completed contract settlements, showing how their position performed (with both initial strike rate and final settlement rate in BTC/pH/day).
  - Display the total BTC earned based on their speculation.

### **3. Hedging or Rolling Over Contracts:**
- When a contract is nearing its expiration, users should be able to easily hedge or roll it over with another contract at the current market rate for BTC/pH/day. 
  - The system can prompt users to roll over or hedge at the current market rate, providing a seamless way to stay engaged with the market.
  - This feature could also display an automatic price recommendation based on the user's position and the market rate.

### **4. Contract Swap Interface (for VTXO Swaps):**
- **Swap Contract Interface:** 
  - If Alice wants to swap her position with Carol, the interface should let Alice select Carol's offer from a list of available swaps (based on the BTC/pH/day rate).
  - The visual indicator for contract swaps should show which side of the contract Alice is taking (e.g., buying or selling), as well as the BTC/pH/day rate and the expiry date for the new contract.
  - **Swap History:** Display a list of all previous swaps, indicating how they performed and whether users made a profit/loss.

### **5. Notifications & Alerts:**
- **Price Alerts:** Allow users to set alerts when the BTC/pH/day rate reaches a certain threshold that they are watching.
- **Expiry Reminders:** Notify users as their contracts near expiration and remind them of any potential action they may want to take (e.g., roll over, hedge, exit).

### **6. Contract Exits:**
- When it’s time to exit a contract, the interface should allow users to choose their exit strategy:
  - **Exit at Settlement:** If the contract expires, show the settlement amount based on BTC/pH/day rate.
  - **Exit via Swap:** If they’re selling their VTXO, offer an option to list it on the market for other users to take over the position.

By focusing on "BTC per PetaHash Per Day" as the primary metric and removing the complexity of block height and block time, the user interface can present a clean and clear experience for traders. The goal is to ensure users are always making decisions based on easily understood and relevant data.
