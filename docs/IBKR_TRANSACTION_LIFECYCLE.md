# IBKR Transaction Lifecycle Management

## Overview

This document describes the lifecycle management of IBKR (Interactive Brokers) transactions in the Investment Portfolio Manager, including how transactions flow from import to allocation, and how they are managed throughout their lifecycle.

## Problem Statement

### Original Issue

When an IBKR transaction was allocated to portfolios and then the portfolio transaction(s) were deleted, the IBKR transaction would effectively disappear from the system:

- The `IBKRTransaction` record remained with `status="processed"`
- The `IBKRTransactionAllocation` records were cascade-deleted when portfolio `Transaction` records were deleted
- The transaction did not reappear in the IBKR Inbox
- Users had no way to view or recover processed transactions

This was an oversight in the original implementation that has been addressed.

## Solution Overview

The solution implements:

1. **Auto-revert mechanism**: When all portfolio transactions linked to an IBKR transaction are deleted, the IBKR transaction automatically reverts to "pending" status
2. **Processed transaction viewing**: Users can view processed IBKR transactions via tabs in the IBKR Inbox
3. **Allocation management**: Users can view, modify, or unallocate processed transactions
4. **Visual linking**: Portfolio transactions show an IBKR badge indicating they originated from IBKR imports

## Transaction Status Flow

```
┌─────────────┐
│   pending   │ ← Import from IBKR
└──────┬──────┘
       │
       │ Allocate to portfolio(s)
       ├──────────────────┐
       │                  │
       ▼                  ▼
┌─────────────┐    ┌──────────┐
│  processed  │    │ ignored  │
└──────┬──────┘    └──────────┘
       │
       │ Delete all allocations
       │ (auto-revert)
       │
       ▼
┌─────────────┐
│   pending   │ ← Ready for reallocation
└─────────────┘
```

## Database Schema

### Key Models

#### IBKRTransaction
- **Purpose**: Parent record for IBKR imports
- **Key Fields**:
  - `status`: 'pending', 'processed', or 'ignored'
  - `processed_at`: Timestamp when allocated
  - `ibkr_transaction_id`: Unique IBKR identifier
- **Relationships**:
  - One-to-many with `IBKRTransactionAllocation` (cascade delete)

#### IBKRTransactionAllocation
- **Purpose**: Junction table linking IBKR transactions to portfolios
- **Key Fields**:
  - `allocation_percentage`: Percentage allocated to this portfolio
  - `allocated_amount`: Amount in currency
  - `allocated_shares`: Number of shares
  - `transaction_id`: Link to created portfolio Transaction
- **Foreign Keys**:
  - `ibkr_transaction_id` → `IBKRTransaction.id` (CASCADE)
  - `portfolio_id` → `Portfolio.id` (CASCADE)
  - `transaction_id` → `Transaction.id` (CASCADE)

#### Transaction
- **Purpose**: Portfolio transaction records
- **Relationships**:
  - One-to-one with `IBKRTransactionAllocation` (via `transaction_id` FK)

### Cascade Behavior

When a `Transaction` is deleted:
1. Database cascade deletes the `IBKRTransactionAllocation` record (via `ondelete="CASCADE"`)
2. Application logic counts remaining allocations
3. If count reaches 0, application reverts `IBKRTransaction.status` to "pending"

## Backend Implementation

### 1. Transaction Deletion with Auto-Revert

**File**: `backend/app/routes/transaction_routes.py`

**Logic**:
```python
# Before deleting a transaction:
1. Check if transaction has an IBKRTransactionAllocation
2. If yes, get the ibkr_transaction_id
3. Count how many allocations exist for that IBKR transaction
4. If count == 1 (this is the last one):
   - Revert IBKRTransaction.status to "pending"
   - Clear IBKRTransaction.processed_at
5. Delete the transaction (allocation cascade-deletes)
```

**Scenarios**:
- **Partial deletion** (2 allocations, delete 1): Status stays "processed"
- **Complete deletion** (1 allocation, delete 1): Status reverts to "pending"
- **Sequential deletion** (2 allocations, delete both): Last one triggers revert

### 2. Unallocate Endpoint

**Endpoint**: `POST /ibkr/inbox/<transaction_id>/unallocate`

**Purpose**: Bulk remove all allocations and revert status

**Logic**:
```python
1. Validate IBKR transaction exists and is "processed"
2. Get all IBKRTransactionAllocation records
3. For each allocation:
   - Delete the associated Transaction record
   - Allocation cascade-deletes automatically
4. Revert IBKRTransaction status to "pending"
5. Clear processed_at timestamp
6. Return count of deleted transactions
```

