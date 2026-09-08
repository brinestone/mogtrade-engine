$version: "2"
namespace mogtrade.auth

use smithy.api#readonly
use aws.protocols#restJson1
use mogtrade.core.types#AvailabilityOutput
use mogtrade.core.types#ValidationError
use mogtrade.core.types#EmailAddress
use mogtrade.core.types#ConflictError
use mogtrade.core.types#InternalServerError
use mogtrade.core.types#UnprocessibleError
use mogtrade.core.types#UnauthorizedError
use mogtrade.core.types#Password

@restJson1
@httpBearerAuth
@title("MogTrade Authentication")
service Auth {
    version: "2026-02-09"
    errors: [
        ValidationError,
        InternalServerError
    ]
    operations: [CredentialSignIn, CredentialSignUp, RotateAccessToken, CheckEmailAvailable]
}

@auth([])
@readonly
@tags(["Auth"])
@http(method:"GET", uri:"/api/v1/auth/email-available")
@documentation("Check whether an email available for a user")
operation CheckEmailAvailable{
    input: CheckEmailAvailableInput
    output: AvailabilityOutput
}

@readonly
@tags(["Auth"])
@auth([httpBearerAuth])
@http(method: "GET", uri: "/api/v1/auth/refresh")
@documentation("Rotate access token")
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
    refreshToken: String
    @required
    @httpHeader("X-Device-Id")
    deviceId: String
}

@auth([])
@tags(["Auth"])
@http(method: "POST", uri: "/api/v1/auth/register/credential", code: 201)
@documentation("Create user account using credentials")
operation CredentialSignUp {
    input: CredentialSignUpInput
    errors: [
        ConflictError,
    ]
}

@input
structure CredentialSignUpInput {
    @required
    @documentation("The names of the user")
    names: String
    @required
    @documentation("The email address of the user. MUST be unique")
    email: EmailAddress
    @required
    @documentation("User password")
    password: Password
    @required
    @documentation("Password confirmation. MUST have value equal to the password value")
    confirmPassword: Password
}

@auth([])
@tags(["Auth"])
@http(method: "POST", uri: "/api/v1/auth/login/credential")
@documentation("This endpoint allows legitimate users to obtain a bearer JWT token and a corresponding refresh token")
operation CredentialSignIn {
    input: CredentialSignInInput
    output: SignInOutput
    errors: [
        UnprocessibleError,
        UnauthorizedError,
    ]
}
@input
structure CredentialSignInInput {
    @required
    @documentation("The user's identifying email address")
    email: EmailAddress
    @required
    @documentation("The user's password")
    password: String
    @httpHeader("X-Device-Id")
    @required
    @documentation("The client device's ID")
    deviceId: String
}

structure SignInOutput {
    @required
    @documentation("The access token (JWT) granted to the user")
    accessToken: String
    @required
    @documentation("The refresh token for the client to obtain a new access token on expiration")
    refreshToken: String
}

@input
structure CheckEmailAvailableInput {
    @required
    @documentation("The email address query")
    @httpQuery("email")
    email: String
}
