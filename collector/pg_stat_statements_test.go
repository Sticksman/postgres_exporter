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
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/smartystreets/goconvey/convey"
)

func TestPGStateStatementsCollector(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error opening a stub db connection: %s", err)
	}
	defer db.Close()

	columns := []string{"user", "datname", "queryid", "calls_total", "seconds_total", "rows_total", "block_read_seconds_total", "block_write_seconds_total"}
	rows := sqlmock.NewRows(columns).
		AddRow("postgres", "postgres", 1500, 5, 0.4, 100, 0.1, 0.2)
	mock.ExpectQuery(sanitizeQuery(pgStatStatementsQuery)).WillReturnRows(rows)

	ch := make(chan prometheus.Metric)
	go func() {
		defer close(ch)
		c := PGStatStatementsCollector{}

		if err := c.Update(context.Background(), db, ch); err != nil {
			t.Errorf("Error calling PGStatStatementsCollector.Update: %s", err)
		}
	}()

	expected := []MetricResult{
		{labels: labelMap{"user": "postgres", "datname": "postgres", "queryid": "1500"}, metricType: dto.MetricType_COUNTER, value: 5},
		{labels: labelMap{"user": "postgres", "datname": "postgres", "queryid": "1500"}, metricType: dto.MetricType_COUNTER, value: 0.4},
		{labels: labelMap{"user": "postgres", "datname": "postgres", "queryid": "1500"}, metricType: dto.MetricType_COUNTER, value: 100},
		{labels: labelMap{"user": "postgres", "datname": "postgres", "queryid": "1500"}, metricType: dto.MetricType_COUNTER, value: 0.1},
		{labels: labelMap{"user": "postgres", "datname": "postgres", "queryid": "1500"}, metricType: dto.MetricType_COUNTER, value: 0.2},
	}

	convey.Convey("Metrics comparison", t, func() {
		for _, expect := range expected {
			m := readMetric(<-ch)
			convey.So(expect, convey.ShouldResemble, m)
		}
	})
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled exceptions: %s", err)
	}
}
