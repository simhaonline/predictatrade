package backtest

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// DBCandleReader reads historical candles from the market.candles PostgreSQL table.
type DBCandleReader struct {
	dbURL string
}

// NewDBCandleReader creates a new database candle reader.
func NewDBCandleReader(dbURL string) *DBCandleReader {
	return &DBCandleReader{dbURL: dbURL}
}

// ReadCandles loads candles for a specific symbol+timeframe within the date range.
// Returns candles sorted by time ascending.
func (r *DBCandleReader) ReadCandles(ctx context.Context, symbol string, tf types.Timeframe, start, end time.Time) ([]*types.Candle, error) {
	// Parse connection using pgx
	conn, err := r.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("db connect: %w", err)
	}
	defer conn.Close(ctx)

	query := `
		SELECT time, open, high, low, close, volume, source, is_closed
		FROM market.candles
		WHERE symbol = $1 AND timeframe = $2 AND time >= $3 AND time <= $4
		ORDER BY time ASC`

	rows, err := conn.Query(ctx, query, symbol, string(tf), start, end)
	if err != nil {
		return nil, fmt.Errorf("query candles: %w", err)
	}
	defer rows.Close()

	var candles []*types.Candle
	for rows.Next() {
		var t time.Time
		var o, h, l, c float64
		var v int64
		var source string
		var isClosed bool

		if err := rows.Scan(&t, &o, &h, &l, &c, &v, &source, &isClosed); err != nil {
			return nil, fmt.Errorf("scan candle: %w", err)
		}

		candles = append(candles, &types.Candle{
			Symbol:    symbol,
			Timeframe: tf,
			Time:      t.UTC(),
			Open:      decimal.NewFromFloat(o),
			High:      decimal.NewFromFloat(h),
			Low:       decimal.NewFromFloat(l),
			Close:     decimal.NewFromFloat(c),
			Volume:    v,
			Source:    source,
			Quality:   types.CandleComplete,
			IsClosed:  isClosed,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	sort.Slice(candles, func(i, j int) bool {
		return candles[i].Time.Before(candles[j].Time)
	})

	return candles, nil
}

// ReadAllTimeframes loads candles for the primary timeframe and all higher timeframes.
// Returns the primary candles and a map of higher TF candles.
func (r *DBCandleReader) ReadAllTimeframes(ctx context.Context, symbol string, primaryTF types.Timeframe, higherTFs []types.Timeframe, start, end time.Time) ([]*types.Candle, map[types.Timeframe][]*types.Candle, error) {
	// Read primary timeframe — extend start backward for indicator warmup
	warmupStart := start.AddDate(0, 0, -30) // 30 days warmup for EMA200 etc.

	primary, err := r.ReadCandles(ctx, symbol, primaryTF, warmupStart, end)
	if err != nil {
		return nil, nil, fmt.Errorf("read primary %s: %w", primaryTF, err)
	}

	higher := make(map[types.Timeframe][]*types.Candle)
	for _, tf := range higherTFs {
		candles, err := r.ReadCandles(ctx, symbol, tf, warmupStart, end)
		if err != nil {
			return nil, nil, fmt.Errorf("read higher %s: %w", tf, err)
		}
		higher[tf] = candles
	}

	return primary, higher, nil
}

// connect opens a pgx connection to the database.
func (r *DBCandleReader) connect(ctx context.Context) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, r.dbURL)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
