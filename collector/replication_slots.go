// Copyright 2023 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"context"
	"database/sql"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerCollector("replication_slot", defaultEnabled, NewPGReplicationSlotCollector)
}

type PGReplicationSlotCollector struct {
	log log.Logger
}

func NewPGReplicationSlotCollector(config collectorConfig) (Collector, error) {
	return &PGReplicationSlotCollector{log: config.logger}, nil
}

var pgReplicationSlotCurrentWalLSN = prometheus.NewDesc(
	"pg_replication_slot_current_wal_lsn",
	"current wal lsn value",
	[]string{"slot_name"}, nil,
)

var pgReplicationSlotConfirmedFlushLSN = prometheus.NewDesc(
	"pg_replication_slot_confirmed_flush_lsn",
	"last lsn confirmed flushed to the replication slot",
	[]string{"slot_name"}, nil,
)

var pgReplicationSlotActive = prometheus.NewDesc(
	"pg_replication_slot_is_active",
	"whether the replication slot is active or not",
	[]string{"slot_name"}, nil,
)

var pgReplicationSlotQuery = `SELECT
	slot_name,
	pg_current_wal_lsn() - '0/0' AS current_wal_lsn,
	coalesce(confirmed_flush_lsn, '0/0') - '0/0',
	active
FROM
	pg_replication_slots;
`

func (PGReplicationSlotCollector) Update(ctx context.Context, db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.QueryContext(ctx,
		pgReplicationSlotQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var slot_name string
		var wal_lsn int64
		var flush_lsn int64
		var is_active bool
		var is_active_value int

		if err := rows.Scan(&slot_name, &wal_lsn, &flush_lsn, &is_active); err != nil {
			return err
		}

		if is_active {
			is_active_value = 1
		} else {
			is_active_value = 0
		}

		ch <- prometheus.MustNewConstMetric(
			pgReplicationSlotCurrentWalLSN,
			prometheus.GaugeValue, float64(wal_lsn), slot_name,
		)
		if is_active {
			ch <- prometheus.MustNewConstMetric(
				pgReplicationSlotConfirmedFlushLSN,
				prometheus.GaugeValue, float64(flush_lsn), slot_name,
			)
		}
		ch <- prometheus.MustNewConstMetric(
			pgReplicationSlotActive,
			prometheus.GaugeValue, float64(is_active_value), slot_name,
		)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}
