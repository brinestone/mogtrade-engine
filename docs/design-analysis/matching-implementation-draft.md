# matching-implementation-draft.md

## Matching Engine Implementation Draft

The matching engine is responsible for processing buy and sell orders to find matching opportunities. The core logic lives in `services/matching/engine.go` where the `findMatches()` method is currently a stub.

### Current State

The matching engine has:
- A `MatchingEngine` struct with order book management
- `PlaceMatchOrder` method that adds orders to the appropriate book (bid or ask)
- Empty `findMatches()` method that needs to be implemented
- Order book structure with separate bid and ask books using mutex locks

### Implementation Requirements

The matching algorithm should:

1. **Get Best Prices**: Retrieve the best bid (highest buy price) and best ask (lowest sell price) from the respective order books
2. **Match Logic**: Match orders where the best bid >= best ask (buy price >= sell price)
3. **Match Quantity**: Match the minimum of the available quantities at the best prices
4. **Update Orders**: 
   - Reduce the quantity of matched orders
   - Remove orders with zero quantity
   - Create execution records for filled portions
5. **Emit Events**: Emit match events through the ExchangeHub for real-time updates

### Key Dependencies

- `services/matching/book.go`: Provides order book structure with bid/ask books, best price lookup methods
- `services/market/exchange-hub.go`: Provides event broadcasting mechanism
- `services/orders/service.go`: Provides execution creation functionality

### Implementation Strategy

The matching process should:

1. **Acquire Locks**: Get appropriate locks on both bid and ask books (consider lock ordering to avoid deadlocks)
2. **Get Best Prices**: Retrieve best bid price and quantity, best ask price and quantity
3. **Calculate Match Quantity**: Match quantity equals min(best bid qty, best ask qty)
4. **Match Orders**: 
   - Match orders from ask book (sellers) at best ask price
   - Match orders from bid book (buyers) at best bid price
5. **Update Orders**: 
   - Update order quantities
   - Remove orders with zero quantity
   - Create execution records for matched portions
6. **Emit Events**: Emit match events via ExchangeHub with relevant details

### Code Structure

The implementation should be added to `services/matching/engine.go` in the `findMatches()` method:

```go
func (e *MatchingEngine) findMatches() {
    // 1. Get best bid and ask
    bidLevel := e.orderBook.BestBid()
    askLevel := e.orderBook.BestAsk()
    
    if !bidLevel.Valid || !askLevel.Valid {
        return // No matches possible
    }
    
    // 2. Calculate match quantity
    matchQty := bidLevel.TotalVolume.Min(askLevel.TotalVolume)
    if matchQty.Equal(decimal.Zero) {
        return
    }
    
    // 3. Match orders
    // Implementation would iterate through order books to find matching orders
    // This is a simplified version for illustration
    
    // 3a. Match from ask book (sellers)
    // 3b. Match from bid book (buyers)
    
    // 4. Update order quantities
    // 5. Emit match events
}
```

### Critical Considerations

- **Lock Ordering**: Acquire locks in consistent order (e.g., bid book first, then ask book) to prevent deadlocks
- **Concurrency**: Use mutex locks appropriately to ensure thread safety
- **Order Matching Priority**: Consider time priority for order matching (FIFO within price levels)
- **Execution Records**: Create proper execution records with matched quantity, price, and timestamp
- **Event Emission**: Emit match events with relevant details for real-time updates

### Implementation Notes

The matching engine should handle:
- Partial matches (when only portion of order can be matched)
- Order cancellation during matching (need thread-safe handling)
- Multiple matches per order (orders can be matched against multiple counterparties)
- Time priority for orders at same price level

The implementation should be efficient and avoid unnecessary iterations through the entire order book.