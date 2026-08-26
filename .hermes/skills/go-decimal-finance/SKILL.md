---
name: go-decimal-finance
description: "Use shopspring/decimal for exact financial math in Go."
---

# go-decimal-finance

Use when handling money values (prices, P&L, commissions, margins, lot sizes) in Predict-A-Trade Go code.

## Rule
All financial values use shopspring/decimal. Never float64 for money.

## Common Operations
price := decimal.NewFromFloat(2650.45)
lot := decimal.NewFromFloat(0.01)
margin := price.Mul(lot).Mul(decimal.NewFromInt(100)).Div(decimal.NewFromInt(500))

## Serialization
JSON: shopspring/decimal marshals to string by default
DB (pgx): NUMERIC(38,18) in PostgreSQL = decimal.Decimal via pgtype
Always use string transport for JSON APIs, never float

## Comparison
a.Equal(b), a.GreaterThan(b), a.LessThanOrEqual(b)

## Rounding
price.Round(2) — 2 decimal places for cents

## Pitfalls
Trailing zeros differ in string form, use .Equal() not ==
Division: always specify precision with .DivRound(p, 12)
Never convert decimal to float64 for computation
