# Security Documentation

## Overview

This document covers security considerations, authentication mechanisms, and encryption key management for the Investment Portfolio Manager application.

---

## Table of Contents

1. [Authentication & API Keys](#authentication--api-keys)
2. [IBKR Token Encryption](#ibkr-token-encryption)
3. [Environment Variables](#environment-variables)
4. [Key Management Best Practices](#key-management-best-practices)
5. [Database Security](#database-security)
6. [Logging & Monitoring](#logging--monitoring)

---

## Authentication & API Keys

### INTERNAL_API_KEY

**Purpose**: Authenticates scheduled background tasks and internal service calls.

**Location**: `backend/.env`

**Format**: String (any secure random value)

**Usage**:
```bash
INTERNAL_API_KEY=your-secure-random-key-here
```

**Generation**:
```bash
# Generate a secure random key
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

**Security Notes**:
- Keep this key secret - never commit to version control
- Change periodically (recommended: annually)
- Different key per environment (dev/staging/prod)

---

## IBKR Token Encryption

### Overview

Interactive Brokers (IBKR) Flex Tokens are encrypted at rest using **Fernet symmetric encryption** (cryptography.io). This ensures that even if the database is compromised, IBKR tokens remain protected.

### IBKR_ENCRYPTION_KEY

**Purpose**: Encrypts/decrypts IBKR Flex Tokens stored in the database.

**Format**: 44-character base64-encoded string (Fernet key)

**Example**: `cDZoaLKWN5vjqOY1p2fwE8X9mR7tU3kA4nB6sH0gC2Q=`

### How It Works

The application uses a **three-tier approach** to obtain the encryption key:

```
Priority 1: IBKR_ENCRYPTION_KEY environment variable
            ↓ (if not set)
Priority 2: /data/.ibkr_encryption_key file
            ↓ (if not exists)
Priority 3: Auto-generate new key → save to file
```

### Behavior by Scenario

#### Scenario A: No Key Provided (First Run)

**What Happens:**
1. Application detects no `IBKR_ENCRYPTION_KEY` in environment
2. Checks for `/data/.ibkr_encryption_key` file (not found)
3. Generates new Fernet key automatically
4. Saves key to `/data/.ibkr_encryption_key` with `0600` permissions
5. Logs the generated key with **WARNING** level

**Log Output:**
```
[WARNING] [SECURITY] IBKR encryption key auto-generated
  key: <44-char-key>
  saved_to: /data/.ibkr_encryption_key
  action_required: Save this key for database migrations
  best_practice: Set IBKR_ENCRYPTION_KEY in .env file
```

**Action Required:**
- ✅ **Copy the key from logs** and save it securely (password manager, encrypted notes)
- ✅ **Optional but recommended**: Add to `.env` file for explicit control

#### Scenario B: Key in Environment Variable (Recommended)

**What Happens:**
1. Application loads `IBKR_ENCRYPTION_KEY` from `.env`
2. Uses this key for all encryption/decryption
3. Logs confirmation with **INFO** level
4. Ignores any file-based key

**Log Output:**
```
[INFO] [SECURITY] Using IBKR encryption key from environment variable
  source: IBKR_ENCRYPTION_KEY
  method: environment_variable
```

**Advantages:**
- ✅ Explicit control over encryption key
- ✅ Easy database migrations (just copy .env)
- ✅ Works across reinstalls
- ✅ Consistent with GitOps workflows

#### Scenario C: Key in Persistent File (Existing Installation)

**What Happens:**
1. Application finds `/data/.ibkr_encryption_key` file
2. Loads key from file
3. Uses this key for all encryption/decryption
4. Logs confirmation with **INFO** level

**Log Output:**
```
[INFO] [SECURITY] Using IBKR encryption key from persistent storage
  source: /data/.ibkr_encryption_key
  method: persistent_file
```

**Advantages:**
- ✅ Persists across container restarts
- ✅ Zero configuration needed
- ✅ Automatic management

**Limitations:**
- ❌ Key tied to volume (must backup both database + key file)
- ❌ Database migration requires copying key file manually

### Setting Your Own Key (Recommended)

#### Step 1: Generate a Key

**Using Docker:**
```bash
docker-compose exec backend python -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())"
```

**Using Python Locally:**
```bash
python3 -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())"
```

**Output Example:**
```
cDZoaLKWN5vjqOY1p2fwE8X9mR7tU3kA4nB6sH0gC2Q=
```

#### Step 2: Add to Environment

Edit `backend/.env`:
```bash
IBKR_ENCRYPTION_KEY=<your-generated-key>
```

#### Step 3: Restart Backend

```bash
docker-compose restart backend
```

Check logs to confirm:
```bash
docker-compose logs backend | grep "IBKR encryption key"
```

---

## Environment Variables

### Required Variables

None - all security variables have sensible defaults or auto-generation.

### Optional Variables

| Variable | Purpose | Default | Location |
|----------|---------|---------|----------|
| `INTERNAL_API_KEY` | Authenticate internal services | Auto-generated | backend/.env |
| `IBKR_ENCRYPTION_KEY` | Encrypt IBKR tokens | Auto-generated | backend/.env |
| `IBKR_DEBUG_SAVE_XML` | Save raw IBKR XML responses | `false` | backend/.env |

### Development vs Production

**Development (.env.example):**
```bash
# Use example keys for testing
INTERNAL_API_KEY=dev-key-12345
IBKR_ENCRYPTION_KEY=<leave-empty-for-auto-generation>
IBKR_DEBUG_SAVE_XML=false
```

**Production (actual .env):**
```bash
# Generate unique keys for production
INTERNAL_API_KEY=<unique-production-key>
IBKR_ENCRYPTION_KEY=<unique-production-key>
# Never enable in production:
# IBKR_DEBUG_SAVE_XML=false
```

---

## Key Management Best Practices

### Do's ✅

1. **Use Environment Variables**
   - Set `IBKR_ENCRYPTION_KEY` explicitly in `.env`
   - Easier migrations and reinstalls

2. **Backup Your Keys**
   - Store encryption key in password manager
   - Keep separate from database backups
   - Document key rotation procedures

3. **Secure Storage**
   - Never commit `.env` to version control (already in `.gitignore`)
   - Restrict file permissions: `chmod 600 .env`
   - Use encrypted notes/vaults for key storage

4. **Monitor Logs**
   - Check `LogCategory.SECURITY` logs regularly
   - Verify key source on application startup
   - Alert on encryption failures

5. **Test Restores**
   - Periodically test database restore + key
   - Verify IBKR token decryption works
   - Document restore procedures

### Don'ts ❌

1. **Never Commit Keys**
   - Don't commit `.env` to git
   - Don't hardcode keys in code
   - Don't share keys in chat/email

2. **Don't Lose Keys**
   - Losing key = need to re-enter IBKR token
   - Database without key = encrypted data inaccessible
   - Plan for key recovery scenarios

3. **Don't Use Default Keys**
   - Don't use published example keys
   - Don't reuse keys across environments
   - Don't use weak/predictable keys

4. **Don't Enable Debug in Production**
   - `IBKR_DEBUG_SAVE_XML=true` saves raw transaction data
   - Contains sensitive financial information
   - Only enable for troubleshooting

---

## Database Migration & Backup

### Scenario 1: Migrating to New Server

**With Environment Variable (Easy):**
```bash
# Old server
1. Stop application: docker-compose down
2. Backup database: cp data/db/portfolio_manager.db backup/
3. Copy .env file: cp backend/.env backup/

# New server
4. Restore database: cp backup/portfolio_manager.db data/db/
5. Restore .env: cp backup/.env backend/
6. Start application: docker-compose up -d
7. ✅ IBKR tokens decrypt successfully
```

**With Auto-Generated Key (Requires Extra Step):**
```bash
# Old server
1. Stop application: docker-compose down
2. Backup database: cp data/db/portfolio_manager.db backup/
3. **Backup key file**: cp data/.ibkr_encryption_key backup/

# New server
4. Restore database: cp backup/portfolio_manager.db data/db/
5. **Restore key file**: cp backup/.ibkr_encryption_key data/
6. Start application: docker-compose up -d
7. ✅ IBKR tokens decrypt successfully
```

### Scenario 2: Lost Encryption Key

**Impact:**
- ❌ Cannot decrypt existing IBKR Flex Token
- ✅ Historical transaction imports are NOT affected
- ✅ All other data remains accessible

**Solution:**
```bash
1. Generate new encryption key (see above)
2. Set IBKR_ENCRYPTION_KEY in .env
3. Restart application
4. Navigate to Settings > IBKR Configuration
5. Re-enter IBKR Flex Token (retrieve from IBKR portal if needed)
6. Click Save
7. ✅ New token encrypted with new key
```

### Scenario 3: Key Rotation (Security Event)

**When to Rotate:**
- Suspected key compromise
- Employee departure
- Security audit requirement
- Regular rotation policy (annually)

**Procedure:**
```bash
1. Generate new encryption key
2. Set new IBKR_ENCRYPTION_KEY in .env
3. Restart application
4. Re-enter IBKR Flex Token in Settings
5. ✅ Old tokens inaccessible, new token uses new key
```

---

## Database Security

### Encryption at Rest

- **IBKR Tokens**: Encrypted using Fernet (AES-128)
- **Other Data**: Stored in plaintext (SQLite database)
- **Future**: Consider full database encryption for production

### Access Control

- **File Permissions**: Key file set to `0600` (owner read/write only)
- **Database File**: Stored in Docker volume (isolated from host)
- **Network**: Backend not exposed, only accessible via frontend proxy

### Backup Security

**Recommendations:**
1. **Encrypt backups** using system-level encryption:
   ```bash
   # Example: Encrypt backup with GPG
   tar czf - data/ | gpg -c > backup.tar.gz.gpg
   ```

2. **Store keys separately** from backups:
   - Backups → Cloud storage / external drive
   - Keys → Password manager / KMS

3. **Test restores regularly**:
   - Quarterly restore test
   - Verify decryption works
   - Document any issues

---

## Logging & Monitoring

### Security Logs

All security-related events use `LogCategory.SECURITY`:

**Key Events Logged:**
- Encryption key loaded (INFO)
- Encryption key generated (WARNING)
- Encryption failure (ERROR)
- Key file read errors (ERROR)
- Token encryption success (DEBUG)
- Token decryption success (DEBUG)

### Viewing Security Logs

**Via Log Viewer UI:**
1. Navigate to Settings > Logs
2. Filter by Category: SECURITY
3. Review recent events

**Via CLI:**
```bash
# View all security logs
docker-compose logs backend | grep "SECURITY"

# View encryption key logs
docker-compose logs backend | grep "IBKR encryption key"

# View last 50 security events
docker-compose logs --tail=50 backend | grep "SECURITY"
```

### Monitoring Recommendations

**Alerts to Set Up:**
- ⚠️ WARNING: Encryption key auto-generated (action required)
- 🔴 ERROR: Encryption key unavailable (critical)
- 🔴 ERROR: Token decryption failed (critical)

---

## Debug Mode Security

### IBKR_DEBUG_SAVE_XML

**Purpose**: Saves raw XML responses from IBKR Flex API to disk for troubleshooting.

**Security Risk**: 🔴 HIGH
- Contains complete transaction history
- Includes account numbers, symbols, amounts, dates
- Stored in plaintext on disk

**Usage**:
```bash
# Enable only when troubleshooting
IBKR_DEBUG_SAVE_XML=true

# Disable after debugging
IBKR_DEBUG_SAVE_XML=false
```

**Files Saved To**:
```
backend/data/ibkr_debug/
  └── ibkr_statement_<reference_code>_<timestamp>.xml
```

**Cleanup**:
```bash
# Delete all debug files
rm -rf backend/data/ibkr_debug/*.xml

# Or via Docker
docker-compose exec backend rm -rf /app/data/ibkr_debug/*.xml
```

**Best Practices**:
- ❌ Never enable in production
- ❌ Never commit debug files to git (already in .gitignore)
- ✅ Enable only temporarily
- ✅ Delete files after troubleshooting
- ✅ Review files before sharing with support

---

## Troubleshooting

### "Encryption Key Invalid" Error

**Symptoms:**
- Cannot decrypt IBKR token
- Error in IBKR Configuration page

**Diagnosis:**
```bash
# Check key format (should be 44 chars ending in =)
grep IBKR_ENCRYPTION_KEY backend/.env

# Check logs
docker-compose logs backend | grep "SECURITY"
```

**Solutions:**
1. Verify key format is correct (44 chars, base64, ends with `=`)
2. Regenerate key if corrupted (see Key Rotation)
3. Check file permissions: `ls -l data/.ibkr_encryption_key`

### "Encryption Key Not Available" Error

**Symptoms:**
- Application logs show ERROR
- IBKR features don't work

**Solutions:**
1. Check if key file exists: `ls -l data/.ibkr_encryption_key`
2. Check environment variable: `docker-compose exec backend env | grep IBKR`
3. Review startup logs: `docker-compose logs backend | grep "encryption key"`

### Key File Permission Denied

**Symptoms:**
- Cannot read/write key file
- Logs show permission errors

**Solutions:**
```bash
# Fix permissions
chmod 600 data/.ibkr_encryption_key
chown <your-user>:<your-group> data/.ibkr_encryption_key
```

---

## Summary

**Encryption Key Management:**
- ✅ Auto-generates if not provided (zero config)
- ✅ Persists across restarts (stored in volume)
- ✅ Can be explicitly controlled (environment variable)
- ✅ Comprehensive logging (SECURITY category)

**Best Practices:**
1. Set `IBKR_ENCRYPTION_KEY` in `.env` for production
2. Backup encryption key separately from database
3. Never commit `.env` to version control
4. Monitor security logs regularly
5. Test restore procedures quarterly

**For More Information:**
- IBKR Setup Guide: [docs/IBKR_SETUP.md](IBKR_SETUP.md)
- IBKR Technical Details: [docs/IBKR_TRANSACTION_LIFECYCLE.md](IBKR_TRANSACTION_LIFECYCLE.md)
- Development Setup: [docs/DEVELOPMENT.md](DEVELOPMENT.md)
