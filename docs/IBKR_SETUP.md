# IBKR Flex Integration Setup Guide

## Overview
This guide explains how to set up the Interactive Brokers (IBKR) Flex Web Service integration to automatically import transactions.

## Step 1: Create a Flex Query in IBKR

1. Log in to your IBKR account at https://www.interactivebrokers.com/portal
2. Navigate to **Reports** > **Flex Queries**
3. Click **Create** to create a new Flex Query
4. Configure your query with the following settings:

### Required Sections:

#### Trades Section
Include all trade executions. Select these fields in your Flex Query:
- **Symbol** (Trading ticker symbol)
- **ISIN** (International Securities ID)
- **Description** (Security description)
- **Date/Time** (Trade execution date/time)
- **Quantity** (Number of shares)
- **TradePrice** (Price per share)
- **Net Cash** (Total amount including commissions)
- **Currency** (Trading currency)
- **IB Commission** (Commission fees)
- **Transaction ID** (For tracking - recommended)
- **IB Order ID** (For tracking - recommended)

#### Cash Transactions Section
Include cash movements for dividends and fees. Select these fields:
- **Type** (Transaction type - includes dividends, fees, etc.)
- **Symbol** (Trading ticker symbol)
- **ISIN** (International Securities ID - critical for fund matching)
- **Description** (Transaction description)
- **Date/Time** (Transaction date/time)
- **Amount** (Transaction amount)
- **Currency** (Transaction currency)
- **Transaction ID** (For tracking - recommended)
- **Report Date** (Reporting date - useful for reconciliation)
- **Ex Date** (Ex-dividend date - important for dividend tracking)
- **Code** (Transaction classification code - optional but useful)

### Period Settings:
- **Period**: **Last 365 Calendar Days** (Recommended)
  - The system automatically prevents duplicate imports
  - Ensures you don't miss any delayed transactions or corrections
  - Useful for year-to-date reconciliation
  - The 24-hour cache makes large queries efficient

### Format Settings:
- **Format**: XML (required)
- **Language**: English
- **Date Format**: yyyyMMdd (required)
- **Time Format**: HHmmss
- **Date/Time Separator**: ; (semi-colon)