**Response**:
```json
{
  "success": true,
  "message": "Transaction unallocated successfully. 2 portfolio transactions deleted."
}
```

### 3. View Allocations Endpoint

**Endpoint**: `GET /ibkr/inbox/<transaction_id>/allocations`

**Purpose**: Get detailed allocation information for a processed transaction

**Response**:
```json
{
  "ibkr_transaction_id": "uuid",
  "status": "processed",
  "allocations": [
    {
      "id": "allocation-uuid",
      "portfolio_id": "portfolio-uuid",
      "portfolio_name": "Portfolio A",
      "allocation_percentage": 50.0,
      "allocated_amount": 500.00,
      "allocated_shares": 10.5,
      "transaction_id": "transaction-uuid",
      "transaction_date": "2025-01-15"
    }
  ]
}
```

### 4. Modify Allocations Endpoint

**Endpoint**: `PUT /ibkr/inbox/<transaction_id>/allocations`

**Purpose**: Update allocation percentages for a processed transaction

**Request Body**:
```json
{
  "allocations": [
    {
      "portfolio_id": "portfolio-uuid-1",
      "percentage": 70.0
    },
    {
      "portfolio_id": "portfolio-uuid-2",
      "percentage": 30.0
    }
  ]
}
```

**Logic**:
```python
1. Validate percentages sum to 100%
2. Calculate new amounts and shares based on percentages
3. For each allocation:
   - If portfolio exists in current allocations:
     - Update Transaction record with new amounts
     - Update IBKRTransactionAllocation record
   - If portfolio is new:
     - Create new Transaction record
     - Create new IBKRTransactionAllocation record
   - If portfolio removed:
     - Delete Transaction record
     - Allocation cascade-deletes
4. Keep IBKR transaction status as "processed"
```

### 5. Transaction IBKR Link

**Endpoints**: `GET /transactions` and `GET /transactions/<id>`

**Enhancement**: Add IBKR link information to transaction responses

**Response Addition**:
```json
{
  "id": "transaction-uuid",
  ...existing fields...,
  "ibkr_linked": true,
  "ibkr_transaction_id": "ibkr-uuid"
}
```

**Logic**:
```python
# For each transaction
ibkr_allocation = IBKRTransactionAllocation.query.filter_by(
    transaction_id=transaction.id
).first()

response["ibkr_linked"] = bool(ibkr_allocation)
response["ibkr_transaction_id"] = (
    ibkr_allocation.ibkr_transaction_id if ibkr_allocation else None
)
```

## Frontend Implementation

### 1. IBKR Inbox Tabs

**File**: `frontend/src/pages/IBKRInbox.js`

**UI Changes**:
- Add tab buttons above the transaction table
- Tabs: "Pending" | "Processed"
- Active tab highlighted
- Fetch transactions based on selected tab

**State**:
```javascript
const [selectedStatus, setSelectedStatus] = useState('pending');
```

**Fetch Logic**:
```javascript
const fetchTransactions = async (status = selectedStatus) => {
  const response = await api.get('ibkr/inbox', {
    params: { status },
  });
  setTransactions(response.data);
};
```

### 2. Processed Transaction Actions

**Actions for Processed Status**:
- **View Details**: Opens allocation modal in read-only mode
- **Modify**: Opens allocation modal in edit mode
- **Unallocate**: Confirms and calls unallocate endpoint

**Action Buttons**:
```javascript
{item.status === 'pending' ? (
  <>
    <button onClick={() => handleAllocateTransaction(item)}>Allocate</button>
    <button onClick={() => handleIgnoreTransaction(item.id)}>Ignore</button>
    <button onClick={() => handleDeleteTransaction(item.id)}>Delete</button>
  </>
) : (
  <>
    <button onClick={() => handleViewDetails(item)}>View Details</button>
    <button onClick={() => handleModifyAllocation(item)}>Modify</button>
    <button onClick={() => handleUnallocate(item.id)}>Unallocate</button>
  </>
)}
```

### 3. Allocation Modal Modes

**Modes**:
- **create**: For new allocations (existing behavior)
- **view**: Read-only display of existing allocations
- **edit**: Modify existing allocations

**View Mode**:
- Disable all inputs (select, input fields)
- Hide add/remove allocation buttons
- Show allocation details with portfolio names
- Show links to portfolio transactions
- Only show "Close" button

