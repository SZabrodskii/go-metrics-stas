package repository

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type postgresStorage struct {
	db *sql.DB
}

func newPostgresStorage(db *sql.DB) *postgresStorage {
	return &postgresStorage{
		db: db,
	}
}

var retrySchedule = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

func isPGConnException(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, driver.ErrBadConn) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		code := string(pgErr.Code)
		if strings.HasPrefix(code, "08") {
			return true
		}
		if strings.HasPrefix(code, "40") {
			return true
		}
		switch code {
		case pgerrcode.ConnectionException,
			pgerrcode.ConnectionDoesNotExist,
			pgerrcode.ConnectionFailure,
			pgerrcode.SQLClientUnableToEstablishSQLConnection,
			pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection,
			pgerrcode.TransactionResolutionUnknown,
			pgerrcode.ProtocolViolation:
			return true

		case pgerrcode.TransactionRollback,
			pgerrcode.SerializationFailure,
			pgerrcode.DeadlockDetected:
			return true

		case pgerrcode.CannotConnectNow:
			return true
		}
	}
	return false
}

func retryPG(fn func() error) error {
	var lastErr error
	attempts := len(retrySchedule) + 1

	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if !isPGConnException(err) || i == len(retrySchedule) {
				return err
			}
			t := time.NewTimer(retrySchedule[i])
			<-t.C
			t.Stop()
		}
	}
	return lastErr
}

func (p *postgresStorage) UpdateGauge(id string, value float64) {
	_ = retryPG(func() error {
		_, err := p.db.Exec(`INSERT INTO metrics (id,mtype,value,delta)
			VALUES ($1,'gauge',$2,NULL)
			ON CONFLICT (id,mtype)
			DO UPDATE SET value=EXCLUDED.value, updated_at=now()`, id, value)
		if err != nil {
			return fmt.Errorf("update gauge exec: %w", err)
		}
		return nil
	})
}

func (p *postgresStorage) UpdateCounter(id string, delta int64) {
	_ = retryPG(func() error {
		_, err := p.db.Exec(`INSERT INTO metrics (id,mtype,value,delta)
			VALUES ($1,'counter',NULL,$2)
			ON CONFLICT (id,mtype)
			DO UPDATE SET delta=COALESCE(metrics.delta,0)+EXCLUDED.delta, updated_at=now()`, id, delta)
		if err != nil {
			return fmt.Errorf("update counter exec: %w", err)
		}
		return nil
	})
}

func (p *postgresStorage) GetGauge(id string) (float64, error) {
	var v sql.NullFloat64
	err := retryPG(func() error {
		return p.db.QueryRow(`SELECT value FROM metrics WHERE id = $1 AND mtype='gauge'`, id).Scan(&v)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrGaugeNotFound
		}
		return 0, err
	}
	if !v.Valid {
		return 0, ErrGaugeNotFound
	}
	return v.Float64, nil
}

func (p *postgresStorage) GetCounter(id string) (int64, error) {
	var v sql.NullInt64
	err := retryPG(func() error {
		return p.db.QueryRow(`SELECT delta FROM metrics WHERE id=$1 AND mtype='counter'`, id).Scan(&v)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrCounterNotFound
		}
		return 0, err
	}
	if !v.Valid {
		return 0, ErrCounterNotFound
	}
	return v.Int64, nil
}

func (p *postgresStorage) queryAllMetrics(handle func(*sql.Rows) error) error {
	var rows *sql.Rows

	if err := retryPG(func() error {
		var e error
		rows, e = p.db.Query(`SELECT id, mtype, value, delta FROM metrics`)
		return e
	}); err != nil {
		return fmt.Errorf("query all metrics: %w", err)
	}

	hErr := handle(rows)
	iterErr := rows.Err()
	if iterErr != nil {
		iterErr = fmt.Errorf("rows iteration: %w", iterErr)
	}

	if hErr != nil && iterErr != nil {
		return fmt.Errorf("%v; %w", hErr, iterErr)
	}
	if hErr != nil {
		return hErr
	}
	if iterErr != nil {
		return iterErr
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && iterErr == nil && hErr == nil {
			iterErr = fmt.Errorf("rows close: %w", cerr)
		}
	}()
	return nil
}

func (p *postgresStorage) GetAllMetrics() (map[string]model.Metrics, error) {

	out := make(map[string]model.Metrics)
	err := p.queryAllMetrics(func(rows *sql.Rows) error {
		for rows.Next() {
			var (
				id, metricType string
				value          sql.NullFloat64
				delta          sql.NullInt64
			)
			if err := rows.Scan(&id, &metricType, &value, &delta); err != nil {
				return fmt.Errorf("failed to scan row: %w", err)
			}
			m := model.Metrics{ID: id, MType: metricType}
			if value.Valid {
				v := value.Float64
				m.Value = &v
			}
			if delta.Valid {
				d := delta.Int64
				m.Delta = &d
			}
			out[id] = m
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (p *postgresStorage) UpdateBatch(metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	return retryPG(func() error {
		var tx *sql.Tx
		var err error

		tx, err = p.db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback()

		stmtGauge, err := tx.Prepare(`
          INSERT INTO metrics (id,mtype,value,delta)
          VALUES ($1,'gauge',$2,NULL)
          ON CONFLICT (id,mtype)
          DO UPDATE SET value=EXCLUDED.value, updated_at=now()`)
		if err != nil {
			return fmt.Errorf("prepare gauge stmt: %w", err)
		}
		defer stmtGauge.Close()

		stmtCounter, err := tx.Prepare(`
          INSERT INTO metrics (id,mtype,value,delta)
          VALUES ($1,'counter',NULL,$2)
          ON CONFLICT (id,mtype)
          DO UPDATE SET delta=COALESCE(metrics.delta,0)+EXCLUDED.delta, updated_at=now()`)
		if err != nil {
			return fmt.Errorf("prepare counter stmt: %w", err)
		}
		defer stmtCounter.Close()

		for _, m := range metrics {
			switch m.MType {
			case model.Gauge:
				if m.Value != nil {
					if _, e := stmtGauge.Exec(m.ID, *m.Value); e != nil {
						return fmt.Errorf("exec gauge id=%q: %w", m.ID, e)
					}
				}
			case model.Counter:
				if m.Delta != nil {
					if _, e := stmtCounter.Exec(m.ID, *m.Delta); e != nil {
						return fmt.Errorf("exec counter id=%q: %w", m.ID, e)
					}
				}
			}
		}
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("commit tx: %w", err)
		}
		return nil
	})
}
