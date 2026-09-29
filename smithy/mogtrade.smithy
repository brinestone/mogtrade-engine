$version: "2"

namespace mogtrade.api

use aws.protocols#restJson1
use smithy.api#readonly
use smithy.api#resourceIdentifier
use mogtrade.core.types#InternalServerError
use mogtrade.core.types#WalletType
use mogtrade.core.types#NumericString
use mogtrade.core.types#Password
use mogtrade.core.types#EmailAddress
use mogtrade.core.types#AvailabilityOutput
use mogtrade.core.types#OrderType
use mogtrade.core.types#OrderSide
use mogtrade.core.types#ConflictError
use mogtrade.core.types#OrderStatus
use mogtrade.core.types#Ulid
use mogtrade.core.types#UnprocessibleError
use mogtrade.core.types#UnauthorizedError
use mogtrade.core.types#NotFoundError
use mogtrade.core.types#ValidationError

@restJson1
@httpBearerAuth
@title("MogTrade API")
service Api {
    version: "1.0.0"
    errors: [
        ValidationError
        InternalServerError
    ]
    operations: [
        GetWalletInfo
        CredentialSignIn
        CredentialSignUp
        RotateAccessToken
        CheckEmailAvailable
    ]
    resources: [
        Order
    ]
}

@readonly
@tags(["Economy"])
@auth([httpBearerAuth])
@http(method: "GET", uri: "/api/v1/wallet/{type}")
operation GetWalletInfo {
    input := {
        @required
        @httpLabel
        type: WalletType
    }
    output := {
    @required
    balance: NumericString
    @required
    walletId: Ulid
    @required
    totalTransactions: Long
    lastActivityAt: Timestamp
    }
    errors: [
        NotFoundError
    ]
}

resource Order {
    identifiers: {id: OrderId}
    list: GetOrders
    read: FindOrder
    create: PlaceOrder
    properties: {
        symbol: String
        side: OrderSide
        currency: String
        type: OrderType
        qty: NumericString
        limitPrice: NumericString
        stopPrice: NumericString
        status: OrderStatus
        createdAt: Timestamp
        updatedAt: Timestamp
    }
}

@mixin
structure OrderIdentifiers for Order {
    @required
    $id
}

@mixin
structure OrderTimestamps for Order {
    @required
    $createdAt
    @required
    $updatedAt
}
@mixin
structure OrderData for Order {
    @required
    $symbol
    @required
    $side
    @required
    $currency
    @required
    $type
    @required
    $qty
    $limitPrice
    $stopPrice
    @required
    $status
}

@mixin
structure OrderRecord with [OrderIdentifiers, OrderData, OrderTimestamps] {}

@pattern("^[0-9A-HJKMNP-TV-Z]{26}$")
string OrderId

@auth([httpBearerAuth])
@tags(["Order"])
@http(method: "POST", uri: "/api/v1/orders", code: 202)
operation PlaceOrder {
    input: PlaceOrderInput
    errors: [
        UnprocessibleError
    ]
}

@input
structure PlaceOrderInput with [OrderData] {
    @required
    @notProperty
    @httpHeader("X-Idempotency-Token")
    clientId: String

    @required
    @notProperty
    walletType: WalletType
}

@auth([])
@readonly
@tags(["Order"])
@auth([httpBearerAuth])
@http(method: "GET", uri: "/api/v1/orders/{id}")
operation FindOrder {
    input := for Order {
        @required
        @httpLabel
        $id
    } 
    output := with [OrderRecord] {}
    errors: [
        NotFoundError
    ]
}

@auth([])
@readonly
@tags(["Order"])
@auth([httpBearerAuth])
@http(method: "GET", uri: "/api/v1/orders")
@paginated(items:"data",outputToken: "nextCursor", inputToken: "cursor", pageSize: "limit")
operation GetOrders {
    input: GetOrdersInput
    output: GetOrdersOutput
}

@input
structure GetOrdersInput {
    @httpQuery("limit")
    limit: Integer = 100
    @httpQuery("cursor")
    cursor: OrderId
}

@output
structure GetOrdersOutput {
    nextCursor: OrderId
    @required
    data: OrderLookupList
}

list OrderLookupList {
    member: OrderLookup
}

structure OrderLookup with [OrderRecord]{}

@auth([])
@tags(["Auth"])
@http(method: "POST", uri: "/api/v1/auth/login/credential")
@documentation("Sign in users by obtaining a JWT access token and a refresh token")
operation CredentialSignIn {
    input: CredentialSignInInput
    output: SignInOutput
    errors: [
        UnprocessibleError
        UnauthorizedError
    ]
}

structure SignInOutput {
    @required
    @documentation("The  access token (JWT) granted to the user")
    accessToken: String

    @required
    @documentation("The refresh token")
    refreshToken: String
}

@input
structure CredentialSignInInput {
    @required
    @documentation("The user's identifying email address")
    email: EmailAddress

    @required
    @documentation("The user's password")
    password: Password

    @required
    @httpHeader("X-D-Id")
    @documentation("The client device's ID")
    deviceId: String
}

@auth([])
@tags(["Auth"])
@documentation("Create user account using credentials")
@http(method: "POST", uri: "/api/v1/auth/register/credential", code: 201)
operation CredentialSignUp {
    input: CredentialSignUpInput
    errors: [
        ConflictError
    ]
}

@input
structure CredentialSignUpInput {
    @required
    @documentation("The names of the user")
    names: String

    @required
    @documentation("The email address of the user. NOTE: must be unique")
    email: EmailAddress

    @required
    @documentation("User's password")
    password: Password

    @required
    @documentation("Password confirmation")
    confirmPassword: Password
}

@auth([])
@readonly
@tags(["Auth"])
@documentation("Rotate access token")
@http(method:"GET", uri:"/api/v1/auth/refresh")
operation RotateAccessToken {
    input: RotateAccessTokenInput
    output: SignInOutput
    errors: [
        UnauthorizedError
    ]
}

@input
structure RotateAccessTokenInput {
    @required
    @httpHeader("X-Refresh-Token")
    @documentation("The refresh token from previous sign in/refresh requests")
    refreshToken: String

    @required
    @documentation("The device id")
    @httpHeader("X-d-Id")
    deviceId: String
}

@auth([])
@readonly
@tags(["Auth"])
@http(method: "GET", uri: "/api/v1/auth/email-available")
@documentation("Check whether an email is available for a user to create an account with")
operation CheckEmailAvailable{
    input: CheckEmailAvailableInput
    output: AvailabilityOutput
}

@input
structure CheckEmailAvailableInput {
    @required
    @httpQuery("email")
    @documentation("The email address query")
    email: String
}