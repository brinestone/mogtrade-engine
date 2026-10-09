# Password Reset and Security Best Practices Plan for MogTrade

## Overview
This document outlines best practices for implementing a secure password reset flow and additional security measures for the MogTrade trading platform. Given the sensitive nature of a trading platform dealing with financial transactions, security requirements are paramount.

## 1. Secure Password Reset Flow

### 1.1 Token-Based Password Reset
- **Time-limited reset tokens**: Generate cryptographically secure, random tokens (minimum 32 bytes/base64) that expire after 15-30 minutes
- **One-time use only**: Each token should be single-use and invalidated after use
- **Token storage**: Store hashed tokens in the database (using bcrypt or argon2), never in plain text
- **Token endpoint**: `/auth/password-reset/request` - request a reset link via email
- **Token validation endpoint**: `/auth/password-reset/confirm` - validate token and allow new password setup

### 1.2 Email Verification
- Send password reset link to user's verified email address on file
- Include a unique, unpredictable token in the URL
- URL format: `https://api.mogtrade.com/auth/password-reset/confirm?token=<token>`
- Never include user identifiers in the URL parameter (use only the token)
- Log all password reset requests for audit purposes

### 1.3 Password Requirements
- Minimum 12 characters recommended for trading platform
- Require mixing of character types (uppercase, lowercase, numbers, symbols)
- Prohibit commonly breached passwords (use HaveIBeenPwned API or similar)
- Password history: prevent reuse of last 5 passwords
- Rate limit password changes (maximum 3 changes per 24 hours per user)

## 2. Multi-Factor Authentication (MFA)

### 2.1 Recommended MFA Implementation
- **TOTP (Time-based One-Time Password)**: Google Authenticator, Authy, Microsoft Authenticator
- **Backup codes**: Provide 8-10 single-use backup codes during MFA enrollment
- **Recovery codes**: Separate from backup codes for account recovery

### 2.2 MFA Integration Points
- **During password reset**: Require MFA verification before allowing password change if MFA is enabled
- **Login flow**: MFA required after password reset before accessing account
- **High-risk actions**: MFA required for password changes, API key changes, and withdrawal requests

### 2.3 MFA Device Management
- Allow users to manage trusted devices
- Provide option to revoke all trusted devices
- Session invalidation when MFA is added/removed

## 3. Rate Limiting and Brute Force Protection

### 3.1 Login Rate Limiting
- Maximum 5 failed login attempts per 15 minutes per IP address
- Exponential backoff: 15 min → 30 min → 1 hour → 4 hours
- Captcha after 3 failed attempts
- Progressive delays between attempts

### 3.2 Password Reset Rate Limiting
- Maximum 3 password reset requests per hour per email address
- Maximum 1 password reset per 15 minutes per user account
- Captcha after 3 failed reset requests

### 3.3 Account Lockout Policies
- **Soft lockout**: After 5 failed attempts, require email verification for next login
- **Hard lockout**: After 10 failed attempts, lock account for 24 hours and require support ticket
- Notify user of lockout via email
- Provide clear status of remaining attempts

## 4. Session Management

### 4.1 Session Security
- Regenerate session ID after password change
- Invalidate all existing sessions when password is reset
- Invalidate all sessions when MFA is enabled/disabled
- Maximum session duration: 24 hours, with re-authentication required

### 4.2 Session Invalidator
```go
// After password reset
func (c *User) InvalidateAllSessions(userId string) error {
    // Revoke all active tokens/sessions for the user
    // Store in DB: user_id, token_hash, created_at, invalidated_at
}
```

### 4.3 JWT Token Handling
- Short-lived access tokens (15-30 minutes)
- Refresh tokens with rotation on use
- Revoke refresh tokens on password change
- Store refresh token hashes in database

## 5. Account Security Monitoring

### 5.1 Anomaly Detection
- Flag logins from new devices/locations
- Monitor for unusual trading activity after password reset
- Alert users of suspicious account activity
- Email notification for: new device login, password change, MFA enrollment

