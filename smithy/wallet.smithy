$version: "2"

namespace mogtrade.economy

use aws.protocols#restJson1
use mogtrade.core.types#InternalServerError
use mogtrade.core.types#ResourceId
use mogtrade.core.types#UnauthorizedError
use mogtrade.core.types#ValidationError
use smithy.api#readonly

@restJson1
@httpBearerAuth
@title("Mogtrade Economy")
service Economy {
    version: "1.0.0"
    errors: [
        ValidationError
        InternalServerError
        UnauthorizedError
    ]
    operations: [
        GetWalletInfo
    ]
}

@readonly
@tags(["Economy"])
@http(method: "GET", uri: "/api/v1/wallet/{type}")
operation GetWalletInfo {
    input: GetWalletInfoInput
    output: GetWalletInfoOutput
}

@input
structure GetWalletInfoInput {
    @required
    @httpLabel
    type: WalletType
}

enum WalletType {
    VIRTUAL = "virtual"
    REAL = "real"
}

@output
structure GetWalletInfoOutput {
    balance: Balance
    id: ResourceId
    totalTransactions: Integer
    lastActivityAt: Timestamp
}

@pattern("^[0-9]{0,14}\\.?[0-9]{1,4}$")
string Balance
