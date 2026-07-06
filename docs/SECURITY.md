# 🔒 Security & Compliance Guide

> **Enterprise-grade security features** for GDPR, HIPAA, SOX, and PCI-DSS compliance

## Overview

HaloLogger provides built-in security features that help you meet compliance requirements without external dependencies:

- **PII Masking** - Automatic sensitive data protection
- **Field Encryption** - AES-256 field-level encryption  
- **Audit Logging** - Complete operation tracking
- **Data Retention** - Configurable log rotation and cleanup
- **Access Control** - Secure configuration management

## PII Data Protection

### Automatic PII Detection
```go
import "github.com/go-gen-ecosystem/halolog/masking"

// Create masker with built-in patterns
masker := masking.NewPIIMasker()

// Add common PII patterns
masker.AddPattern("email", `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, "[EMAIL]")
masker.AddPattern("ssn", `\b\d{3}-\d{2}-\d{4}\b`, "[SSN]")
masker.AddPattern("phone", `\b\d{3}-\d{3}-\d{4}\b`, "[PHONE]")
masker.AddPattern("credit_card", `\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`, "[CARD]")

// Configure logger
cfg := &config.ImmutableConfig{
    Level: types.InfoLevel,
    Masking: types.MaskingConfig{
        Enabled: true,
        Masker: masker,
    },
}

logger := halolog.GetLoggerWithConfig("secure-app", cfg)

// PII is automatically masked
logger.Info("User john.doe@example.com with SSN 123-45-6789 logged in")
// Output: User [EMAIL] with [SSN] logged in
```

### Custom PII Patterns
```go
// Healthcare (HIPAA)
masker.AddPattern("patient_id", `\bP[A-Z]{2}\d{6}\b`, "[PATIENT_ID]")
masker.AddPattern("mrn", `\bMRN\d{8}\b`, "[MRN]")
masker.AddPattern("dob", `\b\d{2}/\d{2}/\d{4}\b`, "[DOB]")

// Financial (PCI-DSS)
masker.AddPattern("account", `\bACC\d{10}\b`, "[ACCOUNT]")
masker.AddPattern("routing", `\b\d{9}\b`, "[ROUTING]")
masker.AddPattern("swift", `\b[A-Z]{6}[A-Z0-9]{2}[A-Z0-9]{3}\b`, "[SWIFT]")

// Government
masker.AddPattern("passport", `\b[A-Z]{2}\d{6}\b`, "[PASSPORT]")
masker.AddPattern("driver_license", `\bDL\d{8}\b`, "[DL]")
```

### Context-Aware Masking
```go
// Different masking for different contexts
apiMasker := masking.NewPIIMasker()
apiMasker.AddPattern("api_key", `\b(sk|pk)_test_[a-zA-Z0-9]{24}\b`, "[API_KEY]")

dbMasker := masking.NewPIIMasker()
dbMasker.AddPattern("connection_string", `Server=[^;]+;Database=[^;]+`, "[CONNECTION_STRING]")

// Use different maskers for different loggers
apiLogger := halolog.GetLoggerWithConfig("api", &config.ImmutableConfig{
    Masking: types.MaskingConfig{Enabled: true, Masker: apiMasker},
})

dbLogger := halolog.GetLoggerWithConfig("database", &config.ImmutableConfig{
    Masking: types.MaskingConfig{Enabled: true, Masker: dbMasker},
})
```

## Field-Level Encryption

### Basic Encryption
```go
import "github.com/go-gen-ecosystem/halolog/masking"

// Initialize global encryptor (do this once at startup)
masking.InitGlobalEncryptor("your-32-byte-secret-key-here")

// Use encrypted fields
logger.WithEncryptedField("password", "user_secret_password").
    WithEncryptedField("credit_card", "4111111111111111").
    WithEncryptedField("ssn", "123-45-6789").
    Info("User authentication completed")