**Edit Mode**:
- Pre-populate with existing allocations
- Allow percentage changes
- Allow adding/removing portfolios
- Submit button calls modify endpoint

**Implementation**:
```javascript
const [modalMode, setModalMode] = useState('create'); // 'create' | 'view' | 'edit'
const [existingAllocations, setExistingAllocations] = useState([]);

const handleViewDetails = async (transaction) => {
  const response = await api.get(`ibkr/inbox/${transaction.id}/allocations`);
  setExistingAllocations(response.data.allocations);
  setModalMode('view');
  setSelectedTransaction(transaction);
};

const handleModifyAllocation = async (transaction) => {
  const response = await api.get(`ibkr/inbox/${transaction.id}/allocations`);
  setExistingAllocations(response.data.allocations);
  setModalMode('edit');
  setSelectedTransaction(transaction);
  // Pre-populate allocations state
  setAllocations(response.data.allocations.map(a => ({
    portfolio_id: a.portfolio_id,
    percentage: a.allocation_percentage
  })));
};
```

### 4. IBKR Badge in Portfolio Transactions

**File**: `frontend/src/components/TransactionsTable.js`

**New Column**:
```javascript
{
  key: 'source',
  header: 'Source',
  render: (_, transaction) => (
    transaction.ibkr_linked ? (
      <span className="ibkr-badge" title="From IBKR import">
        IBKR
      </span>
    ) : null
  )
}
```

**CSS** (in TransactionsTable.css or similar):
```css
.ibkr-badge {
  display: inline-block;
  padding: 0.2rem 0.5rem;
  background-color: #0066cc;
  color: white;
  border-radius: 3px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}
```

## Testing Scenarios

### Backend Tests

#### Test 1: Partial Deletion
**Setup**:
1. Import IBKR transaction
2. Allocate 50% to Portfolio A, 50% to Portfolio B
3. Delete Portfolio A's transaction

**Expected**:
- IBKR transaction status remains "processed"
- Portfolio B's transaction still exists
- One IBKRTransactionAllocation remains

#### Test 2: Complete Deletion
**Setup**:
1. Import IBKR transaction
2. Allocate 100% to Portfolio A
3. Delete Portfolio A's transaction

**Expected**:
- IBKR transaction status reverts to "pending"
- processed_at is null
- No IBKRTransactionAllocation records remain
- Transaction appears in IBKR Inbox under "Pending" tab

#### Test 3: Sequential Deletion
**Setup**:
1. Import IBKR transaction
2. Allocate 50% to Portfolio A, 50% to Portfolio B
3. Delete Portfolio A's transaction
4. Delete Portfolio B's transaction

**Expected**:
- After first deletion: Status remains "processed"
- After second deletion: Status reverts to "pending"
- Transaction appears in IBKR Inbox under "Pending" tab

#### Test 4: Unallocate Endpoint
**Setup**:
1. Import IBKR transaction
2. Allocate to multiple portfolios
3. Call POST /ibkr/inbox/{id}/unallocate

**Expected**:
- All portfolio transactions deleted
- All allocations deleted
- Status reverted to "pending"
- Response includes count of deleted transactions

#### Test 5: Modify Allocations
**Setup**:
1. Import IBKR transaction
2. Allocate 50% to Portfolio A, 50% to Portfolio B
3. Call PUT /ibkr/inbox/{id}/allocations with 70/30 split

**Expected**:
- Transaction amounts updated to 70/30 split
- Share counts updated proportionally
- Status remains "processed"
- No new transactions created

### Frontend Tests

#### Test 1: View Processed Transactions
**Steps**:
1. Navigate to IBKR Inbox
2. Click "Processed" tab

**Expected**:
- All processed transactions displayed
- Correct action buttons shown (View, Modify, Unallocate)
- No "Allocate" button visible

#### Test 2: View Allocation Details
**Steps**:
1. On "Processed" tab, click "View Details"

**Expected**:
- Modal opens with allocation information
- Portfolio names displayed
- Percentages and amounts shown
- All inputs disabled
- Only "Close" button visible

#### Test 3: Modify Allocations
**Steps**:
1. On "Processed" tab, click "Modify"
2. Change allocation percentages
3. Click "Update Allocation"

**Expected**:
- Modal opens with current allocations pre-populated
- Can change percentages
- Validation shows if total ≠ 100%
- After submit, transaction updated
- Success message displayed

