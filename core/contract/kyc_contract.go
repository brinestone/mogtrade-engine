package contract

// KYCStatus represents the verification status of a KYC record.
type KYCStatus string

const (
	// KYCPending indicates documents have been submitted and await review.
	KYCPending KYCStatus = "pending"
	// KYCVerifying indicates an admin review is in progress.
	KYCVerifying KYCStatus = "verifying"
	// KYCVerified indicates the KYC has been successfully verified.
	KYCVerified KYCStatus = "verified"
	// KYCRejected indicates the KYC verification was rejected.
	KYCRejected KYCStatus = "rejected"
	// KYCExpired indicates the KYC has expired and needs renewal.
	KYCExpired KYCStatus = "expired"
)

// KYCRiskProfile represents the risk level of a customer.
type KYCRiskProfile string

const (
	// KYCRiskLow indicates low risk.
	KYCRiskLow KYCRiskProfile = "low"
	// KYCRiskMedium indicates medium risk.
	KYCRiskMedium KYCRiskProfile = "medium"
	// KYCRiskHigh indicates high risk.
	KYCRiskHigh KYCRiskProfile = "high"
)

// IdentityDocument stores hashes and a URI for ID document assets.
// JSON keys match the kyc_records.identity_doc column shape.
type IdentityDocument struct {
	FrontHash  string `json:"front_hash"`
	BackHash   string `json:"back_hash"`
	SelfieHash string `json:"selfie_hash"`
	URI        string `json:"uri"`
}

// ProofOfAddressDocument stores a hash and URI for proof-of-address assets.
// JSON keys match the kyc_records.proof_of_address column shape.
type ProofOfAddressDocument struct {
	DocHash string `json:"doc_hash"`
	URI     string `json:"uri"`
}

// KYCRecord contains the KYC verification data for a user.
type KYCRecord struct {
	ID              string
	UserID          string
	Status          KYCStatus
	RiskProfile     KYCRiskProfile
	VerifiedAt      *string // ISO 8601 or nil
	ExpiresAt       *string // ISO 8601 or nil
	Usable          bool
	IdentityDoc     IdentityDocument
	ProofOfAddress  ProofOfAddressDocument
	CreatedAt       string
	UpdatedAt       string
	VerifiedBy      string // admin user ID who performed verification
	RejectionReason string // if status == rejected
}
