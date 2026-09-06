$version: "2"
namespace mogtrade.auth

use smithy.api#readonly
use aws.protocols#restJson1

@title("MogTrade Authentication")
@restJson1
service Auth {
    version: "2026-02-09"
    operations: [CredentialSignIn]
}

@http(method: "POST", uri: "/api/v1/login/credential")
@documentation("This endpoint allows legitimate users to obtain a bearer JWT token and a corresponding refresh token")
operation CredentialSignIn {
    input: CredentialSignInInput
    output: SignInOutput
    errors: [
        ValidationError,
        UnprocessibleError,
        UnauthorizedError,
        InternalServerError,
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
}
@output
structure SignInOutput {
    @required
    @documentation("The access token (JWT) granted to the user")
    accessToken: String
    @required
    @documentation("The refresh token for the client to obtain a new access token on expiration")
    refreshToken: String
}

@error("server")
@httpError(500)
structure InternalServerError {
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

@error("client")
@httpError(422)
structure UnprocessibleError{
    @required
    @documentation("A list of validation messages")
    error: ErrorMessages
}

@error("client")
@httpError(401)
structure UnauthorizedError{
    @required
    error: String
}

list ErrorMessages {
    member: String
}

@pattern("^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+$")
string EmailAddress

@length(min: 6, max: 100)
string Password