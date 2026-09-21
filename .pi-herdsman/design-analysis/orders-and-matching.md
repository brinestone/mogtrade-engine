# orders-and-matching.md

## Order Management & Matching Architecture

The order management system in MogTrade handles order lifecycle from placement to execution, with a focus on matching buy and sell orders.

### Order Lifecycle

Orders flow through these key stages:

1. **Order Placement**: Client submits order via API
2. **Validation**: Risk engine validates order against business rules
3. **Order Storage**: Order is stored in appropriate order book (bid or ask)
4. **Matching**: Engine finds matching counterparties
5. **Execution**: Orders are filled and execution records created
6. **Post-Match**: Order status updated, remaining quantity handled

### Order Types

- **Market Orders**: Immediate execution at best available price
- **Limit Orders**: Execute only at specified price or better
- **Stop Orders**: Triggered when price reaches threshold
- **Stop-Limit Orders**: Combines stop and limit order features

### Order Book Structure

The matching engine uses a dual-book approach:

- **Bid Book**: Contains buy orders (highest price at top)
- **Ask Book**: Contains sell orders (lowest price at top)
- **Price Levels**: Orders grouped by price with time priority
- **Order Matching**: Occurs when bid price >= ask price

### Matching Algorithm

The matching process involves:

1. **Price-Time Priority**: Matches occur at best available price, with time priority for same-price orders
2. **Quantity Matching**: Match quantity equals minimum of available quantities
3. **Order Updates**: 
   - Matched orders have reduced quantities
   - Completed orders are removed
   - New orders may be created for remaining portions
4. **Event Emission**: Match events published via ExchangeHub

### Key Components

- **MatchingEngine**: Core struct managing order books and matching logic
- **OrderBook**: Manages bid/ask books with mutex protection
- **RiskEngine**: Validates orders before matching
- **ExchangeHub**: Broadcasts match events to clients
- **ExecutionService**: Creates execution records for filled orders

### Implementation Notes

The `findMatches()` method in `services/matching/engine.go` needs implementation to:

1. Get best bid and ask prices
2. Calculate match quantity
3. Match orders at best available price
4. Update order quantities
5. Emit match events

The current stub method should be replaced with production-ready matching logic that handles:
- Partial matches
- Order cancellation during matching
- Time priority for same-price orders
- Thread-safe operations with proper locking