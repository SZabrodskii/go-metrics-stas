package repository

import (
	"database/sql"
	"errors"
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
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if strings.HasPrefix(pgErr.Code, "08") {
			return true
		}
		switch pgErr.Code {
		case pgerrcode.ConnectionException,
			pgerrcode.ConnectionDoesNotExist,
			pgerrcode.ConnectionFailure,
			pgerrcode.SQLClientUnableToEstablishSQLConnection,
			pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection,
			pgerrcode.TransactionResolutionUnknown,
			pgerrcode.ProtocolViolation:
			return true
		}
	}
	return false
}

func retryPG(fn func() error) error {
	var err error
	attempts := len(retrySchedule) + 1
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if !isPGConnException(err) || i == len(retrySchedule) {
			return err
		}
		time.Sleep(retrySchedule[i])
	}
	return err
}

func (p *postgresStorage) UpdateGauge(id string, value float64) {
	_ = retryPG(func() error {
		_, err := p.db.Exec(`INSERT INTO metrics (id,mtype,value,delta)
	                   VALUES ($1,'gauge',$2,NULL)
	                   ON CONFLICT (id,mtype)
	                   DO UPDATE SET value=EXCLUDED.value, updated_at=now()`, id, value)
		return err
	})
}

func (p *postgresStorage) UpdateCounter(id string, delta int64) {
	_ = retryPG(func() error {
		_, err := p.db.Exec(`INSERT INTO metrics (id,mtype,value,delta)
                    VALUES ($1,'counter',NULL,$2)
                    ON CONFLICT (id,mtype)
                    DO UPDATE SET delta=COALESCE(metrics.delta,0)+EXCLUDED.delta, updated_at=now()`, id, delta)
		return err
	})
}

func (p *postgresStorage) GetGauge(id string) (float64, error) {
	var v sql.NullFloat64
	err := retryPG(func() error {
		return p.db.QueryRow(`SELECT value FROM metrics WHERE id = $1 AND mtype='gauge'`, id).Scan(&v)
	})
	if errors.Is(err, sql.ErrNoRows) || !v.Valid {
		return 0, ErrGaugeNotFound
	}
	return v.Float64, nil
}

func (p *postgresStorage) GetCounter(id string) (int64, error) {
	var v sql.NullInt64
	err := retryPG(func() error {
		return p.db.QueryRow(`SELECT delta FROM metrics WHERE id=$1 AND mtype='counter'`, id).Scan(&v)
	})
	if errors.Is(err, sql.ErrNoRows) || !v.Valid {
		return 0, ErrCounterNotFound
	}
	return v.Int64, nil
}

func (p *postgresStorage) GetAllMetrics() (map[string]model.Metrics, error) {
	rows, err := p.db.Query(`SELECT id,mtype,value,delta FROM metrics`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]model.Metrics)

	for rows.Next() {
		var (
			id, metricType string
			value          sql.NullFloat64
			delta          sql.NullInt64
		)

		if err := rows.Scan(&id, &metricType, &value, &delta); err != nil {
			return nil, err
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

	if err := rows.Err(); err != nil {
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
			return err
		}
		defer tx.Rollback()

		stmtGauge, err := tx.Prepare(`
          INSERT INTO metrics (id,mtype,value,delta)
          VALUES ($1,'gauge',$2,NULL)
          ON CONFLICT (id,mtype)
          DO UPDATE SET value=EXCLUDED.value, updated_at=now()`)
		if err != nil {
			return err
		}
		defer stmtGauge.Close()

		stmtCounter, err := tx.Prepare(`
          INSERT INTO metrics (id,mtype,value,delta)
          VALUES ($1,'counter',NULL,$2)
          ON CONFLICT (id,mtype)
          DO UPDATE SET delta=COALESCE(metrics.delta,0)+EXCLUDED.delta, updated_at=now()`)
		if err != nil {
			return err
		}
		defer stmtCounter.Close()

		for _, m := range metrics {
			switch m.MType {
			case model.Gauge:
				if m.Value != nil {
					if _, e := stmtGauge.Exec(m.ID, *m.Value); e != nil {
						return e
					}
				}
			case model.Counter:
				if m.Delta != nil {
					if _, e := stmtCounter.Exec(m.ID, *m.Delta); e != nil {
						return e
					}
				}
			}
		}
		return tx.Commit()
	})
}
