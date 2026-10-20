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
use mogtrade.core.types#ForbiddenError
use mogtrade.core.types#Gender
use mogtrade.core.types#KYCStatus
use mogtrade.core.types#KYCRiskProfile

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
        RequestPasswordReset
        ConfirmPasswordReset
        SubmitPasswordReset
        BeginKyc
        GetKycStatus
        UpdateKycProfile
        UpdateKycDocuments
        GetPresignedUploadUrl
        ListKycApplications
        BeginKycVerification
        ApproveKyc
        RejectKyc
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

// ─────────────────────────────────────────────────────────────────────────────
// Password Reset
// ─────────────────────────────────────────────────────────────────────────────

@auth([])
@tags(["Auth"])
@http(method: "POST", uri: "/api/v1/auth/password-reset/request", code: 202)
@documentation("Initiate a password-reset flow. Sends a reset link to the supplied email address if an account exists.")
operation RequestPasswordReset {
    input: RequestPasswordResetInput
}

@input
structure RequestPasswordResetInput {
    @required
    @documentation("The email address of the account to reset")
    email: EmailAddress

    @required
    @documentation("A URL the server will embed in the reset email. The server appends ?vt=<token> to this URL before redirecting.")
    callbackUrl: String
}

@auth([])
@readonly
@tags(["Auth"])
@http(method: "GET", uri: "/api/v1/auth/password-reset/confirm")
@documentation("Validates a password-reset token and redirects to the client-supplied callback URL. Marks the verification token as primed so it can only be used once.")
operation ConfirmPasswordReset {
    input: ConfirmPasswordResetInput
    errors: [
        NotFoundError
        UnprocessibleError
    ]
}

@input
structure ConfirmPasswordResetInput {
    @required
    @httpQuery("token")
    @documentation("The raw verification token received via email")
    token: String
}

@auth([])
@tags(["Auth"])
@http(method: "POST", uri: "/api/v1/auth/password-reset/submit", code: 202)
@documentation("Completes the password-reset flow by applying the new password. The verification token must have been primed via ConfirmPasswordReset first.")
operation SubmitPasswordReset {
    input: SubmitPasswordResetInput
    errors: [
        NotFoundError
        UnprocessibleError
    ]
}

@input
structure SubmitPasswordResetInput {
    @required
    @documentation("The raw verification token from the reset email")
    token: String

    @required
    @length(min: 12)
    @documentation("The new password. Must be at least 12 characters long.")
    newPassword: Password
}

// ─────────────────────────────────────────────────────────────────────────────
// KYC (Know Your Customer)
// ─────────────────────────────────────────────────────────────────────────────

@auth([httpBearerAuth])
@tags(["KYC"])
@http(method: "POST", uri: "/api/v1/kyc", code: 200)
@documentation("Opens a new KYC verification flow for the authenticated user. Returns 102 Processing if a KYC record already exists.")
operation BeginKyc {
    output: BeginKycOutput
    errors: [
        UnauthorizedError
    ]
}

@output
structure BeginKycOutput {
    @required
    @documentation("The RFC 3339 timestamp by which the KYC must be completed")
    expiresAt: String
}

@auth([httpBearerAuth])
@readonly
@tags(["KYC"])
@http(method: "GET", uri: "/api/v1/kyc")
@documentation("Returns the KYC status for the authenticated user, including the list of pending actions.")
operation GetKycStatus {
    output: GetKycStatusOutput
    errors: [
        UnauthorizedError
    ]
}

@output
structure GetKycStatusOutput {
    @required
    @documentation("Current KYC status")
    status: KYCStatus

    @documentation("RFC 3339 timestamp when the KYC record expires")
    expiresAt: Timestamp

    @documentation("RFC 3339 deadline by which verification must be completed")
    verifyBefore: Timestamp

    @required
    @documentation("List of actions the user still needs to take")
    pending: StringList

    @required
    @documentation("Whether the profile is complete enough to submit for verification")
    isReady: Boolean
}

@auth([httpBearerAuth])
@tags(["KYC"])
@http(method: "POST", uri: "/api/v1/kyc/profile", code: 200)
@documentation("Updates the KYC profile for the authenticated user with personal details required for verification.")
operation UpdateKycProfile {
    input: UpdateKycProfileInput
    errors: [
        UnauthorizedError
    ]
}

@input
structure UpdateKycProfileInput {
    @required
    @documentation("The user's full legal name(s) as they appear on official documents")
    legalNames: String

    @required
    @documentation("The user's gender. Allowed values: male, female, undefined")
    gender: Gender

    @required
    @documentation("Date of birth in YYYY-MM-DD or RFC 3339 format")
    dob: String

    @documentation("The user's residential address (free-form JSON object)")
    address: Document
}

@auth([httpBearerAuth])
@tags(["KYC"])
@http(method: "POST", uri: "/api/v1/kyc/documents", code: 200)
@documentation("Batch-uploads KYC document fingerprints. At least one document must be provided. Each document's fingerprint is verified against the object store before being saved.")
operation UpdateKycDocuments {
    input: UpdateKycDocumentsInput
    errors: [
        UnauthorizedError
        UnprocessibleError
    ]
}

@input
structure UpdateKycDocumentsInput {
    @documentation("Front side of the government-issued ID")
    front: DocumentUploadItem

    @documentation("Back side of the government-issued ID")
    back: DocumentUploadItem

    @documentation("A selfie/portrait photo matching the ID")
    selfie: DocumentUploadItem

    @documentation("Proof-of-address document (e.g. utility bill, bank statement)")
    proofOfAddress: DocumentUploadItem

    @documentation("Optional base URI for the identity document bundle")
    baseUri: String
}

