package superpixel_utils

import (
	"reflect"
	"testing"

	"github.com/segtile/segtile/pkg/superpixel_map"
)

func TestEnforceConnectivity_AlreadyConnected(t *testing.T) {
	labelMap := &superpixel_map.LabelMap{
		Width:  3,
		Height: 2,
		Labels: []uint32{
			1, 1, 2,
			1, 2, 2,
		},
	}
	originalLabels := append([]uint32(nil), labelMap.Labels...)

	if changed := EnforceConnectivity(labelMap); changed {
		t.Fatal("EnforceConnectivity changed an already connected label map")
	}
	if !reflect.DeepEqual(labelMap.Labels, originalLabels) {
		t.Fatalf("label map changed: got %v, want %v", labelMap.Labels, originalLabels)
	}
}
