// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package transform // import "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/internal/transform"

import (
	"fmt"

	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	cpb "go.opentelemetry.io/proto/otlp/common/v1"
	mpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	rpb "go.opentelemetry.io/proto/otlp/resource/v1"
)

// ResourceMetrics returns an OTLP ResourceMetrics generated from rm. If rm
// contains invalid ScopeMetrics, an error will be returned along with an OTLP
// ResourceMetrics that contains partial OTLP ScopeMetrics.
func ResourceMetrics(rm metricdata.ResourceMetrics) (*mpb.ResourceMetrics, error) {
	sms, err := ScopeMetrics(rm.ScopeMetrics)
	return &mpb.ResourceMetrics{
		Resource: &rpb.Resource{
			Attributes: AttrIter(rm.Resource.Iter()),
		},
		ScopeMetrics: sms,
		SchemaUrl:    rm.Resource.SchemaURL(),
	}, err
}

// ScopeMetrics returns a slice of OTLP ScopeMetrics generated from sms. If
// sms contains invalid metric values, an error will be returned along with a
// slice that contains partial OTLP ScopeMetrics.
func ScopeMetrics(sms []metricdata.ScopeMetrics) ([]*mpb.ScopeMetrics, error) {
	errs := &multiErr{datatype: "ScopeMetrics"}
	out := make([]*mpb.ScopeMetrics, 0, len(sms))
	for _, sm := range sms {
		ms, err := Metrics(sm.Metrics)
		if err != nil {
			errs.append(err)
		}

		out = append(out, &mpb.ScopeMetrics{
			Scope: &cpb.InstrumentationScope{
				Name:    sm.Scope.Name,
				Version: sm.Scope.Version,
			},
			Metrics:   ms,
			SchemaUrl: sm.Scope.SchemaURL,
		})
	}
	return out, errs.errOrNil()
}

// Metrics returns a slice of OTLP Metric generated from ms. If ms contains
// invalid metric values, an error will be returned along with a slice that
// contains partial OTLP Metrics.
func Metrics(ms []metricdata.Metrics) ([]*mpb.Metric, error) {
	errs := &multiErr{datatype: "Metrics"}
	out := make([]*mpb.Metric, 0, len(ms))
	for _, m := range ms {
		o, err := metric(m)
		if err != nil {
			// Do not include invalid data. Drop the metric, report the error.
			errs.append(errMetric{m: o, err: err})
			continue
		}
		out = append(out, o)
	}
	return out, errs.errOrNil()
}

func metric(m metricdata.Metrics) (*mpb.Metric, error) {
	var err error
	out := &mpb.Metric{
		Name:        m.Name,
		Description: m.Description,
		Unit:        string(m.Unit),
	}
	switch a := m.Data.(type) {
	case metricdata.Gauge[int64]:
		out.Data = Gauge[int64](a)
	case metricdata.Gauge[float64]:
		out.Data = Gauge[float64](a)
	case metricdata.Sum[int64]:
		out.Data, err = Sum[int64](a)
	case metricdata.Sum[float64]:
		out.Data, err = Sum[float64](a)
	case metricdata.Histogram:
		out.Data, err = Histogram(a)
	default:
		return out, fmt.Errorf("%w: %T", errUnknownAggregation, a)
	}
	return out, err
}

// Gauge returns an OTLP Metric_Gauge generated from g.
func Gauge[N int64 | float64](g metricdata.Gauge[N]) *mpb.Metric_Gauge {
	return &mpb.Metric_Gauge{
		Gauge: &mpb.Gauge{
			DataPoints: DataPoints(g.DataPoints),
		},
	}
}

// Sum returns an OTLP Metric_Sum generated from s. An error is returned with
// a partial Metric_Sum if the temporality of s is unknown.
func Sum[N int64 | float64](s metricdata.Sum[N]) (*mpb.Metric_Sum, error) {
	t, err := Temporality(s.Temporality)
	if err != nil {
		return nil, err
	}
	return &mpb.Metric_Sum{
		Sum: &mpb.Sum{
			AggregationTemporality: t,
			IsMonotonic:            s.IsMonotonic,
			DataPoints:             DataPoints(s.DataPoints),
		},
	}, nil
}

// DataPoints returns a slice of OTLP NumberDataPoint generated from dPts.
func DataPoints[N int64 | float64](dPts []metricdata.DataPoint[N]) []*mpb.NumberDataPoint {
	out := make([]*mpb.NumberDataPoint, 0, len(dPts))
	for _, dPt := range dPts {
		ndp := &mpb.NumberDataPoint{
			Attributes:        AttrIter(dPt.Attributes.Iter()),
			StartTimeUnixNano: uint64(dPt.StartTime.UnixNano()),
			TimeUnixNano:      uint64(dPt.Time.UnixNano()),
		}
		switch v := any(dPt.Value).(type) {
		case int64:
			ndp.Value = &mpb.NumberDataPoint_AsInt{
				AsInt: v,
			}
		case float64:
			ndp.Value = &mpb.NumberDataPoint_AsDouble{
				AsDouble: v,
			}
		}
		out = append(out, ndp)
	}
	return out
}

// Histogram returns an OTLP Metric_Histogram generated from h. An error is
// returned with a partial Metric_Histogram if the temporality of h is
// unknown.
func Histogram(h metricdata.Histogram) (*mpb.Metric_Histogram, error) {
	t, err := Temporality(h.Temporality)
	if err != nil {
		return nil, err
	}
	return &mpb.Metric_Histogram{
		Histogram: &mpb.Histogram{
			AggregationTemporality: t,
			DataPoints:             HistogramDataPoints(h.DataPoints),
		},
	}, nil
}

// HistogramDataPoints returns a slice of OTLP HistogramDataPoint generated
// from dPts.
func HistogramDataPoints(dPts []metricdata.HistogramDataPoint) []*mpb.HistogramDataPoint {
	out := make([]*mpb.HistogramDataPoint, 0, len(dPts))
	for _, dPt := range dPts {
		sum := dPt.Sum
		hdp := &mpb.HistogramDataPoint{
			Attributes:        AttrIter(dPt.Attributes.Iter()),
			StartTimeUnixNano: uint64(dPt.StartTime.UnixNano()),
			TimeUnixNano:      uint64(dPt.Time.UnixNano()),
			Count:             dPt.Count,
			Sum:               &sum,
			BucketCounts:      dPt.BucketCounts,
			ExplicitBounds:    dPt.Bounds,
		}
		if v, ok := dPt.Min.Value(); ok {
			hdp.Min = &v
		}
		if v, ok := dPt.Max.Value(); ok {
			hdp.Max = &v
		}
		out = append(out, hdp)
	}
	return out
}

// Temporality returns an OTLP AggregationTemporality generated from t. If t
// is unknown, an error is returned along with the invalid
// AggregationTemporality_AGGREGATION_TEMPORALITY_UNSPECIFIED.
func Temporality(t metricdata.Temporality) (mpb.AggregationTemporality, error) {
	switch t {
	case metricdata.DeltaTemporality:
		return mpb.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA, nil
	case metricdata.CumulativeTemporality:
		return mpb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE, nil
	default:
		err := fmt.Errorf("%w: %s", errUnknownTemporality, t)
		return mpb.AggregationTemporality_AGGREGATION_TEMPORALITY_UNSPECIFIED, err
	}
}
