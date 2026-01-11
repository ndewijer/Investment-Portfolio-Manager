# IBKR Transaction Processing Features

This document describes the features available for processing IBKR transactions in the Investment Portfolio Manager.

## Table of Contents

- [Single Transaction Processing](#single-transaction-processing)
- [Bulk Transaction Processing](#bulk-transaction-processing)
- [Allocation Presets](#allocation-presets)
- [Default Allocation on Import](#default-allocation-on-import)
- [Transaction Lifecycle](#transaction-lifecycle)
- [Best Practices](#best-practices)

---

## Single Transaction Processing

### Overview

Process individual IBKR transactions by allocating them to one or more portfolios.

### How to Use

1. Navigate to **IBKR Inbox** from the navigation bar
2. View pending transactions in the **Pending** tab
3. Click **Allocate** button for the transaction you want to process
4. Select portfolio(s) and allocation percentages
5. Click **Process Transaction**

### Features

- **Fund Matching**: System automatically matches transactions to funds by ISIN or symbol
- **Eligible Portfolios**: Only portfolios containing the matched fund are shown
- **Multiple Portfolios**: Split transactions across multiple portfolios
- **Percentage Validation**: System ensures allocations sum to exactly 100%
- **Real-time Amount Calculation**: See exact currency amounts and shares as you enter percentages

### Allocation Modal

The allocation modal shows:
- **Transaction details**: Compact single-line format showing symbol, description, type, amount, and quantity
- **Fund matching information**: Shows if fund was matched by ISIN or symbol
- **Portfolio selection**: Dropdowns to select portfolios
- **Percentage inputs**: Enter allocation percentages
- **Real-time amount display**: Below each percentage field, see the calculated amount and shares for that allocation
- **Total validation**: Green/red indicator showing if allocations sum to 100%

### Amount Display

As you enter allocation percentages, the system automatically displays:
- **Currency amount**: The exact amount in transaction currency for that percentage
- **Share count**: The number of shares (if applicable) for that percentage
- **Example**: If allocating 50% of a €2,500 transaction with 50 shares:
  - Display shows: "€1,250.00 EUR • 25.0000 shares"

The amount display automatically adjusts its position based on the layout:
- Single allocation: Positioned further right (no Remove button)
- Multiple allocations: Positioned to accommodate the Remove button

---

## Bulk Transaction Processing

### Overview

Process multiple IBKR transactions at once using identical allocation settings. This feature saves time when you want to allocate several transactions to the same portfolio distribution.

### How to Use

1. Navigate to **IBKR Inbox** → **Pending** tab
2. Use checkboxes to select multiple transactions:
   - Click individual checkboxes for specific transactions
   - Use **Select All** to select all pending transactions
3. Click **Bulk Allocate** button
4. **System checks eligibility** for all selected transactions
5. Review eligibility warnings/confirmation
6. Configure portfolio allocations (same for all selected transactions)
7. Click **Process [N] Transactions**

### Eligibility Checks

When you click "Bulk Allocate", the system automatically:

✅ **Checks each transaction** for fund matching (by ISIN or symbol)
✅ **Identifies eligible portfolios** that contain each fund
✅ **Finds common portfolios** that can handle ALL selected transactions
✅ **Shows warnings** for problematic transactions
✅ **Filters portfolio list** to only show common eligible portfolios

**Example Messages**:
- ✓ "All transactions can be allocated to 2 portfolio(s)" - All good! Shows 2 portfolios in dropdown
- ⚠️ "3 transaction(s) have funds not found in the system. Please add these funds first." - **No portfolios shown**
- ⚠️ "2 transaction(s) have funds that are not assigned to any portfolio. Please add these funds to portfolios first." - **No portfolios shown**
- ⚠️ "No common portfolios found. Transactions have funds in different portfolios." - **No portfolios shown**

### Important Notes

- ✅ **Same Allocations**: All selected transactions receive identical portfolio distribution percentages
- ✅ **Eligibility Validation**: System checks all transactions before showing allocation modal
- ✅ **Common Portfolios Only**: Only portfolios that can handle ALL transactions are shown
- ✅ **Individual Processing**: Each transaction is processed separately to its eligible portfolios
- ✅ **Error Handling**: If some transactions fail, others continue processing
- ✅ **Result Summary**: Shows count of successful and failed transactions

### Limitations & Protections

- ❌ Cannot set different allocations for each transaction in bulk mode
- ✅ **Portfolio list is empty** if any transaction has unmatched funds - this prevents invalid processing
- ✅ **Only common portfolios shown** - system automatically filters to portfolios that can handle ALL transactions
- ℹ️ If some transactions have issues, you'll see which ones and can fix them before processing

### Example Use Cases

**Scenario 1: Monthly Investment**
- You import 10 transactions for the same fund
- You want to allocate all to your "Growth Portfolio"
- Select all 10 transactions → Bulk Allocate → 100% Growth Portfolio

**Scenario 2: Split Allocation**
- You have 5 dividend transactions
- You want to split all 50/50 between two portfolios
- Select all 5 → Bulk Allocate → 50% Portfolio A, 50% Portfolio B

**Scenario 3: Three-Way Split**
- You have 8 buy transactions for different funds
- You want to allocate all 33.33% across three portfolios
- Select all 8 → Bulk Allocate → Use "Equal Distribution" preset

---

## Allocation Presets

### Overview

Allocation presets are quick-action buttons that automatically calculate and fill allocation percentages, saving manual calculation time.

### Available Presets

#### 📊 Equal Distribution

**What it does**: Distributes 100% equally among all currently selected portfolios in the allocation list.

**How it works**:
1. Counts the number of portfolio rows in your allocation
2. Divides 100% by that number
3. Sets each portfolio to the equal percentage
4. Adjusts the last portfolio to ensure exactly 100% (handles rounding)

**Example**:
- You have 3 portfolios added
- Click "Equal Distribution"
- Result: Each gets 33.33%, last adjusted to 33.34%

**Use When**:
- You want to split evenly among several portfolios
- You already added the desired portfolios to the list
- You need a quick 50/50, 33/33/33, or 25/25/25/25 split

**Important**: Only affects portfolios already in the list. Does NOT automatically add all available portfolios.

#### ➕ Distribute Remaining

**What it does**: Distributes remaining percentage only to portfolios that currently have 0%.

**How it works**:
1. Calculates remaining percentage (100% - current total)
2. Finds all portfolios with exactly 0%
3. Divides remaining percentage equally among those portfolios
4. Adjusts last portfolio to ensure exactly 100%

**Example 1**: Basic usage
- Portfolio A: 60%
- Portfolio B: 0%
- Portfolio C: 0%
- Click "Distribute Remaining"
- Result: A stays 60%, B gets 20%, C gets 20%

**Example 2**: Single remainder
- Portfolio A: 70%
- Portfolio B: 25%
- Portfolio C: 0%
- Click "Distribute Remaining"
- Result: A stays 70%, B stays 25%, C gets 5%

**Use When**:
- You manually set some percentages and want to auto-fill the rest
- You have a primary portfolio allocation and want to split remainder
- You're adjusting existing allocations

**Error Messages**:
- "All 100% is already allocated" - No remaining percentage to distribute
- "No portfolios with 0% to distribute remaining percentage" - All portfolios already have values

### Preset Workflow Example

**Goal**: Allocate 60% to Growth, split remaining 40% between Income and Balanced

Steps:
1. Click "Allocate" or "Bulk Allocate"
2. Add three portfolios (Growth, Income, Balanced)
3. Manually set Growth to 60%
4. Leave Income and Balanced at 0%
5. Click "Distribute Remaining"
6. Result: Growth 60%, Income 20%, Balanced 20%
7. Click "Process Transaction"

---

## Default Allocation on Import

### Overview

Automatically allocate imported IBKR transactions to your portfolios using a configured default preset. This feature eliminates the need to manually allocate transactions that follow the same portfolio distribution pattern.

**Added in**: Version 1.3.5

### How It Works

1. **Configure Once**: Set up your default allocation preset in Configuration → IBKR Setup
2. **Enable Auto-Allocation**: Toggle "Enable default allocation on import"
3. **Automatic Processing**: When automated imports run (Tuesday-Saturday at 06:30), matching transactions are automatically allocated
4. **Manual Fallback**: Transactions that can't be auto-allocated remain in "Pending" for manual processing

### Setup Instructions

1. Navigate to **Configuration** → **IBKR Setup** tab
2. Scroll to **Default Allocation on Import** section
3. Check **Enable default allocation on import**
4. Click **Configure Default Allocation** button
5. Select portfolios and enter allocation percentages (must sum to 100%)
6. Click **Apply Preset**
7. Click **Update Configuration** to save

### Configuration States

The UI shows two distinct states:

- **Current preset**: The saved configuration in the database
- **Updated preset** (with "Pending save" badge): Changes staged but not yet saved

This makes it clear when you need to click "Update Configuration" to persist changes.

### Eligibility Requirements

For a transaction to be automatically allocated, it must meet ALL these conditions:

✅ **Fund Exists**: Transaction's fund (by ISIN or symbol) must exist in the system
✅ **Fund in ALL Portfolios**: The fund must be present in ALL configured portfolios in your default preset
✅ **No Errors**: Transaction import must complete without errors

### What Happens During Auto-Allocation

**Successful Auto-Allocation**:
1. System checks if fund exists in all configured portfolios
2. Transaction is allocated using your default percentages
3. Status changes from "pending" to "processed"
4. Appears in **Processed** tab with allocation details
5. Logged with ✅ success message

**Failed Auto-Allocation** (transaction stays pending):
- ❌ Fund not found in system
- ❌ Fund missing from one or more configured portfolios
- ❌ System error during allocation

Failed transactions remain in "Pending" tab for manual processing.

### Import Logging

Check **Configuration** → **System Settings** → **Logs** to see auto-allocation activity:

```
ℹ️ Starting automated IBKR import
ℹ️ Applying default allocations to newly imported transactions
✅ Auto-allocated transaction VWCE (iShares Core MSCI World)
ℹ️ Auto-allocated 5 transaction(s) using default preset
✅ Automated IBKR import completed: 8 imported, 2 skipped, 0 errors, 5 auto-allocated
```

### Use Cases

**Scenario 1: Regular Investment Plan**
- You import the same fund monthly
- Always allocate 60% Portfolio A, 40% Portfolio B
- Set default allocation → All future imports auto-allocate

**Scenario 2: Multi-Portfolio Strategy**
- You have 3 portfolios with same funds
- Always split 33.33% / 33.33% / 33.34%
- Configure once → No more manual allocation

**Scenario 3: Mixed Strategy**
- Most transactions follow standard allocation
- Some need special handling
- Default allocation handles 80% automatically
- Manually allocate the 20% that need custom treatment

### Modifying Default Allocations

To change your default allocation:

1. Go to **Configuration** → **IBKR Setup**
2. Click **Configure Default Allocation**
3. Modify portfolios or percentages
4. Click **Apply Preset**
5. Click **Update Configuration**

The UI shows both old and new presets until you save, making it clear what will change.

### Disabling Auto-Allocation

To stop auto-allocating but keep your preset:

1. Uncheck **Enable default allocation on import**
2. Click **Update Configuration**
3. Your preset is saved but not applied

To completely remove the preset:

1. Uncheck **Enable default allocation on import**
2. Click **Configure Default Allocation**
3. Clear all allocations (the system will prompt)
4. Click **Update Configuration**

### Important Notes

⚠️ **No Retroactive Application**
- Default allocations only apply to NEW imports
- Existing pending transactions are NOT automatically processed
- Use Bulk Allocate for existing pending transactions

⚠️ **Fund Must Exist in ALL Portfolios**
- If fund is in Portfolio A but not Portfolio B, transaction stays pending
- This prevents partial allocations or errors
- Add fund to all portfolios first, then transaction auto-allocates on next import

ℹ️ **Import Summary**
- Each import shows auto-allocation count in success message
- Check logs for detailed per-transaction results
- Errors don't stop other transactions from processing

### Best Practices

**✅ Do**:
- Use default allocation for recurring, standard investments
- Keep funds synchronized across configured portfolios
- Monitor logs to ensure auto-allocation is working
- Test with 1-2 imports before enabling for large volumes

**❌ Don't**:
- Configure portfolios that don't contain the same funds
- Expect pending transactions to be retroactively processed
- Use for funds that frequently need custom allocations
- Forget to click "Update Configuration" after changing preset

### Troubleshooting

**"Transactions aren't auto-allocating"**

Check:
1. ✅ Is "Enable default allocation on import" checked?
2. ✅ Did you click "Update Configuration" after setting preset?
3. ✅ Does the fund exist in ALL configured portfolios?
4. ✅ Is automated import enabled?
5. ✅ Check logs for specific error messages

**"Some transactions auto-allocate, others don't"**

Likely cause: Funds not present in all portfolios
- Check which fund failed in logs
- Add missing fund to all portfolios in default preset
- Next import will auto-allocate that fund

**"I see 'Updated preset' but it's not working"**

You need to save the configuration:
- "Updated preset" with "Pending save" badge means NOT saved
- Click "Update Configuration" button to persist changes
- "Current preset" shows what's actually being used

### Technical Details

**Processing Order**:
1. Automated import runs (06:30, Tue-Sat)
2. Transactions imported to inbox
3. Import status: "imported"
4. Default allocation check runs
5. Eligible transactions allocated
6. Status changes: "pending" → "processed"

**Performance**:
- Auto-allocation adds ~50ms per transaction
- Failures are logged but don't stop other allocations
- Each transaction processed independently

**API Endpoints**:
- Configuration: `GET/POST /api/ibkr/config`
- Fields: `default_allocation_enabled`, `default_allocations`

---

## Transaction Lifecycle

### Status Flow

```
Pending → Processed → (Can be Modified/Unallocated)
  ↓
Ignored (if not relevant)
  ↓
Deleted (if imported incorrectly)
```

### Available Actions by Status

#### Pending Transactions
- **Allocate**: Process transaction to portfolios
- **Ignore**: Mark as irrelevant (changes status to "ignored")
- **Delete**: Remove from system entirely

#### Processed Transactions
- **View Details**: See allocation breakdown
- **Modify**: Change allocation percentages or portfolios
- **Unallocate**: Revert to pending (deletes portfolio transactions)

### Bulk Actions (Pending Only)

- **Bulk Allocate**: Process multiple with same settings
- **Bulk Ignore**: Ignore multiple at once
- **Bulk Delete**: Delete multiple at once

---

## Best Practices

### Before Bulk Processing

1. **Check Fund Matching**: Review that all transactions have matching funds
2. **Verify Portfolios**: Ensure selected portfolios contain the required funds
3. **Group Similar**: Select transactions for the same type of allocation
4. **Start Small**: Test with 2-3 transactions before bulk processing many

### Using Presets Effectively

1. **Equal Distribution**:
   - Add portfolios first, then use preset
   - Use for truly equal splits
   - Verify the result matches your intent

2. **Distribute Remaining**:
   - Set your primary allocations first
   - Use for "remainder" portfolios
   - Double-check zeros are correct before clicking

### Allocation Strategies

**Strategy 1: Fixed Primary + Split Remainder**
- Best for: Regular investments with a core portfolio
- Steps:
  1. Set primary portfolio (e.g., 70%)
  2. Add secondary portfolios
  3. Use "Distribute Remaining"

**Strategy 2: Equal Split**
- Best for: Balanced portfolio approach
- Steps:
  1. Add all desired portfolios
  2. Use "Equal Distribution"

**Strategy 3: Manual Control**
- Best for: Specific percentage requirements
- Steps:
  1. Manually enter each percentage
  2. Watch the total indicator (green = valid, red = invalid)

### Common Mistakes to Avoid

❌ **Don't**: Bulk allocate transactions for different funds without checking eligibility
✅ **Do**: Group transactions by fund or use single transaction processing

❌ **Don't**: Assume "Equal Distribution" adds all portfolios
✅ **Do**: Add portfolios to the list first, then use the preset

❌ **Don't**: Use "Distribute Remaining" when you want to replace all allocations
✅ **Do**: Use "Equal Distribution" for fresh percentage splits

❌ **Don't**: Process hundreds of transactions at once without testing
✅ **Do**: Test with small batches first

---

## Troubleshooting

### "Allocations must sum to exactly 100%"

**Cause**: Your percentages don't total 100% (shown in red)

**Solutions**:
- Use "Equal Distribution" to auto-calculate
- Use "Distribute Remaining" if you have partial allocations
- Manually adjust percentages to reach 100.00%

### "Cannot allocate the same portfolio multiple times"

**Cause**: Same portfolio selected in multiple rows

**Solutions**:
- Remove duplicate rows
- Combine percentages into single row for that portfolio

### "Please select a portfolio for each allocation"

**Cause**: One or more portfolio dropdowns not selected

**Solutions**:
- Select a portfolio in each row
- Remove empty rows using "Remove" button

### Bulk Processing Partial Failures

**Cause**: Some transactions couldn't be processed (fund not in portfolio, etc.)

**Response**:
- System shows "X processed, Y failed"
- Check individual transaction error messages
- Failed transactions remain in pending state
- Successfully processed transactions are moved to "Processed" tab

### Fund Not Found Errors

**Cause**: IBKR transaction symbol/ISIN doesn't match any fund in system

**Solutions**:
- Add the fund to your funds list first
- Ensure ISIN matches exactly
- Check symbol matches if ISIN not available
- Contact support if fund should be matched

---

## Keyboard Shortcuts

### Checkbox Selection
- **Click**: Toggle single transaction
- **Shift+Click**: (Future) Range selection
- **Ctrl/Cmd+A**: (Future) Select all visible

### Modal Navigation
- **Enter**: (Future) Submit allocation
- **Escape**: Close modal
- **Tab**: Navigate between portfolio fields

---

## Technical Details

### API Endpoints

**Single Transaction**:
- `POST /api/ibkr/inbox/{transaction_id}/allocate`

**Bulk Processing**:
- `POST /api/ibkr/inbox/bulk-allocate`

**Eligible Portfolios**:
- `GET /api/ibkr/inbox/{transaction_id}/eligible-portfolios`

### Validation Rules

- Allocations must sum to 100.00% (±0.01% tolerance)
- Each allocation must have a selected portfolio
- No duplicate portfolios in allocation list
- Transaction must be in "pending" status for processing
- Transaction must be in "processed" status for modification

### Performance

- Bulk operations process sequentially (not parallel)
- Average processing time: ~200ms per transaction
- Recommended batch size: Up to 50 transactions
- Failures don't affect other transactions in batch

---

## Feature Comparison Matrix

| Feature | Single | Bulk |
|---------|--------|------|
| Process transactions | ✅ | ✅ |
| Different allocations per transaction | ✅ | ❌ |
| Same allocations for all | N/A | ✅ |
| Fund matching check | ✅ | ⚠️ Individual |
| Eligible portfolio filtering | ✅ | ❌ |
| Error handling | Immediate | Per transaction |
| Use allocation presets | ✅ | ✅ |
| Modify after processing | ✅ | ✅ |
| Speed (10 transactions) | Slower | Faster |

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.3.5 | 2025-12 | Added default allocation on import feature |
| 1.3.1 | 2025-11 | Added bulk processing and allocation presets |
| 1.3.0 | 2025-11 | Initial IBKR transaction processing |

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/ndewijer/investment-portfolio-manager/issues
- Documentation: `docs/IBKR_SETUP.md`
- Architecture: `docs/ARCHITECTURE.md`

---

**Last Updated**: 2025-12-17 (Version 1.3.5)
**Maintained By**: @ndewijer