structure DocumentUploadItem {
    @documentation("The object/storage key name returned by the presigned-upload endpoint")
    objectName: String

    @documentation("The public or internal URL of the uploaded document")
    url: String

    @required
    @length(min: 64, max: 64)
    @documentation("SHA-256 hex fingerprint of the uploaded file, used to verify integrity")
    fingerprint: String
}

@auth([])
@readonly
@tags(["KYC"])
@http(method: "GET", uri: "/api/v1/kyc/presigned-upload")
@documentation("Generates one or more presigned PUT URLs for uploading KYC documents directly to object storage. Requires at least one document type query parameter. Optionally requires authentication.")
operation GetPresignedUploadUrl {
    input: GetPresignedUploadUrlInput
    output: GetPresignedUploadUrlOutput
}

@input
structure GetPresignedUploadUrlInput {
    @required
    @httpQuery("types")
    @documentation("Comma-separated list of document types to generate upload URLs for. Allowed values: front, back, selfie, proof_of_address")
    types: String
}

@output
structure GetPresignedUploadUrlOutput {
    @documentation("Presigned upload info keyed by document type")
    front: PresignedUploadInfo

    @documentation("Presigned upload info keyed by document type")
    back: PresignedUploadInfo

    @documentation("Presigned upload info keyed by document type")
    selfie: PresignedUploadInfo

    @documentation("Presigned upload info keyed by document type")
    proofOfAddress: PresignedUploadInfo
}

structure PresignedUploadInfo {
    @required
    @documentation("The presigned URL to PUT the file to")
    url: String

    @required
    @documentation("The object name / storage key to reference when submitting documents")
    objectName: String

    @required
    @documentation("Unix timestamp (seconds) when the presigned URL expires")
    expiresAt: Long

    @required
    @documentation("List of accepted MIME types for this document type")
    allowedTypes: StringList

    @required
    @documentation("HTTP method to use when uploading (always PUT)")
    httpMethod: String
}

// ── Verifier / Admin Endpoints ─────────────────────────────────────────────

@auth([httpBearerAuth])
@readonly
@tags(["KYC", "Admin"])
@http(method: "GET", uri: "/api/v1/kyc/applications")
@documentation("Lists all KYC applications. Optionally filters by risk profile. Requires kyc_verifier, admin, or compliance role.")
operation ListKycApplications {
    input: ListKycApplicationsInput
    output: ListKycApplicationsOutput
    errors: [
        UnauthorizedError
        ForbiddenError
    ]
}

@input
structure ListKycApplicationsInput {
    @httpQuery("risk_profile")
    @documentation("Optional risk profile filter. Allowed values: low, medium, high")
    riskProfile: KYCRiskProfile
}

@output
structure ListKycApplicationsOutput {
    @required
    records: KycRecordList

    @required
    count: Integer
}

list KycRecordList {
    member: KycRecordSummary
}

structure KycRecordSummary {
    @required
    id: Ulid

    @required
    userId: Ulid

    @required
    status: KYCStatus

    expiresAt: Timestamp
    verifyBefore: Timestamp
    isReady: Boolean
    usable: Boolean
}

@auth([httpBearerAuth])
@tags(["KYC", "Admin"])
@http(method: "POST", uri: "/api/v1/kyc/verify", code: 200)
@documentation("Marks a KYC application as 'verifying' (admin review in progress). Requires kyc_verifier, admin, or compliance role.")
operation BeginKycVerification {
    input: BeginKycVerificationInput
    errors: [
        UnauthorizedError
        ForbiddenError
        NotFoundError
    ]
}

@input
structure BeginKycVerificationInput {
    @required
    @documentation("The ID of the user whose KYC is being reviewed")
    userId: Ulid
}

@auth([httpBearerAuth])
@tags(["KYC", "Admin"])
@http(method: "PATCH", uri: "/api/v1/kyc/{user}/approve", code: 200)
@documentation("Approves a KYC application, marking the user as verified. Optionally sets a risk profile. Requires kyc_verifier, admin, or compliance role.")
operation ApproveKyc {
    input: ApproveKycInput
    errors: [
        UnauthorizedError
        ForbiddenError
        NotFoundError
    ]
}

@input
structure ApproveKycInput {
    @required
    @httpLabel
    @documentation("The ID of the user being approved")
    user: Ulid

    @documentation("Risk profile to assign to the user. Defaults to 'low'. Allowed values: low, medium, high")
    riskProfile: KYCRiskProfile
}

@auth([httpBearerAuth])
@tags(["KYC", "Admin"])
@http(method: "PATCH", uri: "/api/v1/kyc/{user}/reject", code: 200)
@documentation("Rejects a KYC application with a reason. Requires kyc_verifier, admin, or compliance role.")
operation RejectKyc {
    input: RejectKycInput
    errors: [
        UnauthorizedError
        ForbiddenError
        NotFoundError
    ]
}

@input
structure RejectKycInput {
    @required
    @httpLabel
    @documentation("The ID of the user being rejected")
    user: Ulid

    @required
    @documentation("Human-readable reason for the rejection")
    reason: String
}

// ── Shared utility types ───────────────────────────────────────────────────

list StringList {
    member: String
}