### 5.2 Security Events to Track
- Failed login attempts
- Password reset requests
- MFA enrollment/disenrollment
- Password changes
- Session terminations
- API key creation/rotation

## 6. Compliance Considerations

### 6.1 GDPR Requirements
- User rights: ability to request data deletion
- Record-keeping: log all password reset requests with timestamps
- Data minimization: only request necessary information for password reset
- Retention policy: reset tokens deleted after 24 hours

### 6.2 PCI-DSS Considerations (if applicable)
- Never store full payment card data in user profiles
- Secure handling of authentication data
- Encryption of sensitive data at rest and in transit
- Regular security assessments

### 6.3 Financial Regulations
- Enhanced verification for password changes on trading accounts
- Additional confirmation for changes to withdrawal addresses
- Audit trails for all account security changes

## 7. Implementation Roadmap

### Phase 1: Immediate (1-2 weeks)
- [ ] Implement password reset request endpoint
- [ ] Generate and store hashed reset tokens
- [ ] Send email with reset link
- [ ] Implement token validation and new password setup
- [ ] Add password complexity validation
- [ ] Add rate limiting on password reset requests

### Phase 2: Enhanced Security (2-4 weeks)
- [ ] Implement TOTP MFA support
- [ ] Add backup code generation
- [ ] Implement account lockout policies
- [ ] Add session invalidation on password change
- [ ] Implement rate limiting on login attempts

### Phase 3: Advanced Security (1-2 months)
- [ ] Anomaly detection and alerting
- [ ] Comprehensive audit logging
- [ ] GDPR data export/deletion tools
- [ ] Integration with HaveIBeenPwned for password checking
- [ ] API endpoint for security status

## 8. Technical Implementation Notes

### 8.1 Database Schema Changes
```sql
-- Add to users table or separate table
ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_token VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_token_expires TIMESTAMP;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enabled BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_secret VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until TIMESTAMP;
```

### 8.2 Token Generation (Go example)
```go
import "golang.org/x/crypto/bcrypt"

// Generate reset token
func generateResetToken() (string, error) {
    tokenBytes := make([]byte, 32)
    if _, err := rand.Read(tokenBytes); err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(tokenBytes), nil
}

// Hash token for storage
func hashToken(token string) (string, error) {
    return bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
}

// Check token validity
func checkToken(hash string, token string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(token)) == nil
}
```

### 8.3 Email Template
```
Subject: MogTrade - Password Reset Request

Hello,

We received a request to reset your password. Click the link below to create a new password:

https://app.mogtrade.com/auth/password-reset/confirm?token={{.Token}}

This link will expire in 30 minutes for your security.

If you did not request this password reset, you can safely ignore this email. Your current password will remain active.

Best regards,
The MogTrade Security Team
```

## 9. Security Checklist

- [ ] Cryptographically secure token generation
- [ ] Time-limited tokens (15-30 minutes)
- [ ] Single-use token invalidation
- [ ] Hashed token storage (bcrypt/argon2)
- [ ] Email verification for reset requests
- [ ] Password complexity requirements
- [ ] Rate limiting on reset requests (3/hour)
- [ ] Account lockout policies
- [ ] MFA support (TOTP)
- [ ] Session invalidation on password change
- [ ] Audit logging of all security events
- [ ] GDPR compliance for user data rights
- [ ] HaveIBeenPwned integration for password checking
- [ ] Email notifications for security events
- [ ] Captcha after multiple failed attempts
- [ ] IP-based rate limiting
- [ ] Device fingerprinting for anomaly detection

## 10. Testing Recommendations

- [ ] Test password reset flow end-to-end
- [ ] Test token expiration (force expiry)
- [ ] Test single-use token restriction
- [ ] Test rate limiting behavior
- [ ] Test account lockout and unlock
- [ ] Test MFA enrollment and verification
- [ ] Test session invalidation after password change
- [ ] Test with compromised passwords from breach databases
- [ ] Test email delivery and link functionality
- [ ] Penetration testing of reset flow