package repository

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
)

func ExportAll(s Storage) ([]model.Metrics, error) {
	m, err := s.GetAllMetrics()
	if err != nil {
		return nil, err
	}
	out := make([]model.Metrics, 0, len(m))

	for _, v := range m {
		out = append(out, v)
	}
	return out, nil
}

func SaveToFile(s Storage, path string) error {
	list, err := ExportAll(s)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil && !os.IsExist(err) {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err = enc.Encode(list); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func LoadFromFile(s Storage, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var list []model.Metrics
	dec := json.NewDecoder(f)
	if err = dec.Decode(&list); err != nil {
		return err
	}

	for _, m := range list {
		switch m.MType {
		case model.Gauge:
			if m.Value != nil {
				s.UpdateGauge(m.ID, *m.Value)
			}
		case model.Counter:
			if m.Delta != nil {
				s.UpdateCounter(m.ID, *m.Delta)
			}
		}
	}
	return nil
}