```

### Advanced Encryption Configuration
```go
// Custom encryption configuration
encryptor := masking.NewEncryptor(masking.EncryptionConfig{
    Algorithm: masking.AES256_GCM,
    Key:       []byte("your-32-byte-secret-key-here"),
    KeyRotation: 24 * time.Hour, // Rotate keys every 24 hours
})

// Use with specific logger
logger := halolog.GetLoggerWithConfig("secure", &config.ImmutableConfig{
    Encryption: types.EncryptionConfig{
        Enabled: true,
        Encryptor: encryptor,
    },
})
```

### Encryption with Key Rotation
```go
// Implement key rotation strategy
keyManager := masking.NewKeyManager()
keyManager.AddKey("key-v1", []byte("old-secret-key"))
keyManager.AddKey("key-v2", []byte("new-secret-key"))
keyManager.SetCurrentKey("key-v2")

encryptor := masking.NewEncryptorWithKeyManager(keyManager)

// Logs will be encrypted with current key
// Old logs can still be decrypted with key manager
```

## Audit Logging

### Compliance-Audit Trail
```go
// Enable audit logging
cfg := &config.ImmutableConfig{
    Level: types.InfoLevel,
    Audit: types.AuditConfig{
        Enabled: true,
        Trail: true,
        Immutable: true, // Prevent log modification
    },
}

auditLogger := halolog.GetLoggerWithConfig("audit", cfg)

// Audit logs are automatically tagged
auditLogger.WithField("user_id", 12345).
    WithField("action", "data_access").
    WithField("resource", "patient_records").
    WithField("ip", "192.168.1.100").
    Audit("User accessed sensitive data")
```

### Tamper-Proof Logging
```go
// Create tamper-proof logger
tamperProof := halolog.GetLoggerWithConfig("compliance", &config.ImmutableConfig{
    Audit: types.AuditConfig{
        Enabled: true,
        Immutable: true,
        HashChain: true, // Link logs with cryptographic hashes
        Sign: true,      // Sign each log entry
    },
})

// Each log entry is cryptographically linked
// Any modification breaks the chain
tamperProof.Audit("Financial transaction processed")
```

## Data Retention & Cleanup

### Automatic Log Rotation
```go
// Configure rotation for compliance
rotationConfig := &file.RotationConfig{
    MaxSize:    100 * 1024 * 1024, // 100MB
    MaxAge:     90 * 24 * time.Hour, // 90 days (SOX requirement)
    MaxBackups: 365,                // 1 year of daily backups
    Compress:   true,
    LocalTime:  true,
}

// For HIPAA: 6 years retention
hipaaRotation := &file.RotationConfig{
    MaxSize:    50 * 1024 * 1024, // 50MB
    MaxAge:     6 * 365 * 24 * time.Hour, // 6 years
    MaxBackups: 2190, // 6 years of daily backups
    Compress:   true,
    Encrypt:    true, // Encrypt archived logs
}
```

### Secure Log Deletion
```go
// Configure secure deletion
deletionConfig := &file.DeletionConfig{
    SecureDelete: true,    // Overwrite before deletion
    OverwritePasses: 3,    // DoD 5220.22-M standard
    Delay: 24 * time.Hour, // Wait before deletion
}

rotationConfig.Deletion = deletionConfig
```

## Access Control & Configuration Security

### Secure Configuration Loading
```go
// Load from encrypted config file
configData, err := encryption.DecryptFile("config.encrypted", key)
if err != nil {
    log.Fatal("Failed to load secure config")
}

