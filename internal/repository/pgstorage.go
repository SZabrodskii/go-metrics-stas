package repository

import (
	"database/sql"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
)

type postgresStorage struct {
	db *sql.DB
}

func newPostgresStorage(db *sql.DB) *postgresStorage {
	return &postgresStorage{
		db: db,
	}
}

func (p *postgresStorage) UpdateGauge(id string, value float64) {
	_, _ = p.db.Exec(`INSERT INTO metrics (id,mtype,value,delta)
	                   VALUES ($1,'gauge',$2,NULL)
	                   ON CONFLICT (id,mtype)
	                   DO UPDATE SET value=EXCLUDED.value, updated_at=now()`, id, value)
}

func (p *postgresStorage) UpdateCounter(id string, delta int64) {
	_, _ = p.db.Exec(`INSERT INTO metrics (id,mtype,value,delta)
                    VALUES ($1,'counter',NULL,$2)
                    ON CONFLICT (id,mtype)
                    DO UPDATE SET delta=COALESCE(metrics.delta,0)+EXCLUDED.delta, updated_at=now()`, id, delta)
}

func (p *postgresStorage) GetGauge(id string) (float64, error) {
	var v sql.NullFloat64
	err := p.db.QueryRow(`SELECT value FROM metrics WHERE id = $1 AND mtype='gauge'`, id).Scan(&v)
	if err == sql.ErrNoRows || !v.Valid {
		return 0, ErrGaugeNotFound
	}
	return v.Float64, nil
}

func (p *postgresStorage) GetCounter(id string) (int64, error) {
	var v sql.NullInt64
	err := p.db.QueryRow(`SELECT delta FROM metrics WHERE id=$1 AND mtype='counter'`, id).Scan(&v)
	if err == sql.ErrNoRows || !v.Valid {
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
		var id, metricType string
		var value sql.NullFloat64
		var delta sql.NullInt64

		if err := rows.Scan(&id, &metricType, &value, &delta); err != nil {
			return nil, err
		}
		m := model.Metrics{ID: id, MType: metricType}
		if value.Valid {
			val := value.Float64
			m.Value = &val
		}
		if delta.Valid {
			del := delta.Int64
			m.Delta = &del
		}
		out[id] = m
	}
	return out, rows.Err()
}