#### Test 4: Unallocate Transaction
**Steps**:
1. On "Processed" tab, click "Unallocate"
2. Confirm in dialog

**Expected**:
- Confirmation dialog appears
- After confirm, transaction unallocated
- Transaction moves to "Pending" tab
- Success message displayed

#### Test 5: IBKR Badge Display
**Steps**:
1. Allocate IBKR transaction to portfolio
2. Navigate to that portfolio's transactions

**Expected**:
- Transaction has IBKR badge in "Source" column
- Tooltip shows "From IBKR import"
- Badge is visually distinct

#### Test 6: Delete Transaction Auto-Revert
**Steps**:
1. Allocate IBKR transaction to single portfolio
2. Navigate to portfolio transactions
3. Delete the transaction
4. Go back to IBKR Inbox → "Pending" tab

**Expected**:
- Transaction reappears in "Pending" tab
- Can be reallocated

## Migration Notes

### No Database Migration Required

The existing database schema already supports this functionality:
- `IBKRTransaction.status` field exists with 'pending'/'processed'/'ignored' values
- `IBKRTransactionAllocation` has correct cascade behavior
- No new tables or columns needed

### Application Logic Only

All changes are in application logic:
- Route handlers
- Frontend UI components
- Status management logic

## API Endpoints Summary

### New Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/ibkr/inbox/<id>/unallocate` | Remove all allocations and revert to pending |
| GET | `/ibkr/inbox/<id>/allocations` | Get allocation details for processed transaction |
| PUT | `/ibkr/inbox/<id>/allocations` | Update allocation percentages |

### Modified Endpoints

| Method | Endpoint | Change |
|--------|----------|--------|
| GET | `/transactions` | Add ibkr_linked and ibkr_transaction_id fields |
| GET | `/transactions/<id>` | Add ibkr_linked and ibkr_transaction_id fields |
| DELETE | `/transactions/<id>` | Add auto-revert logic for IBKR transactions |

## Logging and Audit Trail

All IBKR transaction status changes are logged:

```python
logger.log(
    level=LogLevel.INFO,
    category=LogCategory.IBKR,
    message="IBKR transaction status reverted to pending",
    details={
        "ibkr_transaction_id": ibkr_txn.ibkr_transaction_id,
        "reason": "All allocations deleted",
        "allocation_count": 0
    }
)
```

Log events:
- Transaction allocated: `status=processed`
- Transaction unallocated: `status=pending`
- Allocation modified: `status=processed` (no change)
- Auto-revert triggered: `status=pending`

## Future Enhancements

Potential future improvements:

1. **Partial Unallocation**: Allow removing individual portfolio allocations
2. **Allocation History**: Track historical changes to allocations
3. **Bulk Operations**: Process multiple IBKR transactions at once
4. **Allocation Templates**: Save common allocation patterns
5. **Auto-allocation Rules**: Automatically allocate based on rules

## Troubleshooting

### Transaction Doesn't Reappear After Deletion

**Check**:
1. Database status: `SELECT status FROM ibkr_transaction WHERE id = '...'`
2. Application logs for revert message
3. Frontend tab selection (ensure on "Pending" tab)

**Fix**:
```sql
-- Manually revert if needed
UPDATE ibkr_transaction
SET status = 'pending', processed_at = NULL
WHERE id = '...';
```

### Allocation Count Mismatch

**Check**:
```sql
-- Count allocations
SELECT COUNT(*) FROM ibkr_transaction_allocation
WHERE ibkr_transaction_id = '...';

-- Check associated transactions
SELECT t.id FROM transaction t
JOIN ibkr_transaction_allocation a ON a.transaction_id = t.id
WHERE a.ibkr_transaction_id = '...';
```

### Cascade Delete Not Working

**Verify foreign key constraints**:
```sql
-- Check constraint definition
SELECT sql FROM sqlite_master
WHERE type = 'table'
AND name = 'ibkr_transaction_allocation';
```

Should include: `FOREIGN KEY(transaction_id) REFERENCES transaction(id) ON DELETE CASCADE`

## References

- **Models**: See `backend/app/models.py` lines 535-638
- **IBKR Routes**: See `backend/app/routes/ibkr_routes.py`
- **Transaction Routes**: See `backend/app/routes/transaction_routes.py`
- **IBKR Service**: See `backend/app/services/ibkr_transaction_service.py`
- **Fund Matching**: See `backend/app/services/fund_matching_service.py`

---

**Last Updated**: 2025-11-05 (Version 1.3.0)
**Maintained By**: @ndewijer
