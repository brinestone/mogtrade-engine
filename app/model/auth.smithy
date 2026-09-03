$version: "2"
namespace mogtrade.auth

use smithy.api#readonly
use aws.protocols#restJson1

@title("MogTrade Authentication")
@restJson1
@httpBearerAuth
service Auth {
    version: "2026-02-09"
    operations: [CredentialSignIn]
}

@http(method: "POST", uri: "/login/credential")
operation CredentialSignIn {
    input: CredentialSignInInput
    output: SignInOutput
    errors: [
        ErrorResponse
    ]
}
@input
structure CredentialSignInInput {
    @required
    username: String
    @required
    password: String
}
@output
structure SignInOutput {
    @required
    accessToken: String
    @required
    refreshToken: String
}

@error("client")
structure ErrorResponse {
    @required
    error: String
}