cfg, err := config.LoadFromYAML(configData)
if err != nil {
    log.Fatal("Invalid configuration")
}
```

### Environment-Based Security
```go
// Different security levels per environment
func getSecurityConfig(env string) *config.ImmutableConfig {
    switch env {
    case "production":
        return &config.ImmutableConfig{
            Level: types.InfoLevel,
            Masking: types.MaskingConfig{Enabled: true},
            Encryption: types.EncryptionConfig{Enabled: true},
            Audit: types.AuditConfig{Enabled: true, Immutable: true},
        }
    case "staging":
        return &config.ImmutableConfig{
            Level: types.DebugLevel,
            Masking: types.MaskingConfig{Enabled: true},
        }
    case "development":
        return &config.ImmutableConfig{
            Level: types.DebugLevel,
            // No masking/encryption in dev
        }
    default:
        return &config.ImmutableConfig{Level: types.InfoLevel}
    }
}
```

## Compliance Frameworks

### GDPR Compliance
```go
// GDPR requires data minimization and right to be forgotten
gdprConfig := &config.ImmutableConfig{
    Level: types.InfoLevel,
    Masking: types.MaskingConfig{
        Enabled: true,
        Patterns: []masking.Pattern{
            {Name: "personal_data", Pattern: `\b[A-Za-z]+ [A-Za-z]+\b`, Mask: "[NAME]"},
            {Name: "email", Pattern: `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, Mask: "[EMAIL]"},
            {Name: "phone", Pattern: `\b\d{3}-\d{3}-\d{4}\b`, Mask: "[PHONE]"},
        },
    },
    Retention: types.RetentionConfig{
        MaxAge: 30 * 24 * time.Hour, // 30 days max retention
        AutoDelete: true,
    },
}
```

### HIPAA Compliance
```go
// HIPAA requires encryption and audit trails
hipaaConfig := &config.ImmutableConfig{
    Level: types.InfoLevel,
    Encryption: types.EncryptionConfig{
        Enabled: true,
        Algorithm: masking.AES256_GCM,
        KeyRotation: 90 * 24 * time.Hour, // 90 days
    },
    Audit: types.AuditConfig{
        Enabled: true,
        Immutable: true,
        HashChain: true,
        Sign: true,
    },
    Retention: types.RetentionConfig{
        MaxAge: 6 * 365 * 24 * time.Hour, // 6 years
        Encrypt: true,
        SecureDelete: true,
    },
}
```

### SOX Compliance
```go
// SOX requires tamper-proof audit trails
soxConfig := &config.ImmutableConfig{
    Level: types.InfoLevel,
    Audit: types.AuditConfig{
        Enabled: true,
        Immutable: true,
        HashChain: true,
        Sign: true,
        Notarize: true, // Third-party notarization
    },
    Retention: types.RetentionConfig{
        MaxAge: 7 * 365 * 24 * time.Hour, // 7 years
        WORM: true, // Write Once Read Many
    },
}
```

### PCI-DSS Compliance
```go
// PCI-DSS requires card data protection
pciConfig := &config.ImmutableConfig{
    Level: types.InfoLevel,
    Masking: types.MaskingConfig{
        Enabled: true,
        Patterns: []masking.Pattern{
            {Name: "card_number", Pattern: `\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`, Mask: "[CARD]"},
            {Name: "cvv", Pattern: `\b\d{3,4}\b`, Mask: "[CVV]"},
            {Name: "expiry", Pattern: `\b\d{2}/\d{2}\b`, Mask: "[EXPIRY]"},
        },
    },
    Encryption: types.EncryptionConfig{
        Enabled: true,
        Algorithm: masking.AES256_GCM,
        KeyManagement: "HSM", // Hardware Security Module
    },
    Network: types.NetworkConfig{
        TLS: true,
        MTLS: true, // Mutual TLS
    },
}
```

## Security Best Practices

### 1. Principle of Least Privilege
```go
// Only log what's necessary for business operations
logger.Info("Transaction completed", 
    "amount", amount,           // OK - business data
    "user_id", userID,          // OK - reference data
    // "password", password,    // ❌ Never log passwords
    // "ssn", ssn,              // ❌ Never log SSNs
)
```

### 2. Secure Key Management
```go
// Use environment variables or secure key stores
encryptionKey := os.Getenv("HALOLOG_ENCRYPTION_KEY")
if encryptionKey == "" {
    log.Fatal("HALOLOG_ENCRYPTION_KEY not set")
}

// Or use cloud key management
key, err := awsKMS.GetKey("alias/halolog-encryption")
if err != nil {
    log.Fatal("Failed to get encryption key")
}
```

### 3. Regular Security Audits
```go
// Enable security logging
securityLogger := halolog.GetLoggerWithConfig("security", &config.ImmutableConfig{
    Level: types.InfoLevel,
    Audit: types.AuditConfig{
        Enabled: true,
        Immutable: true,
        HashChain: true,
    },
})

// Log security events
securityLogger.WithField("event_type", "config_change").
    WithField("user", currentUser).
    WithField("change", "log_level").
    Audit("Security configuration changed")
```

### 4. Network Security
```go
// Secure log transmission
cfg := &config.ImmutableConfig{
    Network: types.NetworkConfig{
        TLS: true,
        CertFile: "/path/to/cert.pem",
        KeyFile: "/path/to/key.pem",
        CAFile: "/path/to/ca.pem",
        MTLS: true, // Mutual authentication
    },
}
```

## Incident Response

### Breach Detection
```go
// Monitor for suspicious patterns
func detectBreach(entry *types.LogEntry) bool {
    // Check for data exfiltration attempts
    if entry.Level >= types.ErrorLevel && 
       strings.Contains(entry.Message, "unauthorized") {
        return true
    }
    
    // Check for unusual data access patterns
    if entry.Fields["access_count"] != nil {
        count := entry.Fields["access_count"].(int)
        if count > 1000 { // Unusual high access
            return true
        }
    }
    
    return false
}

// Alert on breach detection
if detectBreach(entry) {
    alertSender.Send(&types.AlertPayload{
        Level: "critical",
        Message: "Potential security breach detected",
        Details: entry,
    })
}
```

### Forensic Analysis
```go
// Enable detailed forensic logging
forensicLogger := halolog.GetLoggerWithConfig("forensic", &config.ImmutableConfig{
    Level: types.DebugLevel,
    Audit: types.AuditConfig{
        Enabled: true,
        Immutable: true,
        HashChain: true,
        Sign: true,
        Timestamp: types.TimestampConfig{
            Format: time.RFC3339Nano,
            Monotonic: true,
        },
    },
})

// Capture detailed system state
forensicLogger.WithField("process_id", os.Getpid()).
    WithField("user_id", currentUser).
    WithField("session_id", sessionID).
    WithField("ip_address", clientIP).
    WithField("user_agent", userAgent).
    Audit("User session started")
```

## Compliance Checklist

### Pre-Production Checklist
- [ ] PII masking enabled for production data
- [ ] Encryption configured for sensitive fields
- [ ] Audit logging enabled for compliance requirements
- [ ] Log retention policies configured
- [ ] Secure key management implemented
- [ ] Access controls in place
- [ ] Incident response procedures documented
- [ ] Regular security audits scheduled

### Post-Deployment Monitoring
- [ ] Monitor for unmasked PII in logs
- [ ] Verify encryption key rotation
- [ ] Check audit log integrity
- [ ] Review access patterns
- [ ] Test incident response procedures
- [ ] Update security policies as needed

## Resources

- **[Security Best Practices](https://github.com/go-gen-ecosystem/halolog/wiki/Security-Best-Practices)** - Detailed security guidelines
- **[Compliance Templates](https://github.com/go-gen-ecosystem/halolog/wiki/Compliance-Templates)** - Pre-built configurations
- **[Incident Response](https://github.com/go-gen-ecosystem/halolog/wiki/Incident-Response)** - Security incident procedures
- **[Audit Requirements](https://github.com/go-gen-ecosystem/halolog/wiki/Audit-Requirements)** - Audit trail specifications

---

**🔒 HaloLogger: Security-first logging for enterprise applications**

*Need help with compliance? Contact our security team at security@halolog.dev*