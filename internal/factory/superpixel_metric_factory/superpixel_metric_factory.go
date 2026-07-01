package superpixel_metric_factory

import (
	"fmt"

	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/average_boundary_distance"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/density_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/f1_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/labeling_calibration_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/regularity_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/weighted_f1_score"
)

const (
	// Algorithm names
	nameDensityScore             = "density_score"
	nameAverageBoundaryDistance  = "average_boundary_distance"
	nameRegularityScore          = "regularity_score"
	nameLabelingCalibrationScore = "labeling_calibration_score"
	nameWeightedF1Score          = "weighted_f1_score"
	nameF1Score                  = "f1_score"
)

func NewSuperpixelMetricByName(name string) (superpixel_metrics.SpMetric, error) {
	switch name {
	case nameDensityScore:
		return &density_score.SpMetricDensityScore{}, nil
	case nameAverageBoundaryDistance:
		return &average_boundary_distance.SpMetricAverageBoundaryDistance{}, nil
	case nameRegularityScore:
		return &regularity_score.SpMetricRegularityScore{}, nil
	case nameLabelingCalibrationScore:
		return &labeling_calibration_score.SpMetricLabelingCalibrationScore{}, nil
	case nameWeightedF1Score:
		return &weighted_f1_score.SpMetricWeightedF1Score{}, nil
	case nameF1Score:
		return &f1_score.SpMetricF1Score{}, nil
	default:
		return nil, fmt.Errorf("Unknown algorithm name: %s", name)
	}
}

func GetMetricName(impl superpixel_metrics.SpMetric) (string, error) {
	switch impl.(type) {
	case *density_score.SpMetricDensityScore:
		return nameDensityScore, nil
	case *average_boundary_distance.SpMetricAverageBoundaryDistance:
		return nameAverageBoundaryDistance, nil
	case *regularity_score.SpMetricRegularityScore:
		return nameRegularityScore, nil
	case *labeling_calibration_score.SpMetricLabelingCalibrationScore:
		return nameLabelingCalibrationScore, nil
	case *weighted_f1_score.SpMetricWeightedF1Score:
		return nameWeightedF1Score, nil
	case *f1_score.SpMetricF1Score:
		return nameF1Score, nil
	default:
		return "unknown", fmt.Errorf("Unknown implementation: %T", impl)
	}
}
