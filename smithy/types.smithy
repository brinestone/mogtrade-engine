$version: "2"
namespace mogtrade.core.types

use smithy.api#readonly


structure AvailabilityOutput {
    @required
    @documentation("Whether the requested resource identifier is available or not")
    available: Boolean
}

@error("client")
@httpError(409)
structure ConflictError{
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


@pattern("^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+$")
string EmailAddress

@sensitive
@length(min: 6, max: 100)
string Password

list ErrorMessages {
    member: String
}