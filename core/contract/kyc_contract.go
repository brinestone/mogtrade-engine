package contract

// KYCStatus represents the verification status of a KYC record.
type KYCStatus string

const (
	// KYCPending indicates the KYC verification is in progress.
	KYCPending KYCStatus = "pending"
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

// KYCRecord contains the KYC verification data for a user.
type KYCRecord struct {
	ID              string
	UserID          string
	Status          KYCStatus
	RiskProfile     KYCRiskProfile
	VerifiedAt      *string // ISO 8601 or nil
	ExpiresAt       *string // ISO 8601 or nil
	IdentityDoc     DocumentRef
	ProofOfAddress  DocumentRef
	CreatedAt       string
	UpdatedAt       string
	VerifiedBy      string // admin user ID who performed verification
	RejectionReason string // if status != verified
}

// DocumentRef represents a reference to a document stored in object storage.
type DocumentRef struct {
	URI        string // URI/URL where the document is stored
	SHA256     string // SHA256 hash for integrity verification
	UploadedAt string // when the document was uploaded (ISO 8601)
}
