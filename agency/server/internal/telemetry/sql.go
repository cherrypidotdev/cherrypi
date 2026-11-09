// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package telemetry

import (
	"database/sql"
	"database/sql/driver"
	"net/url"
	"os"
	"strings"

	"github.com/XSAM/otelsql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func SQLOpen(
	driverName, dsn string,
	dbSystem attribute.KeyValue,
	extra ...attribute.KeyValue,
) (*sql.DB, error) {
	if !Enabled() {
		return sql.Open(driverName, dsn)
	}

	return otelsql.Open(driverName, dsn, sqlOptions(dsn, dbSystem, extra)...)
}

func SQLOpenConnector(
	connector driver.Connector,
	dsn string,
	dbSystem attribute.KeyValue,
	extra ...attribute.KeyValue,
) *sql.DB {
	if !Enabled() {
		return sql.OpenDB(connector)
	}

	return otelsql.OpenDB(connector, sqlOptions(dsn, dbSystem, extra)...)
}

func RegisterDBStats(
	db *sql.DB,
	dsn string,
	dbSystem attribute.KeyValue,
	extra ...attribute.KeyValue,
) {
	if !Enabled() {
		return
	}

	_, _ = otelsql.RegisterDBStatsMetrics(db, sqlOptions(dsn, dbSystem, extra)...)
}

func SQLPoolAttribute(pool string) attribute.KeyValue {
	return attribute.String("cherrypi.db.pool", pool)
}

func sqlOptions(
	dsn string,
	dbSystem attribute.KeyValue,
	extra []attribute.KeyValue,
) []otelsql.Option {
	attributes := make([]attribute.KeyValue, 0, len(extra)+2)
	attributes = append(attributes, dbSystem)
	if namespace := dbNamespace(dsn, dbSystem); namespace != "" {
		attributes = append(attributes, semconv.DBNamespace(namespace))
	}
	attributes = append(attributes, extra...)

	return []otelsql.Option{
		otelsql.WithTracerProvider(otel.GetTracerProvider()),
		otelsql.WithMeterProvider(otel.GetMeterProvider()),
		otelsql.WithAttributes(attributes...),

		otelsql.WithSpanOptions(otelsql.SpanOptions{
			OmitConnResetSession: true,
			RowsChildOfQuery:     true,
		}),
	}
}

func dbNamespace(dsn string, dbSystem attribute.KeyValue) string {
	if dbSystem == semconv.DBSystemNamePostgreSQL {
		return postgresNamespace(dsn)
	}

	path := dsn
	if index := strings.IndexRune(path, '?'); index >= 0 {
		path = path[:index]
	}
	path = strings.TrimPrefix(path, "file:")
	if path == "" || strings.HasPrefix(path, ":memory:") || path == "memory" {
		return "memory"
	}

	return path
}

func postgresNamespace(dsn string) string {
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		if name := strings.TrimPrefix(u.Path, "/"); name != "" {
			return name
		}
	}

	for _, field := range strings.Fields(dsn) {
		for _, key := range []string{"dbname=", "database="} {
			if name, ok := strings.CutPrefix(field, key); ok {
				return name
			}
		}
	}

	if name := strings.TrimSpace(os.Getenv("PGDATABASE")); name != "" {
		return name
	}

	return "postgres"
}