### General Configuration:
- **Include Canceled Trades?**: NO (canceled trades shouldn't appear in portfolios)
- **Include Currency Rates?**: YES (useful for future currency conversion features)
- **Include Audit Trail Fields?**: YES (helpful for debugging and reconciliation)
- **Profit and Loss**: Default/Base Currency P&L (doesn't affect transaction import)
- **Display Account Alias in Place of Account ID?**: Your preference (either works)
- **Breakout by Day?**: NO (keep all transactions in a single report)

### Important Notes About Field Names:
- The field names shown in the IBKR UI (e.g., "Trade Price") will appear in camelCase in the XML output (e.g., "tradePrice")
- The application automatically handles this conversion
- **Note**: When you select "Currency", IBKR may automatically change it to "CurrencyPrimary" in the final query - this is expected and the application handles both variations

5. **Save** the query and note the **Query ID** (you'll need this later)

## Step 2: Enable Flex Web Service

1. Stay on the **Reports** > **Flex Queries** page (where you just created your query)
2. Look for the **Flex Web Service** section on that same page
3. Click **Enable** or toggle the switch to enable Flex Web Service
4. Once enabled, you'll see the option to generate tokens

## Step 3: Generate Flex Token

1. In the **Flex Web Service** section (from Step 2), click **Generate Token**
2. Configure the token:
   - **Token Validity**: Maximum 1 year (365 days) - recommend setting to 1 year
   - **IP Address Restriction** (Optional):
     - Leave blank for unrestricted access (recommended for dynamic IPs)
     - Or enter specific IP addresses if you have a static IP
     - **Note**: If using Docker or cloud hosting, ensure the server IP is allowed
3. Click **Generate**
4. **IMPORTANT**: Copy the token immediately - it's only shown once
5. Store the token securely (you'll add it to the application in the next step)

### Important Notes About Token Expiration:
- Tokens are valid for maximum 1 year from creation date
- The application will track your token expiration date and warn you 30 days before expiry
- **Set a reminder** to regenerate your token before it expires
- When regenerating, simply update the token in the application settings (no need to reconfigure the query)

## Step 4: Configure Encryption (Optional)

IBKR Flex Tokens are encrypted at rest for security. The application will **auto-generate** an encryption key on first startup if you don't provide one.

**Option A: Let It Auto-Generate (Easiest)**
- No action required
- Key is generated and saved to `/data/.ibkr_encryption_key`
- Check logs for the generated key (save it for backup purposes)

**Option B: Set Your Own Key (Recommended for Production)**

Add to `backend/.env`:
```bash
IBKR_ENCRYPTION_KEY=<your-generated-key>
```

Generate a key with:
```bash
docker-compose exec backend python -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())"
```

**📖 For Complete Security Documentation:**
See [docs/SECURITY.md](SECURITY.md) for:
- Encryption key management
- Migration and backup procedures
- Key rotation
- Security best practices

## Step 5: Configure in Application UI

1. Start your application
2. Navigate to **Settings** or **IBKR Configuration** page
3. Enter your:
   - **Flex Token** (from Step 3)
   - **Flex Query ID** (from Step 1)
   - **Token Expiration Date** (recommended: set to 1 year from today)
4. Enable **Auto-Import** if you want weekly automated imports
5. Click **Save Configuration**
6. Click **Test Connection** to verify your setup

**Note**: The application will warn you 30 days before your token expires.

## Step 6: First Import

1. Navigate to **IBKR Inbox**
2. Click **Import Now** to trigger your first manual import
3. Review the imported transactions
4. Allocate each transaction to your portfolios
5. Click **Process Transaction** to finalize

## Managing IBKR Transactions

The IBKR Inbox provides a complete interface for managing imported transactions:

**Pending Tab:**
- Review newly imported transactions
- Allocate to one or more portfolios
- Ignore or delete unwanted transactions

**Processed Tab:**
- View allocation details
- Modify existing allocations
- Unallocate to start over

**Transaction Badges:**
- IBKR transactions show an **IBKR** badge in portfolio views
- Helps distinguish imported vs manually-entered transactions

**📖 For Complete Transaction Lifecycle Documentation:**
See [docs/IBKR_TRANSACTION_LIFECYCLE.md](IBKR_TRANSACTION_LIFECYCLE.md) for technical details on:
- Transaction status flow and auto-revert mechanism
- API endpoints and backend implementation
- Troubleshooting and testing scenarios

## IBKR Flex API Details

### API Endpoints:
- **Request Report**: `https://gdcdyn.interactivebrokers.com/Universal/servlet/FlexStatementService.SendRequest`
- **Get Statement**: `https://gdcdyn.interactivebrokers.com/Universal/servlet/FlexStatementService.GetStatement`

### How it works:
1. Application sends request with your Token and Query ID
2. IBKR responds with a Reference Code
3. Application polls for statement using Reference Code
4. Once ready, IBKR returns the statement XML/CSV
5. Application parses and stores transactions in local database
6. Statements are cached locally for 24 hours to avoid duplicate API calls

### Rate Limiting:
- IBKR Flex API has rate limits
- The application caches responses for 24 hours
- Automated imports run tue - sat at 05:05
- Manual imports can be triggered anytime (uses cache if available)

## Troubleshooting

### "Invalid Token" Error
- Verify your token is correct (no extra spaces)
- Generate a new token in IBKR Portal if needed
- Update the token in application settings

### "Query Not Found" Error
- Verify your Query ID is correct
- Ensure the query exists in your IBKR account
- Check that the query is active (not deleted)

### No Transactions Imported
- Verify your query date range includes recent transactions
- Check that you have transactions in your IBKR account during that period
- Review application logs for detailed error messages

### Duplicate Transactions
- The system automatically prevents duplicates using IBKR's transaction ID
- If you see duplicates, check the logs for warnings

## Security Notes

1. **Token Storage**: Your Flex Token is encrypted using the `IBKR_ENCRYPTION_KEY` before being stored in the database
2. **Write-Only**: The token is never displayed in the UI after being saved (write-only)
3. **Environment Variable**: Keep your `IBKR_ENCRYPTION_KEY` secure and never commit it to version control
4. **Key Rotation**: If you need to change the encryption key, you must re-enter your IBKR token

## Support

For IBKR-specific questions:
- IBKR Flex Query Documentation: https://www.interactivebrokers.com/en/software/am/am/reports/activityflexqueries.htm
- IBKR Client Portal: https://www.interactivebrokers.com/portal

For application issues:
- Check the Log Viewer in the application
- Review error messages in the IBKR Inbox
- Report issues on GitHub: https://github.com/ndewijer/investment-portfolio-manager/issues
