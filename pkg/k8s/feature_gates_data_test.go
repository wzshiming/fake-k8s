package k8s

import (
	"fmt"
	"testing"
)

func TestRawData(t *testing.T) {
	for i, data := range rawData {
		if err := data.Verification(); err != nil {
			t.Error(data, err)
		}

		if data.Until >= 0 && i+1 < len(rawData) {
			nextData := rawData[i+1]
			if data.Name == nextData.Name {
				if data.Until+1 != nextData.Since {
					t.Error(data, fmt.Errorf("invalid until: %d + 1 != next since: %d", data.Until, data.Since))
				}
			}
		}
	}
}
