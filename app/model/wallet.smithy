$version: "2"
namespace mograde.order

service Billing {
    version: "2026-02-09"
    resources: [
Wallet
    ]
}

resource Wallet {
    identifiers: {walletId: Ulid}
    properties: {
        @required
        ownerId: Ulid

        @required
        startingBalance: Float

        @required
        currentBalance: Float
        
        @required
        totalTransactions: Int

        @required
        lastActivityAt timestamp
    }
    read: GetTransaction
    list: ListTransactions
}

@pattern("^[0-9A-HJ-TV-Zabcdefghjkmnp-tv-z]{26}$")
string Ulid 