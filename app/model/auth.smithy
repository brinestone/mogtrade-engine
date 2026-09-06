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
    email: EmailAddress
    @required
    password: Password
}
@output
structure SignInOutput {
    @required
    accessToken: String
    @required
    refreshToken: String
}

@error("server")
@httpError(500)
structure InternalServerError {
    @required
    error: String
}

@error("client")
@httpError(400)
structure ValidationError {
    @required
    error: ErrorMessages
}

@error("client")
@httpError(422)
structure UnprocessibleError{
    @required
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