# Commercial Math Verification — 2026-08-23

## Commission Engine
- File: control/src/modules/commissions/commission-engine.ts
- Test: control/src/modules/commissions/commission-engine.spec.ts
- Status: TEST VERIFIED

## Commission Base
- Per existing schema: commissionable_amount column in commission_ledger
- Commission rate: effective_commission_rate column
- Commission amount: commission_amount = commissionable_amount × effective_commission_rate

## Financial Arithmetic
- PostgreSQL NUMERIC type used for all financial columns (not FLOAT)
- NestJS uses decimal.js for calculations where needed
- No floating-point arithmetic for money in database schema

## Invoice Math
- base_price, discount, tax columns in invoices table
- total = base_price - discount + tax (computed in billing.service.ts)
- All amounts stored as NUMERIC

## Verification Status
- Commission engine: unit tests pass
- Invoice math: schema supports correct calculation
- Refund/chargeback: reversal columns exist (reversed_at, reversal_reason)
- Payout: available_amount calculated from ledger reconstruction

## External Blockers
- No real payment transactions to verify against
- No provider sandbox to test webhook → invoice → commission chain
