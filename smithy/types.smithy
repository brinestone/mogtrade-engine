$version: "2"

namespace mogtrade.core.types

structure AvailabilityOutput {
    @required
    @documentation("Whether the requested resource identifier is available or not")
    available: Boolean
}

@error("client")
@httpError(409)
structure ConflictError {
    @required
    @documentation("The error message from the server")
    error: String
}

@error("client")
@httpError(400)
structure ValidationError {
    @required
    @documentation("A list of validation messages")
    error: ErrorMessages
}

@error("server")
@httpError(500)
structure InternalServerError {
    @required
    @documentation("The error message from the server")
    error: String
}

@error("client")
@httpError(422)
structure UnprocessibleError {
    @required
    @documentation("A list of validation messages")
    error: ErrorMessages
}

@error("client")
@httpError(401)
structure UnauthorizedError {
    @required
    error: String
}

@error("client")
@httpError(403)
@documentation("The authenticated user does not have the required role/permission")
structure ForbiddenError {
    @required
    error: String
}

@pattern("^[0-9A-HJKMNP-TV-Z]{26}$")
string Ulid

@pattern("^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+$")
string EmailAddress

@sensitive
@length(min: 6, max: 100)
string Password

list ErrorMessages {
    member: String
}

@error("client")
@httpError(404)
@documentation("A resource was not found")
structure NotFoundError {
    error: String
}

enum WalletType {
    VIRTUAL = "virtual"
    REAL = "real"
}

enum OrderStatus {
    PENDING = "pending"
    SUBMITTED = "submitted"
    PARTIALLY_FILLED="partially_filled"
    FILLED="filled"
    CANCELLED="cancelled"
    REJECTED="rejected"
    EXPIRED="expired"
}
enum OrderSide {
    BUY = "buy"
    SELL = "sell"
}
enum OrderType {
STOP = "stop"
LIMIT = "limit"
MARKET = "market"
STOP_LIMIT = "stop_limit"
}


@pattern("^\\-?[0-9]+(\\.[0-9]{1,8})?$")
string NumericString
@pattern("^[A-Z]{3}$")
string Currency

/// The gender of a KYC profile holder.
enum Gender {
    MALE = "male"
    FEMALE = "female"
    UNDEFINED = "undefined"
}

/// The verification status of a KYC record.
enum KYCStatus {
    /// Documents submitted; awaiting admin review.
    PENDING = "pending"
    /// An admin review is actively in progress.
    VERIFYING = "verifying"
    /// Verification was successful.
    VERIFIED = "verified"
    /// Verification was rejected.
    REJECTED = "rejected"
    /// The KYC record has expired and must be renewed.
    EXPIRED = "expired"
}

/// The risk level assigned to a customer after KYC verification.
enum KYCRiskProfile {
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
}
